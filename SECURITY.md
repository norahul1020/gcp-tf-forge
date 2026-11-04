# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 0.1.x   | :white_check_mark: |

---

## Zero-Trust Security by Design

`gcp-tf-forge` enforces strict security constraints:
1. **Zero Secret Payload Reads**:
   - The tool requires only `roles/secretmanager.viewer` (metadata).
   - It will **never** request or invoke `secretmanager.versions.access`.
2. **Database Credential Protection**:
   - Database passwords are never emitted in plaintext.
   - `lifecycle { ignore_changes = [password] }` is automatically injected into generated Cloud SQL resources to prevent overwriting live database credentials.
3. **API Destruction Shield**:
   - Injects `disable_on_destroy = false` and `lifecycle { prevent_destroy = true }` on `google_project_service` resources.
4. **Read-Only Pre-flight Check**:
   - `gcp-tf-forge doctor` verifies that credentials have read-only access and alerts if run with excessive privileges (such as Owner).

---

## Reporting a Vulnerability

If you discover a security vulnerability within `gcp-tf-forge`:
- **Please DO NOT open a public GitHub issue.**
- Instead, submit a private advisory through GitHub's Security Advisories tab or email the maintainers directly.
- Include a detailed description of the vulnerability, reproduction steps, and potential impact.
- We will acknowledge receipt within 48 hours and work with you to develop a coordinated disclosure plan.
