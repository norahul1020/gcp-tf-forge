package main

import (
	"context"
	"fmt"

	"gcp-tf-forge/pkg/brownfield"
	"gcp-tf-forge/pkg/compliance"
	"gcp-tf-forge/pkg/discovery"
	"gcp-tf-forge/pkg/graph"
	"gcp-tf-forge/pkg/hclgen"
	"gcp-tf-forge/pkg/hierarchy"
	"gcp-tf-forge/pkg/matcher"

	"github.com/spf13/cobra"
)

var (
	bundleNetwork string
	importDefaults bool
	existingState string
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export GCP infrastructure into domain-separated Terraform modules and *.auto.tfvars",
	Long: `Export performs full discovery, module reconciliation, and HCL code emission.
It generates domain-separated configuration with native *.auto.tfvars files,
database password lifecycle shields, API destruction shields, and Terraform 1.5+
import blocks.`,
	RunE: runExport,
}

func init() {
	exportCmd.Flags().StringVar(&bundleNetwork, "bundle-network", "all", "Network bundling: 'all' (VPC+Subnets+Firewalls), 'subnets-only', or 'none'")
	exportCmd.Flags().BoolVar(&importDefaults, "import-defaults", false, "Include default VPC and auto-created system service accounts")
	exportCmd.Flags().StringVar(&existingState, "existing-state", "", "Path or GCS URI to existing .tfstate to skip already-managed assets")
	rootCmd.AddCommand(exportCmd)
}

func runExport(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	if projectID == "" {
		return fmt.Errorf("project ID is required. Use --project or -p flag")
	}

	fmt.Printf("🚀 Starting export for GCP project: %s\n", projectID)
	fmt.Printf("📂 Target Directory: %s\n", outputDir)
	fmt.Printf("📐 Layout Mode: %s\n", layout)
	fmt.Printf("🔗 Network Bundling: %s\n\n", bundleNetwork)

	// Phase 1: Ancestry & Discovery
	lineage, _ := hierarchy.ResolveAncestry(ctx, projectID)
	var assets []*discovery.Asset
	var err error

	if lineage != nil && lineage.CAIAvailable {
		fmt.Println("==> Phase 1: Ingesting assets via Cloud Asset Inventory...")
		scanner := discovery.NewCAIScanner(projectID)
		assets, err = scanner.Scan(ctx)
	}

	if err != nil || len(assets) == 0 {
		fmt.Println("==> Phase 1: Ingesting assets via Direct API Crawler...")
		crawler := discovery.NewDirectCrawler(projectID)
		assets, _ = crawler.Crawl(ctx)
	}

	fmt.Printf("✔ Discovered %d cloud assets.\n", len(assets))

	// Phase 2: State Diffing (if existing state provided)
	if existingState != "" {
		fmt.Printf("==> Phase 2: Diffing against existing state: %s\n", existingState)
		diff, err := brownfield.IngestAndDiff(existingState, assets)
		if err == nil {
			_ = brownfield.GenerateDriftReport(outputDir, projectID, diff)
			fmt.Printf("✔ Identified %d already-managed assets (skipping duplicate import blocks).\n", len(diff.AlreadyManaged))
		}
	}

	// Phase 3: Graph Resolution
	fmt.Println("==> Phase 3: Resolving topology graph & Shared VPC data sources...")
	g := graph.NewGraph(projectID)
	g.Build(assets)

	// Phase 4: Module Matching
	fmt.Println("==> Phase 4: Matching custom & official Google modules...")
	engine := matcher.NewEngine(nil, bundleNetwork)
	matchResult, err := engine.Reconcile(g)
	if err != nil {
		return fmt.Errorf("module reconciliation failed: %w", err)
	}

	// Phase 5: HCL Code Generation
	fmt.Println("==> Phase 5: Emitting HCL2 files, domain *.auto.tfvars, and Terraform 1.5+ import blocks...")
	gen := hclgen.NewGenerator(outputDir, projectID, layout)
	if err := gen.Generate(matchResult); err != nil {
		return fmt.Errorf("code generation failed: %w", err)
	}

	// Phase 6: Compliance Evaluation
	fmt.Println("==> Phase 6: Evaluating CIS GCP Foundations Compliance...")
	eval := compliance.NewEvaluator()
	eval.Evaluate(assets)
	_ = eval.GenerateReport(outputDir, projectID)

	fmt.Printf("\n✨ Successfully generated Terraform codebase in: %s\n", outputDir)
	fmt.Println("\nNext Steps:")
	fmt.Println("  1. cd " + outputDir)
	fmt.Println("  2. terraform init")
	fmt.Println("  3. gcp-tf-forge verify --dir=" + outputDir)
	fmt.Println("  4. terraform apply")
	fmt.Println("  5. gcp-tf-forge cleanup --dir=" + outputDir)
	return nil
}
