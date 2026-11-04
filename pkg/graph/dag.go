package graph

import (
	"fmt"
	"strings"

	"gcp-tf-forge/pkg/discovery"
)

// Graph represents the topological dependency graph of all discovered resources.
type Graph struct {
	ProjectID string
	Nodes     map[string]*Node
}

// NewGraph creates an empty graph for the specified project.
func NewGraph(projectID string) *Graph {
	return &Graph{
		ProjectID: projectID,
		Nodes:     make(map[string]*Node),
	}
}

// Build ingests raw assets, builds nodes, resolves relationships, and filters dynamic instances.
func (g *Graph) Build(assets []*discovery.Asset) {
	// 1. Populate initial nodes
	for _, asset := range assets {
		node := NewNode(asset)

		// Check if asset belongs to an external project (Shared VPC host)
		if asset.ProjectID != "" && asset.ProjectID != g.ProjectID {
			node.IsExternal = true
		}

		// Detect dynamic ephemeral autoscaling nodes
		if g.isDynamicResource(asset) {
			node.IsDynamic = true
		}

		g.Nodes[node.ID] = node
	}

	// 2. Resolve relationships & link dependencies
	for _, node := range g.Nodes {
		g.resolveDependencies(node)
	}
}

// isDynamicResource flags ephemeral VMs that should not be managed directly by Terraform.
func (g *Graph) isDynamicResource(asset *discovery.Asset) bool {
	if asset.AssetType != "compute.googleapis.com/Instance" {
		return false
	}

	// Check for GKE node indicators
	if strings.HasPrefix(asset.DisplayName, "gke-") {
		return true
	}
	if _, ok := asset.Labels["goog-gke-node"]; ok {
		return true
	}

	// Check if VM was created by an Instance Group Manager
	if createdBy, ok := asset.ResourceData["description"].(string); ok {
		if strings.Contains(createdBy, "instanceGroupManagers") {
			return true
		}
	}

	return false
}

// resolveDependencies establishes edges between resources (e.g. Subnet -> VPC, VM -> Subnet).
func (g *Graph) resolveDependencies(node *Node) {
	data := node.Asset.ResourceData

	// Subnet -> Network
	if node.Asset.AssetType == "compute.googleapis.com/Subnetwork" {
		if netURI, ok := data["network"].(string); ok {
			if netNode := g.findNodeByURI(netURI); netNode != nil {
				node.AddDependency(netNode)
			}
		}
	}

	// Firewall -> Network
	if node.Asset.AssetType == "compute.googleapis.com/Firewall" {
		if netURI, ok := data["network"].(string); ok {
			if netNode := g.findNodeByURI(netURI); netNode != nil {
				node.AddDependency(netNode)
			}
		}
	}

	// Route -> Network
	if node.Asset.AssetType == "compute.googleapis.com/Route" {
		if netURI, ok := data["network"].(string); ok {
			if netNode := g.findNodeByURI(netURI); netNode != nil {
				node.AddDependency(netNode)
			}
		}
	}

	// VM Instance -> Subnet / Network
	if node.Asset.AssetType == "compute.googleapis.com/Instance" {
		if ifaces, ok := data["networkInterfaces"].([]interface{}); ok {
			for _, ifaceRaw := range ifaces {
				if iface, ok := ifaceRaw.(map[string]interface{}); ok {
					if subnetworkURI, ok := iface["subnetwork"].(string); ok {
						if subNode := g.findNodeByURI(subnetworkURI); subNode != nil {
							node.AddDependency(subNode)
						} else if strings.Contains(subnetworkURI, "projects/") {
							// External Shared VPC Subnet not yet in graph -> create external data node!
							extNode := g.createExternalDataNode(subnetworkURI, "compute.googleapis.com/Subnetwork")
							node.AddDependency(extNode)
						}
					}
				}
			}
		}
	}

	// GKE NodePool -> GKE Cluster
	if node.Asset.AssetType == "container.googleapis.com/NodePool" {
		if clusterURI, ok := data["cluster_id"].(string); ok {
			if clusterNode := g.findNodeByURI(clusterURI); clusterNode != nil {
				node.AddDependency(clusterNode)
			}
		}
	}
}

func (g *Graph) findNodeByURI(uri string) *Node {
	for id, node := range g.Nodes {
		if id == uri || strings.HasSuffix(id, uri) || strings.HasSuffix(uri, node.Asset.DisplayName) {
			return node
		}
	}
	return nil
}

func (g *Graph) createExternalDataNode(uri string, assetType string) *Node {
	parts := strings.Split(uri, "/")
	projID := ""
	for i, p := range parts {
		if p == "projects" && i+1 < len(parts) {
			projID = parts[i+1]
			break
		}
	}

	name := parts[len(parts)-1]
	asset := &discovery.Asset{
		Name:         uri,
		AssetType:    assetType,
		ProjectID:    projID,
		DisplayName:  name,
		ResourceData: make(map[string]interface{}),
	}

	extNode := NewNode(asset)
	extNode.IsExternal = true
	g.Nodes[uri] = extNode
	return extNode
}

// NodesByDomain returns active non-dynamic nodes grouped by functional domain.
func (g *Graph) NodesByDomain() map[string][]*Node {
	domains := make(map[string][]*Node)
	for _, node := range g.Nodes {
		if node.IsDynamic {
			continue // Skip ephemeral autoscaled instances
		}
		domain := node.Asset.Domain()
		domains[domain] = append(domains[domain], node)
	}
	return domains
}

// TopologicalSort returns a valid order of non-dynamic nodes respecting dependencies.
func (g *Graph) TopologicalSort() ([]*Node, error) {
	inDegree := make(map[string]int)
	for _, node := range g.Nodes {
		if node.IsDynamic {
			continue
		}
		inDegree[node.ID] = len(node.Dependencies)
	}

	var queue []*Node
	for _, node := range g.Nodes {
		if node.IsDynamic {
			continue
		}
		if inDegree[node.ID] == 0 {
			queue = append(queue, node)
		}
	}

	var sorted []*Node
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		sorted = append(sorted, curr)

		for _, dep := range curr.Dependents {
			if dep.IsDynamic {
				continue
			}
			inDegree[dep.ID]--
			if inDegree[dep.ID] == 0 {
				queue = append(queue, dep)
			}
		}
	}

	// Verify cycle-free
	activeCount := 0
	for _, node := range g.Nodes {
		if !node.IsDynamic {
			activeCount++
		}
	}

	if len(sorted) != activeCount {
		return sorted, fmt.Errorf("dependency cycle detected among resources")
	}

	return sorted, nil
}
