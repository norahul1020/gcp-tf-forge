package brownfield

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gcp-tf-forge/pkg/discovery"
)

// TFStatePartial models the necessary parts of a Terraform state JSON file.
type TFStatePartial struct {
	Version   int               `json:"version"`
	Resources []TFStateResource `json:"resources"`
}

type TFStateResource struct {
	Type      string             `json:"type"`
	Name      string             `json:"name"`
	Instances []TFStateInstance  `json:"instances"`
}

type TFStateInstance struct {
	Attributes map[string]interface{} `json:"attributes"`
}

// DiffResult holds the comparative analysis between live cloud and existing state.
type DiffResult struct {
	AlreadyManaged []string // IDs of assets already in state
	Untracked      []string // IDs of assets discovered in cloud but missing from state (ClickOps)
	PhantomState   []string // Resources present in state but deleted in cloud
}

// IngestAndDiff loads an existing state file and categorizes cloud assets.
func IngestAndDiff(stateFilePath string, liveAssets []*discovery.Asset) (*DiffResult, error) {
	result := &DiffResult{
		AlreadyManaged: make([]string, 0),
		Untracked:      make([]string, 0),
		PhantomState:   make([]string, 0),
	}

	stateBytes, err := os.ReadFile(stateFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed reading state file: %w", err)
	}

	var state TFStatePartial
	if err := json.Unmarshal(stateBytes, &state); err != nil {
		return nil, fmt.Errorf("failed parsing state JSON: %w", err)
	}

	managedCloudIDs := make(map[string]bool)
	for _, res := range state.Resources {
		for _, inst := range res.Instances {
			if id, ok := inst.Attributes["id"].(string); ok && id != "" {
				managedCloudIDs[id] = true
			}
			if selfLink, ok := inst.Attributes["self_link"].(string); ok && selfLink != "" {
				managedCloudIDs[selfLink] = true
			}
			if name, ok := inst.Attributes["name"].(string); ok && name != "" {
				managedCloudIDs[name] = true
			}
		}
	}

	liveMap := make(map[string]bool)
	for _, asset := range liveAssets {
		liveMap[asset.Name] = true
		liveMap[asset.DisplayName] = true

		// Check if asset is already tracked in state
		if managedCloudIDs[asset.Name] || managedCloudIDs[asset.DisplayName] {
			result.AlreadyManaged = append(result.AlreadyManaged, asset.DisplayName)
		} else {
			result.Untracked = append(result.Untracked, asset.DisplayName)
		}
	}

	return result, nil
}

// GenerateDriftReport writes the DRIFT_AND_CLICKOPS_REPORT.md report to the output directory.
func GenerateDriftReport(outputDir, projectID string, diff *DiffResult) error {
	var b strings.Builder
	b.WriteString("# Cloud Drift & ClickOps Asset Report\n")
	b.WriteString(fmt.Sprintf("**Target Project:** %s  \n\n", projectID))

	b.WriteString(fmt.Sprintf("- 📦 **Already Managed in Terraform:** %d\n", len(diff.AlreadyManaged)))
	b.WriteString(fmt.Sprintf("- ⚠️ **Untracked / ClickOps Assets Discovered:** %d\n", len(diff.Untracked)))
	b.WriteString(fmt.Sprintf("- 👻 **Phantom Resources (In State, Missing in Cloud):** %d\n\n", len(diff.PhantomState)))

	if len(diff.Untracked) > 0 {
		b.WriteString("### Discovered Untracked ClickOps Resources (To be imported):\n")
		for _, item := range diff.Untracked {
			b.WriteString(fmt.Sprintf("- `%s`\n", item))
		}
		b.WriteString("\n")
	}

	if len(diff.AlreadyManaged) > 0 {
		b.WriteString("### Already Tracked Resources (Import blocks skipped to prevent state collision):\n")
		for _, item := range diff.AlreadyManaged {
			b.WriteString(fmt.Sprintf("- `%s`\n", item))
		}
	}

	reportPath := filepath.Join(outputDir, "DRIFT_AND_CLICKOPS_REPORT.md")
	return os.WriteFile(reportPath, []byte(b.String()), 0644)
}
