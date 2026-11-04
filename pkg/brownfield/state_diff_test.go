package brownfield

import (
	"os"
	"path/filepath"
	"testing"

	"gcp-tf-forge/pkg/discovery"
)

func TestStateDiffing(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "brownfield-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create dummy state file with 1 managed network
	dummyStateJSON := `{
  "version": 4,
  "resources": [
    {
      "type": "google_compute_network",
      "name": "vpc",
      "instances": [
        {
          "attributes": {
            "id": "projects/my-proj/global/networks/managed-vpc",
            "name": "managed-vpc"
          }
        }
      ]
    }
  ]
}`
	statePath := filepath.Join(tempDir, "terraform.tfstate")
	if err := os.WriteFile(statePath, []byte(dummyStateJSON), 0644); err != nil {
		t.Fatal(err)
	}

	liveAssets := []*discovery.Asset{
		{
			Name:        "projects/my-proj/global/networks/managed-vpc",
			DisplayName: "managed-vpc",
			AssetType:   "compute.googleapis.com/Network",
		},
		{
			Name:        "projects/my-proj/regions/us-central1/subnetworks/untracked-sub",
			DisplayName: "untracked-sub",
			AssetType:   "compute.googleapis.com/Subnetwork",
		},
	}

	diff, err := IngestAndDiff(statePath, liveAssets)
	if err != nil {
		t.Fatalf("IngestAndDiff failed: %v", err)
	}

	if len(diff.AlreadyManaged) != 1 || diff.AlreadyManaged[0] != "managed-vpc" {
		t.Errorf("expected managed-vpc to be detected as already managed, got %v", diff.AlreadyManaged)
	}

	if len(diff.Untracked) != 1 || diff.Untracked[0] != "untracked-sub" {
		t.Errorf("expected untracked-sub to be detected as ClickOps untracked, got %v", diff.Untracked)
	}

	// Verify report generation
	if err := GenerateDriftReport(tempDir, "my-proj", diff); err != nil {
		t.Fatalf("GenerateDriftReport failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tempDir, "DRIFT_AND_CLICKOPS_REPORT.md")); os.IsNotExist(err) {
		t.Errorf("drift report was not generated")
	}
}
