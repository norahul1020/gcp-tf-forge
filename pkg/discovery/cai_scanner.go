package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// CAIScanner queries Google Cloud Asset Inventory to discover assets in 1-shot.
type CAIScanner struct {
	ProjectID string
}

// NewCAIScanner creates a new scanner instance for the specified project.
func NewCAIScanner(projectID string) *CAIScanner {
	return &CAIScanner{ProjectID: projectID}
}

// rawCAIAsset matches the JSON structure returned by gcloud asset search-all-resources
type rawCAIAsset struct {
	Name         string            `json:"name"`
	AssetType    string            `json:"assetType"`
	Project      string            `json:"project"`
	Location     string            `json:"location"`
	DisplayName  string            `json:"displayName"`
	Labels       map[string]string `json:"labels"`
	ParentFullResourceName string  `json:"parentFullResourceName"`
}

// Scan discovers all active assets in the project.
func (s *CAIScanner) Scan(ctx context.Context) ([]*Asset, error) {
	scope := fmt.Sprintf("projects/%s", s.ProjectID)

	cmd := exec.CommandContext(ctx, "gcloud", "asset", "search-all-resources",
		fmt.Sprintf("--scope=%s", scope),
		"--format=json")

	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("cai scanner failed: %w (output: %s)", err, string(out))
	}

	var rawAssets []rawCAIAsset
	if err := json.Unmarshal(out, &rawAssets); err != nil {
		return nil, fmt.Errorf("failed parsing CAI asset response: %w", err)
	}

	var assets []*Asset
	for _, raw := range rawAssets {
		// Clean short name
		nameParts := strings.Split(raw.Name, "/")
		shortName := raw.DisplayName
		if shortName == "" && len(nameParts) > 0 {
			shortName = nameParts[len(nameParts)-1]
		}

		asset := &Asset{
			Name:         raw.Name,
			AssetType:    raw.AssetType,
			ProjectID:    s.ProjectID,
			Location:     raw.Location,
			DisplayName:  shortName,
			Labels:       raw.Labels,
			Parent:       raw.ParentFullResourceName,
			ResourceData: make(map[string]interface{}),
		}

		assets = append(assets, asset)
	}

	return assets, nil
}
