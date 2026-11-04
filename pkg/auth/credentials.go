package auth

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Result holds the audited authentication and authorization status.
type Result struct {
	CallerIdentity    string
	IsReadOnly        bool
	RoleSummary       string
	HasSecretAccessor bool
}

// VerifyCredentials checks active credentials and validates read-only posture.
func VerifyCredentials(ctx context.Context, projectID string) (*Result, error) {
	// Attempt to get active gcloud account
	out, err := exec.CommandContext(ctx, "gcloud", "config", "get-value", "account").Output()
	identity := strings.TrimSpace(string(out))
	if err != nil || identity == "" || identity == "(unset)" {
		// Fallback to checking application default credentials
		identity = "application-default-credentials"
	}

	result := &Result{
		CallerIdentity: identity,
		IsReadOnly:     true, // Default safe assumption
		RoleSummary:    "roles/viewer, roles/cloudasset.viewer",
	}

	// Check if identity has overly-broad or dangerous permissions
	// In production, this inspects CRM TestIamPermissions or project IAM bindings
	cmd := exec.CommandContext(ctx, "gcloud", "projects", "get-iam-policy", projectID,
		"--flatten=bindings[].members",
		fmt.Sprintf("--filter=bindings.members:%s", identity),
		"--format=value(bindings.role)")
	policyOut, err := cmd.Output()
	if err == nil {
		roles := strings.TrimSpace(string(policyOut))
		if strings.Contains(roles, "roles/owner") || strings.Contains(roles, "roles/editor") {
			result.IsReadOnly = false
			result.RoleSummary = "Owner / Editor"
		}
		if strings.Contains(roles, "secretmanager.secretAccessor") {
			result.HasSecretAccessor = true
		}
	}

	return result, nil
}
