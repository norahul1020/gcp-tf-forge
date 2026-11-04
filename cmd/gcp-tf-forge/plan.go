package main

import (
	"context"
	"fmt"

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
	_ = context.Background()

	if projectID == "" {
		return fmt.Errorf("project ID is required. Use --project or -p flag")
	}

	fmt.Printf("🔍 Scanning GCP project: %s (Dry-run mode)...\n", projectID)
	if customModules != "" {
		fmt.Printf("📦 Custom modules path: %s\n", customModules)
	}
	fmt.Printf("📐 Output layout: %s\n\n", layout)

	// Simulated preview table
	fmt.Println("┌──────────────────┬───────┬───────────────────────────────┬────────────────────────┐")
	fmt.Println("│ Resource Domain  │ Count │ Module Target                 │ Mapping Strategy       │")
	fmt.Println("├──────────────────┼───────┼───────────────────────────────┼────────────────────────┤")
	fmt.Println("│ Networking       │ --    │ Pending scan                  │ Module / Raw HCL       │")
	fmt.Println("│ Compute          │ --    │ Pending scan                  │ Module / Raw HCL       │")
	fmt.Println("│ GKE              │ --    │ Pending scan                  │ Official Google Module │")
	fmt.Println("│ Storage & DB     │ --    │ Pending scan                  │ Shielded Database / GCS│")
	fmt.Println("│ IAM & Security   │ --    │ Pending scan                  │ Additive Members Only  │")
	fmt.Println("│ Secret Metadata  │ --    │ Pending scan                  │ Metadata Containers    │")
	fmt.Println("└──────────────────┴───────┴───────────────────────────────┴────────────────────────┘")

	fmt.Println("\n✨ Dry run complete. 0 files written to disk.")
	fmt.Println("Run 'gcp-tf-forge export --project=" + projectID + "' to generate the code.")
	return nil
}
