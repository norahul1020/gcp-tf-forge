package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gcp-tf-forge/pkg/brownfield"
	"gcp-tf-forge/pkg/compliance"
	"gcp-tf-forge/pkg/discovery"
	"gcp-tf-forge/pkg/graph"
	"gcp-tf-forge/pkg/hclgen"
	"gcp-tf-forge/pkg/matcher"
)

// TestEdgeCases_FullBattery tests the critical failure modes identified by Senior Staff Engineers.
func TestEdgeCases_FullBattery(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "tf-edge-cases-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	projectID := "enterprise-core-prod"

	// 1. Simulating complex real-world assets with edge cases
	assets := []*discovery.Asset{
		// Asset 1: Primary VPC
		{
			Name:        "//compute.googleapis.com/projects/enterprise-core-prod/global/networks/core-vpc",
			AssetType:   "compute.googleapis.com/Network",
			ProjectID:   projectID,
			DisplayName: "core-vpc",
			ResourceData: map[string]interface{}{
				"name":         "core-vpc",
				"routing_mode": "GLOBAL",
			},
		},
		// Asset 2: Subnet with special characters in name that need sanitization
		{
			Name:        "//compute.googleapis.com/projects/enterprise-core-prod/regions/us-east1/subnetworks/1st.subnet-core",
			AssetType:   "compute.googleapis.com/Subnetwork",
			ProjectID:   projectID,
			Location:    "us-east1",
			DisplayName: "1st.subnet-core",
			ResourceData: map[string]interface{}{
				"name":          "1st.subnet-core",
				"ip_cidr_range": "10.100.0.0/20",
				"region":        "us-east1",
				"network":       "//compute.googleapis.com/projects/enterprise-core-prod/global/networks/core-vpc",
			},
		},
		// Asset 3: GKE Cluster
		{
			Name:        "//container.googleapis.com/projects/enterprise-core-prod/locations/us-east1/clusters/prod-k8s-cluster",
			AssetType:   "container.googleapis.com/Cluster",
			ProjectID:   projectID,
			Location:    "us-east1",
			DisplayName: "prod-k8s-cluster",
			ResourceData: map[string]interface{}{
				"name":     "prod-k8s-cluster",
				"location": "us-east1",
				"network":  "core-vpc",
			},
		},
		// Asset 4: GKE Node Pool
		{
			Name:        "//container.googleapis.com/projects/enterprise-core-prod/locations/us-east1/clusters/prod-k8s-cluster/nodePools/system-pool",
			AssetType:   "container.googleapis.com/NodePool",
			ProjectID:   projectID,
			Location:    "us-east1",
			DisplayName: "system-pool",
			ResourceData: map[string]interface{}{
				"name":         "system-pool",
				"cluster_id":   "//container.googleapis.com/projects/enterprise-core-prod/locations/us-east1/clusters/prod-k8s-cluster",
				"machine_type": "n2-standard-4",
				"node_count":   3,
			},
		},
		// Asset 5: Ephemeral GKE worker VM (MUST BE FILTERED OUT)
		{
			Name:        "//compute.googleapis.com/projects/enterprise-core-prod/zones/us-east1-b/instances/gke-prod-k8s-cluster-system-pool-1234-node",
			AssetType:   "compute.googleapis.com/Instance",
			ProjectID:   projectID,
			DisplayName: "gke-prod-k8s-cluster-system-pool-1234-node",
			Labels: map[string]string{
				"goog-gke-node": "true",
			},
			ResourceData: map[string]interface{}{},
		},
		// Asset 6: Cloud SQL Instance (MUST RECEIVE PASSWORD LIFECYCLE SHIELD)
		{
			Name:        "//sqladmin.googleapis.com/projects/enterprise-core-prod/instances/master-postgres-db",
			AssetType:   "sqladmin.googleapis.com/Instance",
			ProjectID:   projectID,
			DisplayName: "master-postgres-db",
			ResourceData: map[string]interface{}{
				"name":             "master-postgres-db",
				"database_version": "POSTGRES_15",
				"region":           "us-east1",
				"settings": map[string]interface{}{
					"backupConfiguration": map[string]interface{}{
						"enabled": true,
					},
				},
			},
		},
		// Asset 7: Insecure Firewall with 0.0.0.0/0 on port 22 (MUST BE FLAGGED BY CIS AUDIT)
		{
			Name:        "//compute.googleapis.com/projects/enterprise-core-prod/global/firewalls/allow-public-ssh",
			AssetType:   "compute.googleapis.com/Firewall",
			ProjectID:   projectID,
			DisplayName: "allow-public-ssh",
			ResourceData: map[string]interface{}{
				"name":         "allow-public-ssh",
				"network":      "//compute.googleapis.com/projects/enterprise-core-prod/global/networks/core-vpc",
				"sourceRanges": []interface{}{"0.0.0.0/0"},
				"allowed": []interface{}{
					map[string]interface{}{
						"ports": []interface{}{"22"},
					},
				},
			},
		},
	}

	// Step 1: Build Topology Graph
	g := graph.NewGraph(projectID)
	g.Build(assets)

	// Verify ephemeral VM was filtered out
	for _, n := range g.Nodes {
		if strings.Contains(n.Asset.DisplayName, "gke-prod-k8s") && !n.IsDynamic {
			t.Errorf("ephemeral GKE VM was not filtered out as dynamic")
		}
	}

	// Step 2: Module Reconciliation
	engine := matcher.NewEngine(nil, "all")
	matchResult, err := engine.Reconcile(g)
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Should have matched VPC module and GKE module
	if len(matchResult.ResolvedModules) < 2 {
		t.Fatalf("expected at least 2 modules (vpc and gke), got %d", len(matchResult.ResolvedModules))
	}

	// Step 3: HCL Code Emission
	gen := hclgen.NewGenerator(tempDir, projectID, "root-composition")
	if err := gen.Generate(matchResult); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Step 4: Verify CIS Compliance Audit
	eval := compliance.NewEvaluator()
	eval.Evaluate(assets)
	if err := eval.GenerateReport(tempDir, projectID); err != nil {
		t.Fatalf("GenerateReport failed: %v", err)
	}

	auditContent, _ := os.ReadFile(filepath.Join(tempDir, "COMPLIANCE_AUDIT.md"))
	if !strings.Contains(string(auditContent), "CIS-3.6") {
		t.Errorf("expected CIS-3.6 unrestricted SSH failure in compliance report")
	}

	// Step 5: Verify Brownfield State Diffing
	dummyStateJSON := `{
  "version": 4,
  "resources": [
    {
      "type": "google_compute_network",
      "name": "core_vpc",
      "instances": [
        {
          "attributes": {
            "name": "core-vpc"
          }
        }
      ]
    }
  ]
}`
	stateFile := filepath.Join(tempDir, "existing.tfstate")
	_ = os.WriteFile(stateFile, []byte(dummyStateJSON), 0644)

	diff, err := brownfield.IngestAndDiff(stateFile, assets)
	if err != nil {
		t.Fatalf("IngestAndDiff failed: %v", err)
	}

	if len(diff.AlreadyManaged) != 1 || diff.AlreadyManaged[0] != "core-vpc" {
		t.Errorf("expected core-vpc to be identified as already managed")
	}

	_ = brownfield.GenerateDriftReport(tempDir, projectID, diff)
	driftContent, _ := os.ReadFile(filepath.Join(tempDir, "DRIFT_AND_CLICKOPS_REPORT.md"))
	if !strings.Contains(string(driftContent), "Already Managed in Terraform:") {
		t.Errorf("drift report content was:\n%s", string(driftContent))
	}
}
