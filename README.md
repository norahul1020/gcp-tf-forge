# `gcp-tf-forge` 🛠️☁️

> **The Enterprise-Grade GCP Infrastructure Importer & Modernizer for Terraform**

[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Report Card](https://goreportcard.com/badge/github.com/GoogleCloudPlatform/gcp-tf-forge)](https://goreportcard.com/report/github.com/GoogleCloudPlatform/gcp-tf-forge)

`gcp-tf-forge` is an open-source CLI engine specifically designed to reverse-engineer Google Cloud infrastructure into clean, maintainable, modular Terraform HCL.

Unlike legacy tools that output hundreds of flat, unmanageable raw resources, `gcp-tf-forge`:
- **Synthesizes resources into Terraform Modules** (Your internal custom modules first, official Google modules second, and clean raw HCL as fallback).
- **Separates configurations into domain-isolated `.auto.tfvars`** (`networking.auto.tfvars`, `compute.auto.tfvars`, `storage.auto.tfvars`).
- **Enforces Zero-Trust Security**: Never reads secret payloads and automatically injects lifecycle shields for database passwords and cloud APIs.
- **Eliminates False Drift**: Reconciles GCP API defaults with Terraform Provider defaults for a verified `0 to add, 0 to change, 0 to destroy` plan.
- **Cleans up after itself**: Generates native Terraform 1.5+ `import {}` blocks, and prunes them automatically once resources are bound to state.

---

## 🚀 Quick Start

### 1. Install
```bash
# Download latest binary (or build from source)
make build
sudo mv bin/gcp-tf-forge /usr/local/bin/
```

### 2. Run Pre-flight Doctor
Check your active GCP authentication, read-only permissions, and hierarchy ancestry:
```bash
gcp-tf-forge doctor --project=my-production-app
```

### 3. Dry-Run Plan & Module Preview
Scan the cloud and see how resources will be mapped into modules without writing any files:
```bash
gcp-tf-forge plan \
  --project=my-production-app \
  --custom-modules=./my-company-modules/
```

### 4. Export Clean Modular Terraform Code
Generate domain-separated HCL files, typed variables, and modern `import {}` blocks:
```bash
gcp-tf-forge export \
  --project=my-production-app \
  --custom-modules=./my-company-modules/ \
  --out=./infra-code/
```

### 5. Verify & Import State
Run non-destructive validation and state binding:
```bash
cd ./infra-code
terraform init
terraform plan
terraform apply
```

### 6. Clean Up Transitional Import Files
Verify 100% state binding and prune all `*_imports.tf` before committing to Git:
```bash
gcp-tf-forge cleanup --dir=./infra-code
git add .
git commit -m "feat(infra): reverse-engineered production infra via gcp-tf-forge"
```

---

## 🏛️ Architecture & Process Flow

```
[ 1. Discovery ] ──▶ [ 2. Graph & Topology ] ──▶ [ 3. Module Reconciliation ] ──▶ [ 4. HCL2 Emitter ] ──▶ [ 5. Verification & Pruning ]
  CAI 1-Shot Scanner     Resolves Subnet->VPC      User Custom Modules             Domain *.auto.tfvars     terraform validate & plan
  Targeted Enricher      Shared VPC Data Sources   Google Official Modules         Lifecycle Shields        Auto-prunes *_imports.tf
  Direct API Fallback    Dynamic Node Filter       Clean Raw Fallback              Terraform 1.5+ imports   Clean Git repository
```

---

## 🛡️ Enterprise Safety Features

- **Zero-Secret Payload Guarantee**: Strictly queries Secret Manager metadata (`roles/secretmanager.viewer`). Never queries secret versions or payloads (`roles/secretmanager.secretAccessor` is blocked).
- **Database Password Shield**: Injects `lifecycle { ignore_changes = [password] }` to ensure production database passwords are never overwritten.
- **API Destruction Shield**: Injects `disable_on_destroy = false` and `prevent_destroy = true` on `google_project_service`.
- **Shared VPC Awareness**: External host project subnets are emitted as `data` sources, preventing cross-project state pollution.
- **CIS GCP Foundations Linter**: Automatically produces `COMPLIANCE_AUDIT.md` scoring the infrastructure against CIS Benchmark v3.0.

---

## 📄 Documentation

For deep technical details, see:
- [Architecture Specification & RFC](docs/ARCHITECTURE.md)
- [Writing Custom Module Adapters](docs/CUSTOM_MODULE_GUIDE.md)
- [CIS GCP Compliance Benchmark Guide](docs/COMPLIANCE_GUIDE.md)

---

## 🤝 Contributing

Contributions are welcome! Please see our [Contributing Guide](CONTRIBUTING.md) and [Code of Conduct](CODE_OF_CONDUCT.md).

## ⚖️ License

Distributed under the Apache 2.0 License. See `LICENSE` for more information.
