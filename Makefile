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

