package hclgen

import (
	"strings"
)

// InjectsLifecycleShields returns HCL lifecycle blocks to safeguard production assets.
func InjectsLifecycleShields(resourceType string) string {
	switch resourceType {
	case "google_sql_user", "google_sql_database_instance":
		return `  lifecycle {
    ignore_changes = [password]
  }
`
	case "google_storage_bucket", "google_bigquery_dataset", "google_kms_crypto_key":
		return `  lifecycle {
    prevent_destroy = true
  }
`
	case "google_project_service":
		return `  disable_on_destroy         = false
  disable_dependent_services = false

  lifecycle {
    prevent_destroy = true
  }
`
	default:
		if strings.Contains(resourceType, "database") || strings.Contains(resourceType, "disk") {
			return `  lifecycle {
    prevent_destroy = true
  }
`
		}
		return ""
	}
}
