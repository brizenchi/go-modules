.PHONY: help test test-race tidy tidy-check fmt fmt-check vet build purity-check lint vuln list verify-templates

PACKAGE_ROOTS := foundation modules stacks
GO_PACKAGES := ./foundation/... ./modules/... ./stacks/...
GO := GOWORK=off GOCACHE=$(CURDIR)/.cache/go-build GOMODCACHE=$(CURDIR)/.cache/gomod go

help: ## Show targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

list: ## List tracked Go package roots
	@printf '%s\n' $(PACKAGE_ROOTS)

verify-templates: ## verify backend and frontend starter templates
	@echo "==> templates/quickstart"
	@(cd templates/quickstart && go test ./... && go build ./...)
	@echo "==> templates/quickstart-nextjs"
	@(cd templates/quickstart-nextjs && npm run verify)

build: ## go build ./... from the repo root module
	@$(GO) build $(GO_PACKAGES)

test: ## go test ./... from the repo root module
	@$(GO) test $(GO_PACKAGES)

test-race: ## go test -race ./... from the repo root module
	@$(GO) test -race $(GO_PACKAGES)

tidy: ## go mod tidy for the repo root module
	@$(GO) mod tidy

tidy-check: ## fail if go.mod / go.sum would change after `go mod tidy`
	@cp go.mod go.mod.bak; \
	had_sum=0; \
	if [ -f go.sum ]; then \
		cp go.sum go.sum.bak; \
		had_sum=1; \
	fi; \
	$(GO) mod tidy || { \
		mv go.mod.bak go.mod; \
		if [ $$had_sum -eq 1 ]; then mv go.sum.bak go.sum; else rm -f go.sum; fi; \
		exit 1; \
	}; \
	mod_dirty=0; \
	sum_dirty=0; \
	if ! diff -q go.mod go.mod.bak >/dev/null 2>&1; then mod_dirty=1; fi; \
	if [ $$had_sum -eq 1 ]; then \
		if ! diff -q go.sum go.sum.bak >/dev/null 2>&1; then sum_dirty=1; fi; \
	elif [ -f go.sum ]; then \
		sum_dirty=1; \
	fi; \
	if [ $$mod_dirty -eq 1 ] || [ $$sum_dirty -eq 1 ]; then \
		echo "✗ go.mod or go.sum is not tidy"; \
		diff -u go.mod.bak go.mod || true; \
		if [ $$had_sum -eq 1 ] && [ $$sum_dirty -eq 1 ]; then \
			diff -u go.sum.bak go.sum || true; \
		elif [ $$had_sum -eq 0 ] && [ $$sum_dirty -eq 1 ]; then \
			echo "new go.sum would be created by go mod tidy"; \
		fi; \
		mv go.mod.bak go.mod; \
		if [ $$had_sum -eq 1 ]; then mv go.sum.bak go.sum; else rm -f go.sum; fi; \
		exit 1; \
	fi; \
	mv go.mod.bak go.mod; \
	if [ $$had_sum -eq 1 ]; then mv go.sum.bak go.sum; else rm -f go.sum; fi; \
	echo "✓ go.mod / go.sum tidy"

fmt: ## gofmt -s -w
	@gofmt -s -w .

fmt-check: ## fail if gofmt would change anything
	@out=$$(gofmt -s -l .); \
	if [ -n "$$out" ]; then \
		echo "✗ files need gofmt:"; \
		echo "$$out"; \
		exit 1; \
	fi; \
	echo "✓ gofmt clean"

vet: ## go vet ./... from the repo root module
	@$(GO) vet $(GO_PACKAGES)

lint: ## golangci-lint over the repo root module
	@command -v golangci-lint >/dev/null 2>&1 || { \
		echo "golangci-lint not installed; see https://golangci-lint.run/usage/install/"; exit 1; }
	@golangci-lint run --config $(CURDIR)/.golangci.yml $(GO_PACKAGES)

vuln: ## govulncheck over the repo root module
	@command -v govulncheck >/dev/null 2>&1 || { \
		echo "govulncheck not installed; run: go install golang.org/x/vuln/cmd/govulncheck@latest"; exit 1; }
	@GOWORK=off GOCACHE=$(CURDIR)/.cache/go-build GOMODCACHE=$(CURDIR)/.cache/gomod govulncheck $(GO_PACKAGES)

purity-check: ## ensure no package imports forbidden host-app paths
	@bad=$$(grep -rE '"github\.com/[^"]+/(internal|pkg/models|pkg/middleware)"' \
		$(PACKAGE_ROOTS) --include='*.go' 2>/dev/null | grep -v _test.go | wc -l | tr -d ' '); \
	if [ "$$bad" != "0" ]; then \
		echo "✗ purity violated: $$bad forbidden imports found"; \
		grep -rE '"github\.com/[^"]+/(internal|pkg/models|pkg/middleware)"' \
			$(PACKAGE_ROOTS) --include='*.go' | grep -v _test.go; \
		exit 1; \
	fi; \
	echo "✓ pkg purity ok"
