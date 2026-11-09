.DEFAULT_GOAL := help

GO := go
PYTHON := python3
FLAREBYTE := gh flarebyte
export GOTOOLCHAIN := local
export GOCACHE := $(CURDIR)/.gocache
export GOMODCACHE := $(CURDIR)/.gomodcache

.PHONY: help check-tools install-tools-help config format lint test test-go test-scripts smoke coverage review build build-go build-dist build-release doc-design validate-design

config: ## Validate the flarebyte configuration.
	$(FLAREBYTE) config validate

format: ## Format Go sources.
	$(FLAREBYTE) format

lint: config ## Run Go static checks.
	$(FLAREBYTE) lint

test: test-go test-scripts smoke ## Run Go, release-tooling, and CLI smoke checks.

test-go: ## Run configured Go tests.
	$(FLAREBYTE) test

test-scripts: ## Test release preflight and artifact staging offline.
	$(PYTHON) -m unittest discover -s scripts -p 'test_*.py'

smoke: build-go ## Check help and scaffold failure behavior.
	$(PYTHON) scripts/smoke.py ./gh-arrow

coverage: ## Report Go coverage.
	$(FLAREBYTE) cov

# Formatting must finish before checks, including under make -j.
review: ## Format, validate, lint, and test locally.
	$(MAKE) format
	$(MAKE) lint test validate-design

build: build-dist ## Build configured platform artifacts and checksums.

build-go: ## Build the local extension executable at the repository root.
	$(GO) build -o gh-arrow ./cmd/gh-arrow

build-dist: config ## Build artifacts through the installed flarebyte extension.
	$(FLAREBYTE) build

build-release: config ## Stage verified release assets; requires a release version.
	$(PYTHON) scripts/build_release.py

validate-design: ## Validate specification CSV structure and references.
	sh scripts/validate-design.sh

doc-design: ## Validate and regenerate the specification with flyb.
	sh scripts/doc-design.sh

check-tools: ## Report availability of development tools.
	@sh scripts/check-tools.sh

install-tools-help: ## Show tool setup guidance without installing anything.
	@printf '%s\n' 'Go 1.24+: https://go.dev/doc/install' 'GitHub CLI: https://cli.github.com/' 'gh extension install flarebyte/gh-flarebyte --pin v0.24.0' 'Python 3.11+: https://www.python.org/downloads/' 'flyb (documentation generation): install the flyb CLI.'

help: ## Show available targets.
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z0-9_.-]+:.*## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)
