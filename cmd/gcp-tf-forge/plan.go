package main

import (
	"context"
	"fmt"

	"gcp-tf-forge/pkg/compliance"
	"gcp-tf-forge/pkg/discovery"
	"gcp-tf-forge/pkg/graph"
	"gcp-tf-forge/pkg/hierarchy"
	"gcp-tf-forge/pkg/matcher"

	"github.com/spf13/cobra"
)

var planCmd = &cobra.Command{
	Use:   "plan",
	Short: "Scan cloud assets and preview module mapping without writing files to disk",
	Long: `Plan connects to GCP, discovers active infrastructure in memory, evaluates
user custom modules and official Google modules, and prints an executive summary
preview of how resources will be grouped into domains and modules.`,
	RunE: runPlan,
}

func init() {
	rootCmd.AddCommand(planCmd)
}

func runPlan(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	if projectID == "" {
		return fmt.Errorf("project ID is required. Use --project or -p flag")
	}

	fmt.Printf("🔍 Scanning GCP project: %s (Dry-run mode)...\n", projectID)
	lineage, err := hierarchy.ResolveAncestry(ctx, projectID)
	if err == nil {
		fmt.Printf("✔ Hierarchy: %s (%s)\n", lineage.Breadcrumb, lineage.Type)
	}

	// 1. Asset Discovery (CAI with fallback)
	var assets []*discovery.Asset
	if lineage != nil && lineage.CAIAvailable {
		scanner := discovery.NewCAIScanner(projectID)
		assets, err = scanner.Scan(ctx)
	}

	if err != nil || len(assets) == 0 {
		fmt.Println("ℹ️  Using direct API crawler fallback...")
		crawler := discovery.NewDirectCrawler(projectID)
		assets, _ = crawler.Crawl(ctx)
	}

	fmt.Printf("✔ Discovered %d live cloud assets.\n", len(assets))

	// 2. Build Dependency Graph
	g := graph.NewGraph(projectID)
	g.Build(assets)

	// 3. Reconcile Modules
	engine := matcher.NewEngine(nil, "all")
	matchResult, err := engine.Reconcile(g)
	if err != nil {
		return fmt.Errorf("module reconciliation failed: %w", err)
	}

	// 4. Run Compliance Evaluation
	eval := compliance.NewEvaluator()
	eval.Evaluate(assets)

	// Print Summary Table
	fmt.Println("\n┌──────────────────┬───────┬───────────────────────────────┬────────────────────────┐")
	fmt.Println("│ Resource Domain  │ Count │ Module Target                 │ Mapping Strategy       │")
	fmt.Println("├──────────────────┼───────┼───────────────────────────────┼────────────────────────┤")

	domains := g.NodesByDomain()
	for domain, nodes := range domains {
		modCount := 0
		for _, mod := range matchResult.ResolvedModules {
			if mod.Domain == domain {
				modCount++
			}
		}

		strategy := "Clean Raw Fallback"
		target := "Standard google_*"
		if modCount > 0 {
			strategy = "Google Official Module"
			target = fmt.Sprintf("%d module call(s)", modCount)
		}

		fmt.Printf("│ %-16s │ %-5d │ %-29s │ %-22s │\n", domain, len(nodes), target, strategy)
	}
	fmt.Println("└──────────────────┴───────┴───────────────────────────────┴────────────────────────┘")

	if len(matchResult.ExternalData) > 0 {
		fmt.Printf("\nℹ️  Detected %d external Shared VPC resource(s) (will emit as data sources).\n", len(matchResult.ExternalData))
	}

	passCount := 0
	for _, f := range eval.Findings {
		if f.Status == "PASS" {
			passCount++
		}
	}
	fmt.Printf("🛡️  CIS Benchmark Preview: %d evaluated controls (%d passing).\n", len(eval.Findings), passCount)

	fmt.Println("\n✨ Dry run complete. 0 files written to disk.")
	fmt.Printf("Run 'gcp-tf-forge export --project=%s --out=%s' to generate code.\n", projectID, outputDir)
	return nil
}

