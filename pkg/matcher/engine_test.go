package matcher

import (
	"testing"

	"gcp-tf-forge/pkg/discovery"
	"gcp-tf-forge/pkg/graph"
)

func TestModuleReconciliation(t *testing.T) {
	projectID := "test-project"
	g := graph.NewGraph(projectID)

	assets := []*discovery.Asset{
		{
			Name:        "//compute.googleapis.com/projects/test-project/global/networks/prod-vpc",
			AssetType:   "compute.googleapis.com/Network",
			ProjectID:   projectID,
			DisplayName: "prod-vpc",
			ResourceData: map[string]interface{}{
				"name":         "prod-vpc",
				"routing_mode": "REGIONAL",
			},
		},
		{
			Name:        "//compute.googleapis.com/projects/test-project/regions/us-central1/subnetworks/subnet-web",
			AssetType:   "compute.googleapis.com/Subnetwork",
			ProjectID:   projectID,
			Location:    "us-central1",
			DisplayName: "subnet-web",
			ResourceData: map[string]interface{}{
				"name":          "subnet-web",
				"ip_cidr_range": "10.0.1.0/24",
				"region":        "us-central1",
				"network":       "//compute.googleapis.com/projects/test-project/global/networks/prod-vpc",
			},
		},
		{
			Name:        "//compute.googleapis.com/projects/test-project/zones/us-central1-a/instances/standalone-vm",
			AssetType:   "compute.googleapis.com/Instance",
			ProjectID:   projectID,
			Location:    "us-central1-a",
			DisplayName: "standalone-vm",
			ResourceData: map[string]interface{}{
				"name": "standalone-vm",
			},
		},
	}

	g.Build(assets)

	engine := NewEngine(nil, "all")
	result, err := engine.Reconcile(g)
	if err != nil {
		t.Fatalf("reconcile failed: %v", err)
	}

	// 1. Should have 1 resolved module (VPC)
	if len(result.ResolvedModules) != 1 {
		t.Fatalf("expected 1 resolved module, got %d", len(result.ResolvedModules))
	}

	vpcModule := result.ResolvedModules[0]
	if vpcModule.ModuleName != "vpc" {
		t.Errorf("expected module name 'vpc', got '%s'", vpcModule.ModuleName)
	}
	if vpcModule.Source != "terraform-google-modules/network/google" {
		t.Errorf("unexpected module source: %s", vpcModule.Source)
	}

	// 2. Verify Field Transformation
	if vpcModule.Inputs["network_name"] != "prod-vpc" {
		t.Errorf("expected network_name 'prod-vpc', got %v", vpcModule.Inputs["network_name"])
	}

	subnets, ok := vpcModule.Inputs["subnets"].([]map[string]interface{})
	if !ok || len(subnets) != 1 {
		t.Fatalf("expected 1 bundled subnet in module inputs, got %v", vpcModule.Inputs["subnets"])
	}

	if subnets[0]["subnet_name"] != "subnet-web" {
		t.Errorf("expected subnet_name 'subnet-web', got %v", subnets[0]["subnet_name"])
	}
	if subnets[0]["subnet_ip"] != "10.0.1.0/24" {
		t.Errorf("expected subnet_ip '10.0.1.0/24', got %v", subnets[0]["subnet_ip"])
	}

	// 3. Verify Exact Import Addresses (${region}/${name} key index)
	expectedSubnetImport := `module.vpc.google_compute_subnetwork.subnets["us-central1/subnet-web"]`
	foundImport := false
	for _, target := range vpcModule.ImportTargets {
		if target.ToAddress == expectedSubnetImport {
			foundImport = true
			break
		}
	}
	if !foundImport {
		t.Errorf("expected import target '%s' not found in %v", expectedSubnetImport, vpcModule.ImportTargets)
	}

	// 4. Verify Raw Fallback for Standalone VM
	if len(result.RawResources) != 1 {
		t.Fatalf("expected 1 raw fallback resource, got %d", len(result.RawResources))
	}
	if result.RawResources[0].Asset.DisplayName != "standalone-vm" {
		t.Errorf("expected standalone-vm as raw resource, got %s", result.RawResources[0].Asset.DisplayName)
	}
}
