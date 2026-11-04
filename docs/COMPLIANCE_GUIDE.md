# CIS Google Cloud Foundations Benchmark Guide

`gcp-tf-forge` includes an automated compliance engine that evaluates discovered cloud assets against the **CIS Google Cloud Foundations Benchmark v3.0** during every scan.

---

## Supported CIS Benchmark Controls

| Control ID | Service | Rule Title | Severity | Default Auto-Remediation in Generated Code |
| :--- | :--- | :--- | :--- | :--- |
| **CIS-3.6** | Compute | Ensure no firewall allows 0.0.0.0/0 on port 22 (SSH) | HIGH | Flags rule in `COMPLIANCE_AUDIT.md` and advises restricting source IP range to Google IAP (`35.235.240.0/20`). |
| **CIS-3.7** | Compute | Ensure no firewall allows 0.0.0.0/0 on port 3389 (RDP) | HIGH | Flags rule and advises restricting to internal corporate ranges. |
| **CIS-4.4** | Compute | Ensure OS Login is enabled for Compute instances | MEDIUM | Injects `enable-oslogin = "TRUE"` into instance custom metadata. |
| **CIS-4.10**| Compute | Ensure Shielded VM is enabled | MEDIUM | Injects `shielded_instance_config { enable_secure_boot = true, enable_vtpm = true }`. |
| **CIS-5.1** | Storage | Ensure Uniform Bucket-Level Access is enabled | MEDIUM | Enforces `uniform_bucket_level_access = true` in storage configuration. |
| **CIS-5.2** | Storage | Ensure Public Access Prevention is enforced | HIGH | Enforces `public_access_prevention = "enforced"` in storage configuration. |
| **CIS-6.2** | SQL | Ensure automated backups are enabled on Cloud SQL | HIGH | Enforces `backup_configuration { enabled = true }`. |

---

## Audit Output (`COMPLIANCE_AUDIT.md`)

When running `gcp-tf-forge export`, the tool automatically outputs `COMPLIANCE_AUDIT.md` in the generated infrastructure directory:
```markdown
# GCP Infrastructure Compliance & Security Audit
Target Project: prod-payment-app
Benchmark: CIS Google Cloud Foundations Benchmark v3.0

### Executive Compliance Score: 88%
- ✔ Passed Controls: 14
- ⚠️ Warnings / Recommendations: 2
- ❌ Critical Failures: 0
```
This enables SecOps and Platform teams to review compliance scores prior to committing infrastructure code to Git.
