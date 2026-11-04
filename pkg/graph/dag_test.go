package graph

import (
	"testing"

	"gcp-tf-forge/pkg/discovery"
)

func TestGraphDependencyAndFiltering(t *testing.T) {
	projectID := "my-app-prod"
	g := NewGraph(projectID)

	assets := []*discovery.Asset{
		{
			Name:        "//compute.googleapis.com/projects/my-app-prod/global/networks/vpc-main",
			AssetType:   "compute.googleapis.com/Network",
			ProjectID:   projectID,
			DisplayName: "vpc-main",
			ResourceData: map[string]interface{}{
				"name": "vpc-main",
			},
		},
		{
			Name:        "//compute.googleapis.com/projects/my-app-prod/regions/us-central1/subnetworks/subnet-web",
			AssetType:   "compute.googleapis.com/Subnetwork",
			ProjectID:   projectID,
			DisplayName: "subnet-web",
			ResourceData: map[string]interface{}{
				"name":    "subnet-web",
				"network": "//compute.googleapis.com/projects/my-app-prod/global/networks/vpc-main",
			},
		},
		{
			Name:        "//compute.googleapis.com/projects/my-app-prod/zones/us-central1-a/instances/web-server-01",
			AssetType:   "compute.googleapis.com/Instance",
			ProjectID:   projectID,
			DisplayName: "web-server-01",
			ResourceData: map[string]interface{}{
				"name": "web-server-01",
				"networkInterfaces": []interface{}{
					map[string]interface{}{
						"subnetwork": "//compute.googleapis.com/projects/my-app-prod/regions/us-central1/subnetworks/subnet-web",
					},
				},
			},
		},
		// Ephemeral GKE worker node that should be filtered out
		{
			Name:        "//compute.googleapis.com/projects/my-app-prod/zones/us-central1-a/instances/gke-cluster-pool-1234-node",
			AssetType:   "compute.googleapis.com/Instance",
			ProjectID:   projectID,
			DisplayName: "gke-cluster-pool-1234-node",
			Labels: map[string]string{
				"goog-gke-node": "true",
			},
			ResourceData: map[string]interface{}{},
		},
		// Standalone VM using Shared VPC subnet from host project
		{
			Name:        "//compute.googleapis.com/projects/my-app-prod/zones/us-central1-b/instances/backend-01",
			AssetType:   "compute.googleapis.com/Instance",
			ProjectID:   projectID,
			DisplayName: "backend-01",
			ResourceData: map[string]interface{}{
				"name": "backend-01",
				"networkInterfaces": []interface{}{
					map[string]interface{}{
						"subnetwork": "projects/shared-vpc-host/regions/us-central1/subnetworks/shared-subnet",
					},
				},
			},
		},
	}

	g.Build(assets)

	// 1. Verify Dynamic Node Filtering
	gkeNode := g.findNodeByURI("gke-cluster-pool-1234-node")
	if gkeNode == nil || !gkeNode.IsDynamic {
		t.Fatalf("expected GKE node to be marked as dynamic")
	}

	// 2. Verify Subnet -> VPC dependency
	subNode := g.findNodeByURI("subnet-web")
	if subNode == nil || len(subNode.Dependencies) != 1 {
		t.Fatalf("expected subnet to have 1 dependency (VPC)")
	}
	if subNode.Dependencies[0].Asset.DisplayName != "vpc-main" {
		t.Errorf("expected subnet dependency to be vpc-main, got %s", subNode.Dependencies[0].Asset.DisplayName)
	}

	// 3. Verify Shared VPC External Data Node
	sharedSubnet := g.findNodeByURI("shared-subnet")
	if sharedSubnet == nil || !sharedSubnet.IsExternal {
		t.Fatalf("expected shared-subnet to be created as an external data node")
	}
	if sharedSubnet.Asset.ProjectID != "shared-vpc-host" {
		t.Errorf("expected external subnet project to be shared-vpc-host, got %s", sharedSubnet.Asset.ProjectID)
	}

	// 4. Verify Topological Sort (VPC before Subnet before VM)
	sorted, err := g.TopologicalSort()
	if err != nil {
		t.Fatalf("topological sort failed: %v", err)
	}

	indices := make(map[string]int)
	for i, n := range sorted {
		indices[n.Asset.DisplayName] = i
	}

	if indices["vpc-main"] > indices["subnet-web"] {
		t.Errorf("expected vpc-main to appear before subnet-web in sorted order")
	}
	if indices["subnet-web"] > indices["web-server-01"] {
		t.Errorf("expected subnet-web to appear before web-server-01 in sorted order")
	}
}
