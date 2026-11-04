package main

import (
	"fmt"
	"os"

	"gcp-tf-forge/pkg/verifier"

	"github.com/spf13/cobra"
)

var verifyDir string

var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Run non-destructive terraform validate and plan to assert 0 drift",
	Long: `Verify executes local terraform validate and terraform plan against the
generated directory to confirm that all import blocks resolve cleanly and that the plan
produces the golden metric: "0 to add, 0 to change, 0 to destroy".`,
	RunE: runVerify,
}

func init() {
	verifyCmd.Flags().StringVarP(&verifyDir, "dir", "d", "./generated-infra", "Directory containing the generated Terraform code")
	rootCmd.AddCommand(verifyCmd)
}

func runVerify(cmd *cobra.Command, args []string) error {
	fmt.Printf("🔍 Running non-destructive verification in directory: %s\n\n", verifyDir)

	result, err := verifier.RunPlanCheck(verifyDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Verification Failed: %v\n", err)
		return err
	}

	fmt.Printf("✔ Terraform Validate: PASSED\n")
	fmt.Printf("✔ Plan Summary: %d to import, %d to add, %d to change, %d to destroy\n",
		result.ToImport, result.ToAdd, result.ToChange, result.ToDestroy)

	if result.ToAdd == 0 && result.ToChange == 0 && result.ToDestroy == 0 {
		fmt.Println("\n✨ Golden Metric Achieved: 100% clean import with ZERO unexpected drift!")
	} else {
		fmt.Printf("\n⚠️  Drift Notice: Review the plan diff above before applying.\n")
	}

	return nil
}
