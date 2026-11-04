package discovery

import (
	"strings"
)

// Asset represents a normalized GCP resource discovered from CAI or direct APIs.
type Asset struct {
	Name         string                 `json:"name"`          // e.g. //compute.googleapis.com/projects/p/regions/r/subnetworks/s
	AssetType    string                 `json:"asset_type"`    // e.g. compute.googleapis.com/Subnetwork
	ProjectID    string                 `json:"project_id"`    // GCP Project ID
	Location     string                 `json:"location"`      // global, region, or zone
	DisplayName  string                 `json:"display_name"`  // Short resource name
	Labels       map[string]string      `json:"labels"`        // Resource labels
	ResourceData map[string]interface{} `json:"resource_data"` // Full live configuration map
	IAMPolicy    map[string]interface{} `json:"iam_policy"`    // Attached IAM policy if present
	Parent       string                 `json:"parent"`        // Ancestry parent URI
}

// Domain categorizes the asset into its functional infrastructure domain.
func (a *Asset) Domain() string {
	switch {
	case strings.HasPrefix(a.AssetType, "compute.googleapis.com/Network"),
		strings.HasPrefix(a.AssetType, "compute.googleapis.com/Subnetwork"),
		strings.HasPrefix(a.AssetType, "compute.googleapis.com/Firewall"),
		strings.HasPrefix(a.AssetType, "compute.googleapis.com/Route"),
		strings.HasPrefix(a.AssetType, "compute.googleapis.com/Router"),
		strings.HasPrefix(a.AssetType, "dns.googleapis.com/"):
		return "networking"

	case strings.HasPrefix(a.AssetType, "container.googleapis.com/"):
		return "gke"

	case strings.HasPrefix(a.AssetType, "compute.googleapis.com/Instance"),
		strings.HasPrefix(a.AssetType, "compute.googleapis.com/InstanceTemplate"),
		strings.HasPrefix(a.AssetType, "compute.googleapis.com/InstanceGroupManager"),
		strings.HasPrefix(a.AssetType, "run.googleapis.com/"):
		return "compute"

	case strings.HasPrefix(a.AssetType, "storage.googleapis.com/"),
		strings.HasPrefix(a.AssetType, "sqladmin.googleapis.com/"),
		strings.HasPrefix(a.AssetType, "redis.googleapis.com/"),
		strings.HasPrefix(a.AssetType, "spanner.googleapis.com/"):
		return "storage"

	case strings.HasPrefix(a.AssetType, "iam.googleapis.com/"),
		strings.HasPrefix(a.AssetType, "cloudresourcemanager.googleapis.com/Project"):
		return "iam"

	case strings.HasPrefix(a.AssetType, "secretmanager.googleapis.com/"),
		strings.HasPrefix(a.AssetType, "cloudkms.googleapis.com/"):
		return "security"

	default:
		return "misc"
	}
}

// TerraformResourceType maps the GCP CAI asset type to the standard HashiCorp Terraform provider resource.
func (a *Asset) TerraformResourceType() string {
	switch a.AssetType {
	case "compute.googleapis.com/Network":
		return "google_compute_network"
	case "compute.googleapis.com/Subnetwork":
		return "google_compute_subnetwork"
	case "compute.googleapis.com/Firewall":
		return "google_compute_firewall"
	case "compute.googleapis.com/Route":
		return "google_compute_route"
	case "compute.googleapis.com/Router":
		return "google_compute_router"
	case "compute.googleapis.com/Instance":
		return "google_compute_instance"
	case "compute.googleapis.com/InstanceTemplate":
		return "google_compute_instance_template"
	case "compute.googleapis.com/InstanceGroupManager":
		return "google_compute_instance_group_manager"
	case "compute.googleapis.com/Disk":
		return "google_compute_disk"
	case "container.googleapis.com/Cluster":
		return "google_container_cluster"
	case "container.googleapis.com/NodePool":
		return "google_container_node_pool"
	case "storage.googleapis.com/Bucket":
		return "google_storage_bucket"
	case "sqladmin.googleapis.com/Instance":
		return "google_sql_database_instance"
	case "iam.googleapis.com/ServiceAccount":
		return "google_service_account"
	case "secretmanager.googleapis.com/Secret":
		return "google_secret_manager_secret"
	case "cloudkms.googleapis.com/KeyRing":
		return "google_kms_key_ring"
	case "cloudkms.googleapis.com/CryptoKey":
		return "google_kms_crypto_key"
	default:
		return ""
	}
}
