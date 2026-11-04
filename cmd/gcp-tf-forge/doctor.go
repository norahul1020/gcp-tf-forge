package main

import (
	"context"
	"fmt"
	"os"

	"gcp-tf-forge/pkg/auth"
	"gcp-tf-forge/pkg/hierarchy"

	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Perform pre-flight verification of GCP credentials, permissions, and hierarchy",
	Long: `Doctor validates your active Google Cloud authentication, verifies a read-only
security posture (ensuring zero-trust secret safety), checks API readiness, and traverses
the organizational hierarchy ancestry.`,
	RunE: runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

func runDoctor(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	if projectID == "" {
		return fmt.Errorf("project ID is required. Use --project or -p flag")
	}

	fmt.Printf("🔍 Running pre-flight security and hierarchy check for project: %s\n\n", projectID)

	// 1. Check Authentication & Security Posture
	authResult, err := auth.VerifyCredentials(ctx, projectID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ GCP Authentication Error: %v\n", err)
		return err
	}

	fmt.Printf("✔ GCP Identity: %s\n", authResult.CallerIdentity)
	if authResult.IsReadOnly {
		fmt.Printf("✔ Security Posture: Read-Only Verified (Zero-Trust Compliant)\n")
	} else {
		fmt.Printf("⚠️  Security Posture: Running with write privileges (%s). For optimal safety, run with Viewer + Cloud Asset Viewer.\n", authResult.RoleSummary)
	}

	if authResult.HasSecretAccessor {
		fmt.Printf("⚠️  Zero-Trust Warning: Identity has secretmanager.secretAccessor. gcp-tf-forge will strictly ignore secret payloads.\n")
	} else {
		fmt.Printf("✔ Secret Safety: Zero-Payload posture verified (no secretAccessor permission)\n")
	}

	// 2. Resolve Hierarchy Ancestry
	lineage, err := hierarchy.ResolveAncestry(ctx, projectID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Hierarchy Resolution Error: %v\n", err)
		return err
	}

	fmt.Printf("✔ Hierarchy Path: %s\n", lineage.Breadcrumb)
	fmt.Printf("✔ Hierarchy Type: %s\n", lineage.Type)

	// 3. Check Cloud Asset Inventory API status
	if lineage.CAIAvailable {
		fmt.Printf("✔ Discovery Engine: Cloud Asset Inventory API available (Fast 1-Shot Scanner enabled)\n")
	} else {
		fmt.Printf("ℹ️  Discovery Engine: CAI API not enabled. Direct API crawler fallback will be used.\n")
	}

	fmt.Printf("\n✨ Doctor check complete! All pre-flight requirements passed.\n")
	return nil
}
