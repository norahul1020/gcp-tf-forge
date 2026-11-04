package hclgen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gcp-tf-forge/pkg/discovery"
	"gcp-tf-forge/pkg/graph"
	"gcp-tf-forge/pkg/matcher"
)

func TestGeneratorEndToEnd(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "tf-gen-test-*")
	if err != nil {
		t.Fatalf("failed creating temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	projectID := "prod-app-1042"
	gen := NewGenerator(tempDir, projectID, "root-composition")

	// Create test match result with a module, a raw resource, and an external Shared VPC subnet
	matchResult := &matcher.MatchResult{
		ResolvedModules: []*matcher.ResolvedModuleCall{
			{
				ModuleName: "vpc",
				Source:     "terraform-google-modules/network/google",
				Version:    "~> 9.0",
				Domain:     "networking",
				Inputs: map[string]interface{}{
					"network_name": "prod-vpc",
					"subnets": []map[string]interface{}{
						{
							"subnet_name": "subnet-a",
							"subnet_ip":   "10.0.1.0/24",
						},
					},
				},
				ImportTargets: []matcher.ImportTarget{
					{
						ToAddress: `module.vpc.google_compute_network.network`,
						CloudID:   "projects/prod-app-1042/global/networks/prod-vpc",
					},
					{
						ToAddress: `module.vpc.google_compute_subnetwork.subnets["us-central1/subnet-a"]`,
						CloudID:   "projects/prod-app-1042/regions/us-central1/subnetworks/subnet-a",
					},
				},
			},
		},
		RawResources: []*graph.Node{
			{
				ID: "sql-user-db",
				Asset: &discovery.Asset{
					Name:        "//sqladmin.googleapis.com/projects/prod-app-1042/instances/prod-db",
					AssetType:   "sqladmin.googleapis.com/Instance",
					ProjectID:   projectID,
					DisplayName: "prod-db",
					ResourceData: map[string]interface{}{
						"name": "prod-db",
					},
				},
			},
		},
		ExternalData: []*graph.Node{
			{
				ID:         "shared-subnet",
				IsExternal: true,
				Asset: &discovery.Asset{
					Name:        "projects/shared-vpc-host/regions/us-central1/subnetworks/shared-sub",
					AssetType:   "compute.googleapis.com/Subnetwork",
					ProjectID:   "shared-vpc-host",
					Location:    "us-central1",
					DisplayName: "shared-sub",
				},
			},
		},
	}

	if err := gen.Generate(matchResult); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// 1. Verify versions.tf & providers.tf
	if _, err := os.Stat(filepath.Join(tempDir, "versions.tf")); os.IsNotExist(err) {
		t.Errorf("versions.tf was not created")
	}
	if _, err := os.Stat(filepath.Join(tempDir, "providers.tf")); os.IsNotExist(err) {
		t.Errorf("providers.tf was not created")
	}

	// 2. Verify Networking Domain Files
	netTF := filepath.Join(tempDir, "networking.tf")
	netImports := filepath.Join(tempDir, "networking_imports.tf")
	netTFVars := filepath.Join(tempDir, "networking.auto.tfvars.json")

	if _, err := os.Stat(netTF); os.IsNotExist(err) {
		t.Errorf("networking.tf was not created")
	}
	if _, err := os.Stat(netImports); os.IsNotExist(err) {
		t.Errorf("networking_imports.tf was not created")
	}
	if _, err := os.Stat(netTFVars); os.IsNotExist(err) {
		t.Errorf("networking.auto.tfvars.json was not created")
	}

	// Check contents of networking_imports.tf
	importBytes, _ := os.ReadFile(netImports)
	importContent := string(importBytes)
	if !strings.Contains(importContent, `to = module.vpc.google_compute_network.network`) {
		t.Errorf("networking_imports.tf missing network import target")
	}
	if !strings.Contains(importContent, `to = module.vpc.google_compute_subnetwork.subnets["us-central1/subnet-a"]`) {
		t.Errorf("networking_imports.tf missing subnet import target with region index")
	}

	// 3. Verify Database Lifecycle Shield
	storageTF := filepath.Join(tempDir, "storage.tf")
	storageBytes, _ := os.ReadFile(storageTF)
	if !strings.Contains(string(storageBytes), "ignore_changes = [password]") {
		t.Errorf("storage.tf missing password lifecycle shield for Cloud SQL")
	}

	// 4. Verify External Shared VPC Data Source
	extDataTF := filepath.Join(tempDir, "external_data.tf")
	extBytes, _ := os.ReadFile(extDataTF)
	if !strings.Contains(string(extBytes), `data "google_compute_subnetwork"`) {
		t.Errorf("external_data.tf missing shared subnet data source: %s", string(extBytes))
	}
}
