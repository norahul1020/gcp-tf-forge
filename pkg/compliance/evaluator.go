package compliance

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gcp-tf-forge/pkg/discovery"
)

// Finding represents an individual compliance check result.
type Finding struct {
	ControlID   string // e.g. "CIS-3.6"
	ResourceName string
	Severity    string // "HIGH", "MEDIUM", "LOW"
	Status      string // "PASS", "WARN", "FAIL"
	Description string
	Remediation string
}

// Evaluator assesses discovered assets against the CIS Google Cloud Foundations Benchmark.
type Evaluator struct {
	Findings []*Finding
}

// NewEvaluator creates a new compliance evaluator.
func NewEvaluator() *Evaluator {
	return &Evaluator{
		Findings: make([]*Finding, 0),
	}
}

// Evaluate runs all CIS rules against the provided assets.
func (e *Evaluator) Evaluate(assets []*discovery.Asset) {
	for _, asset := range assets {
		switch asset.AssetType {
		case "storage.googleapis.com/Bucket":
			e.checkStorageBucket(asset)
		case "compute.googleapis.com/Instance":
			e.checkComputeInstance(asset)
		case "compute.googleapis.com/Firewall":
			e.checkFirewall(asset)
		case "sqladmin.googleapis.com/Instance":
			e.checkCloudSQL(asset)
		}
	}
}

func (e *Evaluator) checkStorageBucket(asset *discovery.Asset) {
	data := asset.ResourceData

	// CIS 5.2: Ensure that Public Access Prevention is enforced
	pap, _ := data["publicAccessPrevention"].(string)
	if pap == "enforced" {
		e.Findings = append(e.Findings, &Finding{
			ControlID:    "CIS-5.2",
			ResourceName: asset.DisplayName,
			Severity:     "HIGH",
			Status:       "PASS",
			Description:  "Public access prevention is strictly enforced.",
		})
	} else {
		e.Findings = append(e.Findings, &Finding{
			ControlID:    "CIS-5.2",
			ResourceName: asset.DisplayName,
			Severity:     "HIGH",
			Status:       "WARN",
			Description:  "Public access prevention is not enforced (currently inherited or unconfigured).",
			Remediation:  "Set public_access_prevention = \"enforced\" in storage configuration.",
		})
	}
}

func (e *Evaluator) checkComputeInstance(asset *discovery.Asset) {
	data := asset.ResourceData

	// CIS 4.10: Ensure Shielded VM is enabled
	shieldedConfig, ok := data["shieldedInstanceConfig"].(map[string]interface{})
	if ok {
		secureBoot, _ := shieldedConfig["enableSecureBoot"].(bool)
		vtpm, _ := shieldedConfig["enableVtpm"].(bool)
		if secureBoot && vtpm {
			e.Findings = append(e.Findings, &Finding{
				ControlID:    "CIS-4.10",
				ResourceName: asset.DisplayName,
				Severity:     "MEDIUM",
				Status:       "PASS",
				Description:  "Shielded VM with Secure Boot and vTPM is enabled.",
			})
			return
		}
	}

	e.Findings = append(e.Findings, &Finding{
		ControlID:    "CIS-4.10",
		ResourceName: asset.DisplayName,
		Severity:     "MEDIUM",
		Status:       "WARN",
		Description:  "Shielded VM Secure Boot or vTPM is disabled.",
		Remediation:  "Enable shielded_instance_config { enable_secure_boot = true, enable_vtpm = true }.",
	})
}

func (e *Evaluator) checkFirewall(asset *discovery.Asset) {
	data := asset.ResourceData

	// CIS 3.6 / 3.7: Ensure no firewall allows 0.0.0.0/0 on port 22 (SSH) or 3389 (RDP)
	sourceRanges, ok := data["sourceRanges"].([]interface{})
	if !ok {
		return
	}

	hasGlobalSource := false
	for _, sr := range sourceRanges {
		if sr == "0.0.0.0/0" {
			hasGlobalSource = true
			break
		}
	}

	if hasGlobalSource {
		if allowed, ok := data["allowed"].([]interface{}); ok {
			for _, allowItem := range allowed {
				if rule, ok := allowItem.(map[string]interface{}); ok {
					if ports, ok := rule["ports"].([]interface{}); ok {
						for _, p := range ports {
							if p == "22" || p == "3389" {
								e.Findings = append(e.Findings, &Finding{
									ControlID:    "CIS-3.6",
									ResourceName: asset.DisplayName,
									Severity:     "HIGH",
									Status:       "FAIL",
									Description:  fmt.Sprintf("Firewall allows unrestricted 0.0.0.0/0 ingress on sensitive port: %v", p),
									Remediation:  "Restrict source ranges to company VPN or Google Identity-Aware Proxy (35.235.240.0/20).",
								})
								return
							}
						}
					}
				}
			}
		}
	}
}

func (e *Evaluator) checkCloudSQL(asset *discovery.Asset) {
	data := asset.ResourceData
	settings, ok := data["settings"].(map[string]interface{})
	if !ok {
		return
	}

	// CIS 6.2: Automated backups
	backupConfig, ok := settings["backupConfiguration"].(map[string]interface{})
	if ok {
		enabled, _ := backupConfig["enabled"].(bool)
		if enabled {
			e.Findings = append(e.Findings, &Finding{
				ControlID:    "CIS-6.2",
				ResourceName: asset.DisplayName,
				Severity:     "HIGH",
				Status:       "PASS",
				Description:  "Automated daily backups are enabled.",
			})
			return
		}
	}

	e.Findings = append(e.Findings, &Finding{
		ControlID:    "CIS-6.2",
		ResourceName: asset.DisplayName,
		Severity:     "HIGH",
		Status:       "WARN",
		Description:  "Automated backups are currently disabled on this database.",
		Remediation:  "Enable backup_configuration { enabled = true }.",
	})
}

// GenerateReport writes the executive COMPLIANCE_AUDIT.md report to the output directory.
func (e *Evaluator) GenerateReport(outputDir, projectID string) error {
	var b strings.Builder
	b.WriteString("# GCP Infrastructure Compliance & Security Audit\n")
	b.WriteString(fmt.Sprintf("**Target Project:** %s  \n", projectID))
	b.WriteString("**Benchmark:** CIS Google Cloud Foundations Benchmark v3.0  \n\n")

	passCount, warnCount, failCount := 0, 0, 0
	for _, f := range e.Findings {
		switch f.Status {
		case "PASS":
			passCount++
		case "WARN":
			warnCount++
		case "FAIL":
			failCount++
		}
	}

	total := passCount + warnCount + failCount
	score := 100
	if total > 0 {
		score = (passCount * 100) / total
	}

	b.WriteString(fmt.Sprintf("### Executive Compliance Score: **%d%%**\n", score))
	b.WriteString(fmt.Sprintf("- ✔ **Passed Controls:** %d\n", passCount))
	b.WriteString(fmt.Sprintf("- ⚠️ **Warnings / Recommendations:** %d\n", warnCount))
	b.WriteString(fmt.Sprintf("- ❌ **Critical Failures:** %d\n\n", failCount))

	b.WriteString("## Detailed Findings\n\n")
	b.WriteString("| Control | Resource | Severity | Status | Description | Remediation |\n")
	b.WriteString("| :--- | :--- | :--- | :--- | :--- | :--- |\n")

	for _, f := range e.Findings {
		b.WriteString(fmt.Sprintf("| `%s` | `%s` | %s | %s | %s | %s |\n",
			f.ControlID, f.ResourceName, f.Severity, f.Status, f.Description, f.Remediation))
	}

	reportPath := filepath.Join(outputDir, "COMPLIANCE_AUDIT.md")
	return os.WriteFile(reportPath, []byte(b.String()), 0644)
}
