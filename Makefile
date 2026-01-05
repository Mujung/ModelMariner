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
	@test -z "$$(gofmt -l .)" || (echo "unformatted files:"; gofmt -l .; exit 1)

.PHONY: traces
traces: ## Regenerate the synthetic trace corpus (deterministic).
	go run testdata/gen_traces.go > $(TRACES)

.PHONY: report
report: build ## Analyze the sample corpus and write artifacts to testdata/output.
	$(BIN_DIR)/$(BINARY) analyze --traces $(TRACES) --policy $(POLICIES) --out $(OUT_DIR) --format text

.PHONY: validate
validate: build ## Validate the sample corpus.
	$(BIN_DIR)/$(BINARY) validate --traces $(TRACES)

.PHONY: dashboard
dashboard: ## Build and test the TypeScript dashboard.
	cd dashboard && npm install && npm test

.PHONY: demo
demo: report ## Run the full pipeline then render the dashboard overview.
	cd dashboard && npm install --silent && npm run build --silent && \
		node dist/dashboard.js ../$(OUT_DIR)/report.json overview

.PHONY: ci
ci: fmt-check vet test dashboard ## Everything CI runs.

.PHONY: clean
clean: ## Remove build artifacts (keeps committed testdata).
	rm -rf $(BIN_DIR) dashboard/dist dashboard/node_modules
