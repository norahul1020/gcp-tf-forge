# Contributing to `gcp-tf-forge`

Thank you for your interest in contributing to `gcp-tf-forge`! We welcome contributions from developers, architects, and open-source enthusiasts.

---

## 🛠️ Development Setup

### Prerequisites
- **Go**: 1.22 or higher
- **Terraform**: 1.5.0 or higher
- **Docker**: (Optional, for running local GCP emulator integration tests)

### Getting the Code
```bash
git clone https://github.com/norahul1020/gcp-tf-forge.git
cd gcp-tf-forge
make build
make test
```

---

## 🧪 Testing Guidelines

Before submitting a Pull Request, ensure that all tests pass with the race detector enabled:
```bash
make test
```

If you are contributing a new module adapter or resource normalizer:
1. Include unit tests in `pkg/matcher/` or `pkg/hclgen/`.
2. Ensure test coverage remains above 75%.
3. Verify that `make lint` passes.

---

## 📐 Coding Standards & Conventions

1. **Idiomatic Go**: Follow [Effective Go](https://golang.org/doc/effective_go) and [Google Go Style Guide](https://google.github.io/styleguide/go/).
2. **Error Handling**: Do not panic in library packages (`pkg/`). Return typed errors wrapped with context (`fmt.Errorf("failed doing x: %w", err)`).
3. **Zero Secret Leaks**: Never log, query, or store sensitive payloads (passwords, private keys, secret values). Always use metadata-only inspection and lifecycle shields.

---

## 🚀 Submitting a Pull Request (PR)

1. Fork the repository and create your feature branch:
   ```bash
   git checkout -b feat/my-new-feature
   ```
2. Commit your changes following [Conventional Commits](https://www.conventionalcommits.org/):
   - `feat(matcher): add Cloud Run v2 module adapter`
   - `fix(hclgen): resolve machine_type URI normalization`
3. Push to your branch and open a PR with a description of the changes and test results.

---

## 📄 License
By contributing to `gcp-tf-forge`, you agree that your contributions will be licensed under the [Apache 2.0 License](LICENSE).
