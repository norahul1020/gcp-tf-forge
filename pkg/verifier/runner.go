package verifier

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// PlanResult holds the summary metrics from a terraform plan execution.
type PlanResult struct {
	ToImport  int
	ToAdd     int
	ToChange  int
	ToDestroy int
}

// RunPlanCheck executes terraform validate and plan within the target directory.
func RunPlanCheck(dir string) (*PlanResult, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("invalid directory path: %w", err)
	}

	// 1. Run terraform validate
	validateCmd := exec.Command("terraform", "validate")
	validateCmd.Dir = absDir
	if out, err := validateCmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("terraform validate failed:\n%s", string(out))
	}

	// 2. Run terraform plan (non-interactive)
	planCmd := exec.Command("terraform", "plan", "-no-color")
	planCmd.Dir = absDir
	out, err := planCmd.CombinedOutput()
	planOutput := string(out)

	result := &PlanResult{}

	// Parse plan summary (e.g. "Plan: 12 to import, 0 to add, 0 to change, 0 to destroy.")
	if strings.Contains(planOutput, "to import") || strings.Contains(planOutput, "to add") {
		for _, line := range strings.Split(planOutput, "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "Plan:") {
				fmt.Sscanf(line, "Plan: %d to import, %d to add, %d to change, %d to destroy.",
					&result.ToImport, &result.ToAdd, &result.ToChange, &result.ToDestroy)
			}
		}
	}

	// If terraform plan exited with error, propagate output
	if err != nil && !strings.Contains(planOutput, "Plan:") {
		return nil, fmt.Errorf("terraform plan error:\n%s", planOutput)
	}

	return result, nil
}
