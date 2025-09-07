.DEFAULT_GOAL := build

PROJECTNAME ?= prometheus-twcli-exporter

GO           ?= go
PREFIX       ?= $(shell pwd)
BIN_DIR      ?= $(shell pwd)
VERSION      := $(shell cat VERSION)
FIRST_GOPATH := $(firstword $(subst :, ,$(shell $(GO) env GOPATH)))

GO_BUILD_PLATFORM ?= linux-amd64

PROMU_VERSION ?= 0.17.0
PROMU_URL     := https://github.com/prometheus/promu/releases/download/v$(PROMU_VERSION)/promu-$(PROMU_VERSION).$(GO_BUILD_PLATFORM).tar.gz

PROMU := $(FIRST_GOPATH)/bin/promu

.PHONY:fmt vet build tarball clean test promu
GOLANGCI_LINT_OPTS ?=
GOLANGCI_LINT_VERSION ?= v2.4.0
GOLANGCI_LINT_URL := https://raw.githubusercontent.com/golangci/golangci-lint/$(GOLANGCI_LINT_VERSION)/install.sh
GOLANGCI_LINT := $(FIRST_GOPATH)/bin/golangci-lint

lint: golangci-lint
	@echo "Running golangci-lint..."
	$(GOLANGCI_LINT) run --timeout=5m

fmt:
	$(GO) fmt ./...
vet: fmt
	$(GO) vet ./...
build: vet promu
	$(PROMU) build --prefix $(PREFIX)
tarball:
	$(PROMU) tarball --prefix $(PREFIX) $(BIN_DIR)
crossbuild: vet promu
	$(PROMU) crossbuild
crossbuild-tarballs: crossbuild
	$(PROMU) crossbuild tarballs
release: crossbuild-tarballs
	$(PROMU) release .tarballs
clean:
	$(GO) clean
	rm $(PROJECTNAME)-$(VERSION).$(GO_BUILD_PLATFORM).tar.gz
test:
	$(GO) test -json -v ./... | go tool gotestfmt
promu:
	$(eval PROMU_TMP := $(shell mktemp -d))
	curl -s -L $(PROMU_URL) | tar -xvzf - -C $(PROMU_TMP)
	mkdir -p $(FIRST_GOPATH)/bin
	cp $(PROMU_TMP)/promu-$(PROMU_VERSION).$(GO_BUILD_PLATFORM)/promu $(FIRST_GOPATH)/bin/promu
	rm -r $(PROMU_TMP)

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
