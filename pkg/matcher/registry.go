package matcher

// BuiltInOfficialModules provides verified specs for official Google Cloud modules.
func BuiltInOfficialModules() []*ModuleSpec {
	return []*ModuleSpec{
		// 1. Google Cloud Network Module
		{
			Name:                 "vpc",
			Source:               "terraform-google-modules/network/google",
			Version:              "~> 9.0",
			PrimaryResourceType:  "google_compute_network",
			InternalResourceName: "network",
			FieldMappings: map[string]string{
				"name":         "network_name",
				"routing_mode": "routing_mode",
				"description":  "description",
			},
			BundledChildren: []ChildRule{
				{
					ResourceType:         "google_compute_subnetwork",
					InternalResourceName: "subnets",
					MapToInputVar:        "subnets",
					IndexKeyPattern:      "${region}/${name}",
					FieldMappings: map[string]string{
						"name":                     "subnet_name",
						"ip_cidr_range":            "subnet_ip",
						"region":                   "subnet_region",
						"description":              "subnet_desc",
						"private_ip_google_access": "subnet_private_access",
						"flow_sampling":            "subnet_flow_logs",
					},
				},
				{
					ResourceType:         "google_compute_route",
					InternalResourceName: "routes",
					MapToInputVar:        "routes",
					IndexKeyPattern:      "${name}",
					FieldMappings: map[string]string{
						"name":             "name",
						"dest_range":       "destination_range",
						"next_hop_gateway": "next_hop_internet",
						"priority":         "priority",
					},
				},
			},
		},

		// 2. Google Cloud GKE Module
		{
			Name:                 "gke",
			Source:               "terraform-google-modules/kubernetes-engine/google",
			Version:              "~> 31.0",
			PrimaryResourceType:  "google_container_cluster",
			InternalResourceName: "cluster",
			FieldMappings: map[string]string{
				"name":     "name",
				"location": "regional",
				"network":  "network",
			},
			BundledChildren: []ChildRule{
				{
					ResourceType:         "google_container_node_pool",
					InternalResourceName: "node_pools",
					MapToInputVar:        "node_pools",
					IndexKeyPattern:      "${name}",
					FieldMappings: map[string]string{
						"name":         "name",
						"machine_type": "machine_type",
						"node_count":   "initial_node_count",
						"min_count":    "min_count",
						"max_count":    "max_count",
					},
				},
			},
		},

		// 3. Google Cloud Storage Bucket Module
		{
			Name:                 "gcs_buckets",
			Source:               "terraform-google-modules/cloud-storage/google",
			Version:              "~> 6.0",
			PrimaryResourceType:  "google_storage_bucket",
			InternalResourceName: "bucket",
			FieldMappings: map[string]string{
				"name":          "names",
				"location":      "location",
				"storage_class": "storage_class",
			},
		},

		// 4. Google Cloud SQL (PostgreSQL / MySQL)
		{
			Name:                 "cloud_sql",
			Source:               "terraform-google-modules/sql-db/google//modules/postgresql",
			Version:              "~> 20.0",
			PrimaryResourceType:  "google_sql_database_instance",
			InternalResourceName: "instance",
			FieldMappings: map[string]string{
				"name":             "name",
				"database_version": "database_version",
				"region":           "region",
				"tier":             "tier",
			},
		},
	}
}
