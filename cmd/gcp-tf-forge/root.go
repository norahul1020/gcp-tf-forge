package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	version = "0.1.0"
	commit  = "dev"
	date    = "unknown"

	// Global CLI flags
	projectID     string
	folderID      string
	orgID         string
	customModules string
	outputDir     string
	layout        string
	verbose       bool
)

var rootCmd = &cobra.Command{
	Use:   "gcp-tf-forge",
	Short: "gcp-tf-forge is an enterprise-grade GCP Terraform importer and modernizer",
	Long: `gcp-tf-forge reverse-engineers Google Cloud infrastructure into clean,
modular, and maintainable Terraform HCL.

Key Features:
  * Synthesizes resources into User Custom Modules or Official Google Modules
  * Isolates domain configurations into auto-loaded *.auto.tfvars
  * Enforces Zero-Trust Security (Zero secret payload access, lifecycle shields)
  * Supports Standalone Projects, Organizations, Folders, and Shared VPCs
  * Reconciles false drift for a verified 0-drift plan
  * Automatically verifies and prunes transitional import blocks`,
	Version: fmt.Sprintf("%s (commit: %s, date: %s)", version, commit, date),
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&projectID, "project", "p", "", "Target GCP Project ID")
	rootCmd.PersistentFlags().StringVarP(&folderID, "folder", "f", "", "Target GCP Folder ID (optional)")
	rootCmd.PersistentFlags().StringVarP(&orgID, "org", "o", "", "Target GCP Organization ID (optional)")
	rootCmd.PersistentFlags().StringVarP(&customModules, "custom-modules", "m", "", "Path to user custom Terraform modules directory")
	rootCmd.PersistentFlags().StringVarP(&outputDir, "out", "d", "./generated-infra", "Output directory for generated Terraform code")
	rootCmd.PersistentFlags().StringVar(&layout, "layout", "root-composition", "Output layout: 'root-composition' (default) or 'independent-stacks'")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose debug logging")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
