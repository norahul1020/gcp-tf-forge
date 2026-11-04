package compliance

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gcp-tf-forge/pkg/discovery"
)

func TestComplianceEvaluation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "compliance-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	assets := []*discovery.Asset{
		// Insecure bucket (no public access prevention)
		{
			Name:        "bucket-insecure",
			AssetType:   "storage.googleapis.com/Bucket",
			DisplayName: "bucket-insecure",
			ResourceData: map[string]interface{}{
				"publicAccessPrevention": "inherited",
			},
		},
		// Insecure firewall (0.0.0.0/0 on port 22)
		{
			Name:        "allow-all-ssh",
			AssetType:   "compute.googleapis.com/Firewall",
			DisplayName: "allow-all-ssh",
			ResourceData: map[string]interface{}{
				"sourceRanges": []interface{}{"0.0.0.0/0"},
				"allowed": []interface{}{
					map[string]interface{}{
						"ports": []interface{}{"22"},
					},
				},
			},
		},
	}

	eval := NewEvaluator()
	eval.Evaluate(assets)

	if len(eval.Findings) != 2 {
		t.Fatalf("expected 2 compliance findings, got %d", len(eval.Findings))
	}

	hasSSHFailure := false
	for _, f := range eval.Findings {
		if f.ControlID == "CIS-3.6" && f.Status == "FAIL" {
			hasSSHFailure = true
		}
	}
	if !hasSSHFailure {
		t.Errorf("expected CIS-3.6 failure for unrestricted SSH")
	}

	// Verify report generation
	if err := eval.GenerateReport(tempDir, "test-proj"); err != nil {
		t.Fatalf("GenerateReport failed: %v", err)
	}

	reportBytes, err := os.ReadFile(filepath.Join(tempDir, "COMPLIANCE_AUDIT.md"))
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(reportBytes), "CIS Google Cloud Foundations Benchmark") {
		t.Errorf("report missing benchmark header")
	}
}
