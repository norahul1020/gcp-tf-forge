package matcher

import (
	"fmt"
	"strings"

	"gcp-tf-forge/pkg/graph"
)

// Engine reconciles graph nodes into high-level Terraform modules or raw resources.
type Engine struct {
	CustomSpecs   []*ModuleSpec
	OfficialSpecs []*ModuleSpec
	BundleNetwork string // "all", "subnets-only", "none"
}

// NewEngine initializes the matcher engine with official specs and options.
func NewEngine(customSpecs []*ModuleSpec, bundleNetwork string) *Engine {
	if bundleNetwork == "" {
		bundleNetwork = "all"
	}
	return &Engine{
		CustomSpecs:   customSpecs,
		OfficialSpecs: BuiltInOfficialModules(),
		BundleNetwork: bundleNetwork,
	}
}

// MatchResult contains the resolved module invocations and unmapped raw resources.
type MatchResult struct {
	ResolvedModules []*ResolvedModuleCall
	RawResources    []*graph.Node
	ExternalData    []*graph.Node
}

// Reconcile evaluates all non-dynamic graph nodes and groups them into modules or raw resources.
func (e *Engine) Reconcile(g *graph.Graph) (*MatchResult, error) {
	result := &MatchResult{
		ResolvedModules: make([]*ResolvedModuleCall, 0),
		RawResources:    make([]*graph.Node, 0),
		ExternalData:    make([]*graph.Node, 0),
	}

	consumedNodes := make(map[string]bool)

	// Step 1: Collect external data sources (Shared VPC subnets, external kms keys, etc.)
	for _, node := range g.Nodes {
		if node.IsExternal {
			result.ExternalData = append(result.ExternalData, node)
			consumedNodes[node.ID] = true
		}
	}

	// Step 2: Evaluate Primary Module Targets (Custom first, then Official)
	allSpecs := append(e.CustomSpecs, e.OfficialSpecs...)
	seenModuleNames := make(map[string]int)

	for _, spec := range allSpecs {
		for _, node := range g.Nodes {
			if consumedNodes[node.ID] || node.IsDynamic {
				continue
			}

			if node.Asset.TerraformResourceType() == spec.PrimaryResourceType {
				// Matched a primary resource! Materialize module call
				seenCount := seenModuleNames[spec.Name]
				seenModuleNames[spec.Name]++

				modCall := e.materializeModule(spec, node, g, consumedNodes, seenCount)
				if modCall != nil {
					result.ResolvedModules = append(result.ResolvedModules, modCall)
				}
			}
		}
	}

	// Step 3: Any remaining non-dynamic, non-consumed node falls back to Raw HCL
	for _, node := range g.Nodes {
		if consumedNodes[node.ID] || node.IsDynamic || node.IsExternal {
			continue
		}
		result.RawResources = append(result.RawResources, node)
	}

	return result, nil
}

// materializeModule builds the inputs, bundles children, and synthesizes import addresses.
func (e *Engine) materializeModule(spec *ModuleSpec, primary *graph.Node, g *graph.Graph, consumed map[string]bool, instanceIdx int) *ResolvedModuleCall {
	consumed[primary.ID] = true

	modName := spec.Name
	if instanceIdx > 0 {
		cleanName := strings.ReplaceAll(strings.ToLower(primary.Asset.DisplayName), "-", "_")
		cleanName = strings.ReplaceAll(cleanName, ".", "_")
		modName = fmt.Sprintf("%s_%s", spec.Name, cleanName)
	}

	primary.TargetModule = fmt.Sprintf("module.%s", modName)

	call := &ResolvedModuleCall{
		ModuleName:       modName,
		Source:           spec.Source,
		Version:          spec.Version,
		Domain:           primary.Asset.Domain(),
		Inputs:           make(map[string]interface{}),
		ImportTargets:    make([]ImportTarget, 0),
		ChildResourceIDs: make([]string, 0),
	}

	// Map primary resource fields
	for rawField, modVar := range spec.FieldMappings {
		if val, ok := primary.Asset.ResourceData[rawField]; ok {
			call.Inputs[modVar] = val
		} else if rawField == "name" {
			call.Inputs[modVar] = primary.Asset.DisplayName
		}
	}

	// Add primary import target
	call.ImportTargets = append(call.ImportTargets, ImportTarget{
		ToAddress: fmt.Sprintf("module.%s.%s.%s", modName, spec.PrimaryResourceType, spec.InternalResourceName),
		CloudID:   primary.Asset.Name,
	})

	// Bundle children if configured
	for _, childRule := range spec.BundledChildren {
		// Check network bundling granularity
		if spec.PrimaryResourceType == "google_compute_network" {
			if e.BundleNetwork == "none" {
				continue // Keep all children standalone!
			}
			if e.BundleNetwork == "subnets-only" && childRule.ResourceType != "google_compute_subnetwork" {
				continue // Skip routes and firewalls
			}
		}

		childItems := make([]map[string]interface{}, 0)

		for _, node := range g.Nodes {
			if consumed[node.ID] || node.IsDynamic || node.IsExternal {
				continue
			}

			if node.Asset.TerraformResourceType() == childRule.ResourceType {
				// Verify dependency link to primary resource
				isChild := false
				for _, dep := range node.Dependencies {
					if dep.ID == primary.ID {
						isChild = true
						break
					}
				}

				if isChild {
					consumed[node.ID] = true
					node.TargetModule = fmt.Sprintf("module.%s", modName)
					call.ChildResourceIDs = append(call.ChildResourceIDs, node.ID)

					// Map child fields
					childInput := make(map[string]interface{})
					for rawF, modF := range childRule.FieldMappings {
						if v, ok := node.Asset.ResourceData[rawF]; ok {
							childInput[modF] = v
						} else if rawF == "name" {
							childInput[modF] = node.Asset.DisplayName
						} else if rawF == "region" {
							childInput[modF] = node.Asset.Location
						}
					}
					childItems = append(childItems, childInput)

					// Calculate exact for_each index key for import
					indexKey := e.resolveIndexKey(childRule.IndexKeyPattern, node)
					toAddress := fmt.Sprintf("module.%s.%s.%s[\"%s\"]",
						modName, childRule.ResourceType, childRule.InternalResourceName, indexKey)

					call.ImportTargets = append(call.ImportTargets, ImportTarget{
						ToAddress: toAddress,
						CloudID:   node.Asset.Name,
					})
				}
			}
		}

		if len(childItems) > 0 {
			call.Inputs[childRule.MapToInputVar] = childItems
		}
	}

	return call
}

func (e *Engine) resolveIndexKey(pattern string, node *graph.Node) string {
	if pattern == "" {
		return node.Asset.DisplayName
	}

	key := pattern
	key = strings.ReplaceAll(key, "${name}", node.Asset.DisplayName)
	key = strings.ReplaceAll(key, "${region}", node.Asset.Location)
	return key
}
