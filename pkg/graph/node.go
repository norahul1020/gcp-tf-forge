package graph

import (
	"gcp-tf-forge/pkg/discovery"
)

// Node represents a single GCP resource in the dependency graph.
type Node struct {
	ID           string           // Unique canonical ID
	Asset        *discovery.Asset // Underlying discovered asset
	Dependencies []*Node          // Nodes this node depends on (parents)
	Dependents   []*Node          // Nodes that depend on this node (children)

	// Operational classification flags
	IsExternal  bool   // True if resource belongs to an external project (e.g. Shared VPC Host)
	IsDynamic   bool   // True if resource is an ephemeral autoscaling VM (GKE worker, MIG instance)
	TargetModule string // Assigned module name (e.g. "module.vpc")
}

// NewNode creates a normalized graph node for an asset.
func NewNode(asset *discovery.Asset) *Node {
	return &Node{
		ID:           asset.Name,
		Asset:        asset,
		Dependencies: make([]*Node, 0),
		Dependents:   make([]*Node, 0),
	}
}

// AddDependency links a dependency relationship: this -> dependsOn
func (n *Node) AddDependency(dependsOn *Node) {
	for _, existing := range n.Dependencies {
		if existing.ID == dependsOn.ID {
			return
		}
	}
	n.Dependencies = append(n.Dependencies, dependsOn)
	dependsOn.Dependents = append(dependsOn.Dependents, n)
}
