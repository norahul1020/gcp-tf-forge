package matcher

// ModuleSpec defines the declarative schema for mapping GCP assets to a Terraform module.
type ModuleSpec struct {
	Name                 string            `yaml:"name"`                   // e.g. "vpc"
	Source               string            `yaml:"source"`                 // e.g. "terraform-google-modules/network/google"
	Version              string            `yaml:"version,omitempty"`      // e.g. "~> 9.0"
	PrimaryResourceType  string            `yaml:"primary_resource"`       // e.g. "google_compute_network"
	InternalResourceName string            `yaml:"internal_resource_name"` // e.g. "network" -> module.vpc.google_compute_network.network
	FieldMappings        map[string]string `yaml:"field_mappings"`         // raw GCP field -> module input var name
	BundledChildren      []ChildRule       `yaml:"bundled_children"`       // Children bundled inside this module
}

// ChildRule defines how child resources (subnets, routes, node pools) map into the parent module.
type ChildRule struct {
	ResourceType         string            `yaml:"resource_type"`          // e.g. "google_compute_subnetwork"
	InternalResourceName string            `yaml:"internal_resource_name"` // e.g. "subnets"
	MapToInputVar        string            `yaml:"map_to_input"`           // e.g. "subnets"
	IndexKeyPattern      string            `yaml:"index_key_pattern"`      // e.g. "${region}/${name}"
	FieldMappings        map[string]string `yaml:"field_mappings"`         // raw -> module var (e.g. name -> subnet_name)
}

// ImportTarget specifies the exact Terraform 1.5+ import address and GCP cloud ID.
type ImportTarget struct {
	ToAddress string // e.g. "module.vpc.google_compute_subnetwork.subnetwork[\"us-central1/subnet-a\"]"
	CloudID   string // e.g. "projects/p/regions/us-central1/subnetworks/subnet-a"
}

// ResolvedModuleCall represents a fully materialized module invocation ready for code generation.
type ResolvedModuleCall struct {
	ModuleName    string
	Source        string
	Version       string
	Domain        string
	Inputs        map[string]interface{}
	ImportTargets []ImportTarget
	ChildResourceIDs []string
}
