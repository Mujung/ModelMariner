# Makefile for modelmariner — offline LLM trace evaluation & routing-policy compiler.
# All targets are self-contained and run fully offline.

BINARY      := modelmariner
CMD         := ./cmd/modelmariner
BIN_DIR     := bin
TRACES      := testdata/fleet.jsonl
POLICIES    := testdata/policies.json
OUT_DIR     := testdata/output
VERSION     := $(shell git describe --tags --always 2>/dev/null || echo 1.0.0)
LDFLAGS     := -s -w -X main.version=$(VERSION)

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help.
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  %-16s %s\n", $$1, $$2}'

.PHONY: build
build: ## Compile the CLI into ./bin.
	@mkdir -p $(BIN_DIR)
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY) $(CMD)

.PHONY: test
test: ## Run all Go unit tests.
	go test ./... -count=1

.PHONY: cover
cover: ## Run tests with a coverage summary.
	go test ./... -cover -count=1

.PHONY: vet
vet: ## Run go vet across the module.
	go vet ./...

.PHONY: fmt
fmt: ## Format all Go sources.
	gofmt -w .

.PHONY: fmt-check
fmt-check: ## Fail if any Go source is unformatted.
