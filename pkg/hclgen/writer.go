package hclgen

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gcp-tf-forge/pkg/graph"
	"gcp-tf-forge/pkg/matcher"
)

// Generator produces clean, domain-separated Terraform HCL and *.auto.tfvars files.
type Generator struct {
	OutputDir string
	ProjectID string
	Layout    string // "root-composition" or "independent-stacks"
	Sanitizer *Sanitizer
}

// NewGenerator creates a new code generator instance.
func NewGenerator(outputDir, projectID, layout string) *Generator {
	if layout == "" {
		layout = "root-composition"
	}
	return &Generator{
		OutputDir: outputDir,
		ProjectID: projectID,
		Layout:    layout,
		Sanitizer: NewSanitizer(),
	}
}

// Generate writes the complete modular codebase to the output directory.
func (g *Generator) Generate(matchResult *matcher.MatchResult) error {
	if err := os.MkdirAll(g.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed creating output directory: %w", err)
	}

	// 1. Generate versions.tf and providers.tf
	if err := g.writeVersionsAndProviders(); err != nil {
		return err
	}

	// 2. Group resolved modules and raw resources by domain
	modulesByDomain := make(map[string][]*matcher.ResolvedModuleCall)
	for _, mod := range matchResult.ResolvedModules {
		modulesByDomain[mod.Domain] = append(modulesByDomain[mod.Domain], mod)
	}

	rawByDomain := make(map[string][]*graph.Node)
	for _, node := range matchResult.RawResources {
		domain := node.Asset.Domain()
		rawByDomain[domain] = append(rawByDomain[domain], node)
	}

	// 3. Collect all active domains
	domainSet := make(map[string]bool)
	for d := range modulesByDomain {
		domainSet[d] = true
	}
	for d := range rawByDomain {
		domainSet[d] = true
	}

	// 4. Generate domain files (.tf, .auto.tfvars, _imports.tf)
	for domain := range domainSet {
		if err := g.writeDomainFiles(domain, modulesByDomain[domain], rawByDomain[domain]); err != nil {
			return err
		}
	}

	// 5. Generate external Shared VPC data sources
	if len(matchResult.ExternalData) > 0 {
		if err := g.writeExternalDataSources(matchResult.ExternalData); err != nil {
			return err
		}
	}

	// 6. Generate root variables.tf
	if err := g.writeRootVariables(); err != nil {
		return err
	}

	return nil
}

func (g *Generator) writeVersionsAndProviders() error {
	versionsHCL := `# Terraform and Provider Version Constraints
terraform {
  required_version = ">= 1.5.0"
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = ">= 5.0.0"
    }
    google-beta = {
      source  = "hashicorp/google-beta"
      version = ">= 5.0.0"
    }
  }
}
`
	if err := os.WriteFile(filepath.Join(g.OutputDir, "versions.tf"), []byte(versionsHCL), 0644); err != nil {
		return err
	}

	providersHCL := fmt.Sprintf(`# Google Cloud Provider Configuration
provider "google" {
  project = var.project_id
}

provider "google-beta" {
  project = var.project_id
}
`)
	return os.WriteFile(filepath.Join(g.OutputDir, "providers.tf"), []byte(providersHCL), 0644)
}

func (g *Generator) writeDomainFiles(domain string, modules []*matcher.ResolvedModuleCall, rawNodes []*graph.Node) error {
	var (
		tfCode      strings.Builder
		importsCode strings.Builder
		tfvarsData  = make(map[string]interface{})
	)

	tfCode.WriteString(fmt.Sprintf("# Domain: %s\n\n", strings.ToUpper(domain)))
	importsCode.WriteString(fmt.Sprintf("# Transitional Import Blocks for Domain: %s\n# Auto-pruned by 'gcp-tf-forge cleanup' post-import.\n\n", domain))

	// 1. Emit Modules
	for _, mod := range modules {
		modVarName := fmt.Sprintf("%s_%s_config", domain, mod.ModuleName)
		tfvarsData[modVarName] = mod.Inputs

		tfCode.WriteString(fmt.Sprintf("module \"%s\" {\n", mod.ModuleName))
		tfCode.WriteString(fmt.Sprintf("  source  = \"%s\"\n", mod.Source))
		if mod.Version != "" {
			tfCode.WriteString(fmt.Sprintf("  version = \"%s\"\n", mod.Version))
		}
		tfCode.WriteString("  project_id = var.project_id\n\n")

		// Pass input variables
		for inputKey := range mod.Inputs {
			tfCode.WriteString(fmt.Sprintf("  %-25s = var.%s.%s\n", inputKey, modVarName, inputKey))
		}
		tfCode.WriteString("}\n\n")

		// Import blocks for module resources
		for _, imp := range mod.ImportTargets {
			importsCode.WriteString("import {\n")
			importsCode.WriteString(fmt.Sprintf("  to = %s\n", imp.ToAddress))
			importsCode.WriteString(fmt.Sprintf("  id = \"%s\"\n", imp.CloudID))
			importsCode.WriteString("}\n\n")
		}
	}

	// 2. Emit Raw Resources (Fallback)
	for _, node := range rawNodes {
		resType := node.Asset.TerraformResourceType()
		if resType == "" {
			continue
		}

		resName := g.Sanitizer.Sanitize(node.Asset.DisplayName)
		cleanAttrs := FilterAttributes(node.Asset.ResourceData)
		tfvarsData[fmt.Sprintf("raw_%s_%s", domain, resName)] = cleanAttrs

		tfCode.WriteString(fmt.Sprintf("resource \"%s\" \"%s\" {\n", resType, resName))
		tfCode.WriteString(fmt.Sprintf("  name    = \"%s\"\n", node.Asset.DisplayName))
		tfCode.WriteString("  project = var.project_id\n")

		// Inject lifecycle shields
		if shield := InjectsLifecycleShields(resType); shield != "" {
			tfCode.WriteString(shield)
		}
		tfCode.WriteString("}\n\n")

		// Import block for raw resource
		importsCode.WriteString("import {\n")
		importsCode.WriteString(fmt.Sprintf("  to = %s.%s\n", resType, resName))
		importsCode.WriteString(fmt.Sprintf("  id = \"%s\"\n", node.Asset.Name))
		importsCode.WriteString("}\n\n")
	}

	// Write files
	tfFileName := filepath.Join(g.OutputDir, fmt.Sprintf("%s.tf", domain))
	importsFileName := filepath.Join(g.OutputDir, fmt.Sprintf("%s_imports.tf", domain))
	tfvarsFileName := filepath.Join(g.OutputDir, fmt.Sprintf("%s.auto.tfvars.json", domain))

	if err := os.WriteFile(tfFileName, []byte(tfCode.String()), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(importsFileName, []byte(importsCode.String()), 0644); err != nil {
		return err
	}

	// Serialize .auto.tfvars.json
	tfvarsJSON, err := json.MarshalIndent(tfvarsData, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(tfvarsFileName, tfvarsJSON, 0644)
}

func (g *Generator) writeExternalDataSources(externalNodes []*graph.Node) error {
	var b strings.Builder
	b.WriteString("# Shared VPC & External Cross-Project Data Sources\n\n")

	for _, node := range externalNodes {
		resType := node.Asset.TerraformResourceType()
		if resType == "" {
			continue
		}

		dataName := g.Sanitizer.Sanitize(fmt.Sprintf("shared_%s", node.Asset.DisplayName))

		b.WriteString(fmt.Sprintf("data \"%s\" \"%s\" {\n", resType, dataName))
		b.WriteString(fmt.Sprintf("  name    = \"%s\"\n", node.Asset.DisplayName))
		if node.Asset.ProjectID != "" {
			b.WriteString(fmt.Sprintf("  project = \"%s\"\n", node.Asset.ProjectID))
		}
		if node.Asset.Location != "" && node.Asset.Location != "global" {
			b.WriteString(fmt.Sprintf("  region  = \"%s\"\n", node.Asset.Location))
		}
		b.WriteString("}\n\n")
	}

	return os.WriteFile(filepath.Join(g.OutputDir, "external_data.tf"), []byte(b.String()), 0644)
}

func (g *Generator) writeRootVariables() error {
	hcl := fmt.Sprintf(`variable "project_id" {
  type        = string
  description = "The target Google Cloud Project ID"
  default     = "%s"
}
`, g.ProjectID)

	return os.WriteFile(filepath.Join(g.OutputDir, "variables.tf"), []byte(hcl), 0644)
}
