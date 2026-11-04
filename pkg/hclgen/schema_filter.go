package hclgen

import (
	"strings"
)

var (
	// Server-computed read-only attributes that must be stripped to prevent false drift
	computedBlacklist = map[string]bool{
		"id":                       true,
		"self_link":                true,
		"selfLink":                 true,
		"etag":                     true,
		"fingerprint":              true,
		"creation_timestamp":       true,
		"creationTimestamp":        true,
		"project_number":           true,
		"projectNumber":            true,
		"generation":               true,
		"metastore_version":        true,
		"gateway_address":          true,
		"gatewayAddress":           true,
		"label_fingerprint":        true,
		"labelFingerprint":         true,
		"metadata_fingerprint":     true,
		"metadataFingerprint":      true,
		"current_operation":        true,
		"operation":                true,
		"status":                   true,
		"cpu_platform":             true,
		"cpuPlatform":              true,
		"instance_id":              true,
		"instanceId":               true,
	}
)

// FilterAttributes cleans raw GCP JSON properties for Terraform code generation.
func FilterAttributes(raw map[string]interface{}) map[string]interface{} {
	clean := make(map[string]interface{})

	for k, v := range raw {
		if computedBlacklist[k] {
			continue // Skip read-only noise
		}

		// Normalize string values (e.g. URI prefixes)
		if strVal, ok := v.(string); ok {
			clean[k] = normalizeValue(k, strVal)
		} else if mapVal, ok := v.(map[string]interface{}); ok {
			cleanSub := FilterAttributes(mapVal)
			if len(cleanSub) > 0 {
				clean[k] = cleanSub
			}
		} else if v != nil {
			clean[k] = v
		}
	}

	return clean
}

func normalizeValue(key, val string) string {
	// Normalize fully qualified machine type URI to short name
	if key == "machine_type" || key == "machineType" {
		parts := strings.Split(val, "/")
		if len(parts) > 0 {
			return parts[len(parts)-1]
		}
	}
	return val
}
