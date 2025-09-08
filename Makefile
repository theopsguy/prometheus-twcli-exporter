.PHONY: build check clean crossbuild crossbuild-tarballs golangci-lint lint promu release tarball test

.DEFAULT_GOAL := build

PROJECTNAME ?= prometheus-twcli-exporter

PREFIX ?= $(shell pwd)
BIN_DIR ?= $(shell pwd)

GO ?= go
GOHOSTOS ?= $(shell $(GO) env GOHOSTOS)
GOHOSTARCH ?= $(shell $(GO) env GOHOSTARCH)
GO_BUILD_PLATFORM ?= $(GOHOSTOS)-$(GOHOSTARCH)
FIRST_GOPATH := $(firstword $(subst :, ,$(shell $(GO) env GOPATH)))

VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo "dev")
REVISION ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BRANCH ?= $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")

PROMU_VERSION ?= 0.17.0
PROMU_URL := https://github.com/prometheus/promu/releases/download/v$(PROMU_VERSION)/promu-$(PROMU_VERSION).$(GO_BUILD_PLATFORM).tar.gz
PROMU := $(FIRST_GOPATH)/bin/promu

GOLANGCI_LINT_OPTS ?=
GOLANGCI_LINT_VERSION ?= v2.4.0
GOLANGCI_LINT_URL := https://raw.githubusercontent.com/golangci/golangci-lint/$(GOLANGCI_LINT_VERSION)/install.sh
GOLANGCI_LINT := $(FIRST_GOPATH)/bin/golangci-lint

lint: golangci-lint
	@echo "Running golangci-lint..."
	$(GOLANGCI_LINT) run --timeout=5m

build: promu
	$(PROMU) build --prefix $(PREFIX)

tarball:
	$(PROMU) tarball --prefix $(PREFIX) $(BIN_DIR)

crossbuild: promu
	$(PROMU) crossbuild

crossbuild-tarballs: crossbuild
	$(PROMU) crossbuild tarballs

release: crossbuild-tarballs
	$(PROMU) release .tarballs

clean: ## Clean build artifacts and temporary files
	@echo "Cleaning build artifacts..."
	$(GO) clean -i ./...
	rm -rf .build/
	rm -rf .tarballs/
	rm -f $(PROJECTNAME)
	rm -f $(PROJECTNAME)-*.tar.gz

test:
	$(GO) test -timeout 5m -json -v ./... | go tool gotestfmt

check: lint test

promu:
	@if [ ! -f $(PROMU) ]; then \
		echo "Downloading promu..."; \
		PROMU_TMP=$$(mktemp -d); \
		if curl -fsSL $(PROMU_URL) | tar -xz -C "$$PROMU_TMP"; then \
			mkdir -p "$(FIRST_GOPATH)/bin"; \
			cp "$$PROMU_TMP/promu-$(PROMU_VERSION).$(GO_BUILD_PLATFORM)/promu" "$(FIRST_GOPATH)/bin/promu"; \
			chmod +x "$(PROMU)"; \
			rm -r "$$PROMU_TMP"; \
			echo "promu downloaded to $(FIRST_GOPATH)/bin/promu"; \
		else \
			echo "Failed to download promu"; \
			rm -r "$$PROMU_TMP"; \
			exit 1; \
		fi; \
	fi

golangci-lint:
	@if [ ! -f $(GOLANGCI_LINT) ]; then \
		echo "Downloading golangci-lint..."; \
		curl -sfL $(GOLANGCI_LINT_URL) \
		| sed -e '/install -d/d' \
		| sh -s -- -b $(FIRST_GOPATH)/bin $(GOLANGCI_LINT_VERSION); \
		if [ ! -f $(GOLANGCI_LINT) ]; then \
			echo "Failed to download golangci-lint"; \
			exit 1; \
		else \
			echo "golangci-lint downloaded to $(GOLANGCI_LINT)"; \
		fi; \
	fi
