package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

// DirectCrawler discovers resources via direct gcloud commands when CAI is disabled.
type DirectCrawler struct {
	ProjectID string
}

func NewDirectCrawler(projectID string) *DirectCrawler {
	return &DirectCrawler{ProjectID: projectID}
}

// Crawl queries key Google Cloud services in parallel.
func (c *DirectCrawler) Crawl(ctx context.Context) ([]*Asset, error) {
	var (
		mu     sync.Mutex
		assets []*Asset
		wg     sync.WaitGroup
	)

	services := []struct {
		assetType string
		cmd       []string
	}{
		{
			assetType: "compute.googleapis.com/Network",
			cmd:       []string{"compute", "networks", "list", fmt.Sprintf("--project=%s", c.ProjectID), "--format=json"},
		},
		{
			assetType: "compute.googleapis.com/Subnetwork",
			cmd:       []string{"compute", "networks", "subnets", "list", fmt.Sprintf("--project=%s", c.ProjectID), "--format=json"},
		},
		{
			assetType: "compute.googleapis.com/Firewall",
			cmd:       []string{"compute", "firewall-rules", "list", fmt.Sprintf("--project=%s", c.ProjectID), "--format=json"},
		},
		{
			assetType: "compute.googleapis.com/Instance",
			cmd:       []string{"compute", "instances", "list", fmt.Sprintf("--project=%s", c.ProjectID), "--format=json"},
		},
		{
			assetType: "storage.googleapis.com/Bucket",
			cmd:       []string{"storage", "buckets", "list", fmt.Sprintf("--project=%s", c.ProjectID), "--format=json"},
		},
		{
			assetType: "sqladmin.googleapis.com/Instance",
			cmd:       []string{"sql", "instances", "list", fmt.Sprintf("--project=%s", c.ProjectID), "--format=json"},
		},
		{
			assetType: "iam.googleapis.com/ServiceAccount",
			cmd:       []string{"iam", "service-accounts", "list", fmt.Sprintf("--project=%s", c.ProjectID), "--format=json"},
		},
		{
			assetType: "container.googleapis.com/Cluster",
			cmd:       []string{"container", "clusters", "list", fmt.Sprintf("--project=%s", c.ProjectID), "--format=json"},
		},
	}

	for _, s := range services {
		wg.Add(1)
		go func(assetType string, args []string) {
			defer wg.Done()

			out, err := exec.CommandContext(ctx, "gcloud", args...).Output()
			if err != nil {
				// Non-fatal if a service has 0 resources or is uninitialized
				return
			}

			var items []map[string]interface{}
			if err := json.Unmarshal(out, &items); err != nil {
				return
			}

			mu.Lock()
			defer mu.Unlock()

			for _, item := range items {
				name, _ := item["name"].(string)
				selfLink, _ := item["selfLink"].(string)
				if selfLink == "" {
					selfLink = name
				}

				labels := make(map[string]string)
				if rawLabels, ok := item["labels"].(map[string]interface{}); ok {
					for k, v := range rawLabels {
						if strVal, ok := v.(string); ok {
							labels[k] = strVal
						}
					}
				}

				loc := "global"
				if region, ok := item["region"].(string); ok {
					loc = extractLastSegment(region)
				} else if zone, ok := item["zone"].(string); ok {
					loc = extractLastSegment(zone)
				}

				assets = append(assets, &Asset{
					Name:         selfLink,
					AssetType:    assetType,
					ProjectID:    c.ProjectID,
					Location:     loc,
					DisplayName:  name,
					Labels:       labels,
					ResourceData: item,
				})
			}
		}(s.assetType, s.cmd)
	}

	wg.Wait()
	return assets, nil
}

func extractLastSegment(uri string) string {
	parts := strings.Split(uri, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return uri
}
