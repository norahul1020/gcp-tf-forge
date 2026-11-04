package main

import (
	"fmt"
	"os"

	"gcp-tf-forge/pkg/verifier"

	"github.com/spf13/cobra"
)

var (
	cleanupDir string
	keepBackup bool
)

var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Verify 100% state binding and prune transitional *_imports.tf files",
	Long: `Cleanup inspects the active Terraform state to ensure all resources have been
successfully imported. Once state binding is verified, it safely deletes transitional
*_imports.tf files so the Git repository contains only clean, native HCL.`,
	RunE: runCleanup,
}

func init() {
	cleanupCmd.Flags().StringVar(&cleanupDir, "dir", "./generated-infra", "Directory containing the generated Terraform code")
	cleanupCmd.Flags().BoolVar(&keepBackup, "backup", true, "Preserve an archived copy of imports in .archive/")
	rootCmd.AddCommand(cleanupCmd)
}

func runCleanup(cmd *cobra.Command, args []string) error {
	fmt.Printf("🧹 Running post-import cleanup in directory: %s\n\n", cleanupDir)

	prunedCount, err := verifier.PruneImportFiles(cleanupDir, keepBackup)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Cleanup Error: %v\n", err)
		return err
	}

	fmt.Printf("✔ State Verification: 100%% of imported resources are confirmed in state.\n")
	fmt.Printf("✔ Pruned %d transitional import files.\n", prunedCount)
	if keepBackup {
		fmt.Printf("✔ Archived import blocks saved to: %s/.archive/imports/\n", cleanupDir)
	}

	fmt.Println("\n✨ Repository is clean and Git-ready! You can now commit pure HCL code.")
	return nil
}
