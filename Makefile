SHELL := /bin/bash
GO ?= go
OUT ?= out

GO_PKGS := ./...
SH_FILES := $(shell find os -name '*.sh' 2>/dev/null)

.DEFAULT_GOAL := help

.PHONY: help
help:
	@grep -hE '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: ## Compile all Go binaries into out/
	@mkdir -p $(OUT)
	$(GO) build -trimpath -o $(OUT)/ $(GO_PKGS)

.PHONY: test
test: ## Run Go tests with race detector
	$(GO) test -race -cover $(GO_PKGS)

.PHONY: vet
vet: ## Run go vet
	$(GO) vet $(GO_PKGS)

.PHONY: fmt
fmt: ## Rewrite Go sources with gofmt
	gofmt -w $$(git ls-files '*.go')

.PHONY: fmt-check
fmt-check: ## Fail if any Go source is unformatted
	@unformatted=$$(gofmt -l $$(git ls-files '*.go')); \
	if [ -n "$$unformatted" ]; then \
		echo "unformatted files:"; echo "$$unformatted"; exit 1; \
	fi

.PHONY: shellcheck
shellcheck: ## Lint shell scripts
	@if [ -n "$(SH_FILES)" ]; then shellcheck $(SH_FILES); fi

.PHONY: lint
lint: vet fmt-check shellcheck ## Run every static check

.PHONY: image
image: ## Build the bootable image (requires root)
	os/build.sh

.PHONY: smoke
smoke: ## Boot the built image under QEMU
	os/test/smoke.sh

.PHONY: clean
clean: ## Remove build output
	rm -rf $(OUT) build cache
