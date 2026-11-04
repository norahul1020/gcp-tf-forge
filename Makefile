GO?=$(shell which go 2>/dev/null || echo "/tmp/go/bin/go")
BINARY_NAME=gcp-tf-forge
VERSION?=0.1.0
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "dev")
DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

.PHONY: all build test lint clean help

all: build

build:
	@echo "==> Building $(BINARY_NAME)..."
	$(GO) build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/gcp-tf-forge

test:
	@echo "==> Running unit tests..."
	$(GO) test -v -race -cover ./pkg/...

lint:
	@echo "==> Running linter..."
	golangci-lint run ./... || $(GO) vet ./...

clean:
	@echo "==> Cleaning build artifacts..."
	rm -rf bin/ dist/

help:
	@echo "Available targets:"
	@echo "  build   - Compile binary into bin/$(BINARY_NAME)"
	@echo "  test    - Run unit tests with race detection"
	@echo "  lint    - Run code analysis"
	@echo "  clean   - Remove build artifacts"
