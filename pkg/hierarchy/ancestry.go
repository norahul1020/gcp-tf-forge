package hierarchy

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// HierarchyType represents the structural organization of the GCP resource.
type HierarchyType string

const (
	TypeStandalone   HierarchyType = "Standalone Project (No Org/Folder)"
	TypeDirectOrg    HierarchyType = "Direct Organization Member"
	TypeNestedFolder HierarchyType = "Nested Folder Hierarchy"
)

// Lineage contains resolved ancestry and scope details.
type Lineage struct {
	ProjectID     string
	Breadcrumb    string
	Type          HierarchyType
	OrgID         string
	FolderIDs     []string
	CAIAvailable  bool
}

// ResolveAncestry queries Cloud Resource Manager to map the ancestry tree.
func ResolveAncestry(ctx context.Context, projectID string) (*Lineage, error) {
	lineage := &Lineage{
		ProjectID:    projectID,
		Breadcrumb:   projectID,
		Type:         TypeStandalone,
		CAIAvailable: true, // Optimistically enabled, verified below
	}

	// Execute gcloud projects get-ancestors to discover hierarchy safely
	cmd := exec.CommandContext(ctx, "gcloud", "projects", "get-ancestors", projectID,
		"--format=value(type,id)")
	out, err := cmd.Output()
	if err != nil {
		// If permission fails or project has no org, treat as standalone safely
		return lineage, nil
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var pathElements []string

	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) == 2 {
			resType, resID := parts[0], parts[1]
			switch resType {
			case "organization":
				lineage.OrgID = resID
				pathElements = append([]string{fmt.Sprintf("Org(%s)", resID)}, pathElements...)
			case "folder":
				lineage.FolderIDs = append(lineage.FolderIDs, resID)
				pathElements = append(pathElements, fmt.Sprintf("Folder(%s)", resID))
			case "project":
				pathElements = append(pathElements, fmt.Sprintf("Project(%s)", resID))
			}
		}
	}

	if len(pathElements) > 0 {
		lineage.Breadcrumb = strings.Join(pathElements, " > ")
	}

	if lineage.OrgID != "" && len(lineage.FolderIDs) > 0 {
		lineage.Type = TypeNestedFolder
	} else if lineage.OrgID != "" {
		lineage.Type = TypeDirectOrg
	} else {
		lineage.Type = TypeStandalone
	}

	// Check if Cloud Asset Inventory API is enabled on this project
	checkCAI := exec.CommandContext(ctx, "gcloud", "services", "list",
		fmt.Sprintf("--project=%s", projectID),
		"--filter=config.name:cloudasset.googleapis.com",
		"--format=value(config.name)")
	caiOut, err := checkCAI.Output()
	if err == nil && strings.TrimSpace(string(caiOut)) != "" {
		lineage.CAIAvailable = true
	} else {
		lineage.CAIAvailable = false
	}

	return lineage, nil
}
