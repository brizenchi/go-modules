.PHONY: help test test-race tidy tidy-check fmt fmt-check vet build purity-check lint vuln list

MODULES := \
	foundation/slog \
	foundation/jwt \
	foundation/httpresp \
	foundation/ginx \
	foundation/config \
	foundation/pgx \
	foundation/rdx \
	auth \
	billing \
	email \
	referral

help: ## Show targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

list: ## List all modules
	@printf '%s\n' $(MODULES)

build: ## go build ./... in every module
	@for m in $(MODULES); do \
		echo "==> $$m"; \
		(cd $$m && go build ./...) || exit 1; \
	done

test: ## go test ./... in every module
	@for m in $(MODULES); do \
		echo "==> $$m"; \
		(cd $$m && go test ./...) || exit 1; \
	done

test-race: ## go test -race ./... in every module
	@for m in $(MODULES); do \
		echo "==> $$m"; \
		(cd $$m && go test -race ./...) || exit 1; \
	done

tidy: ## go mod tidy in every module
	@for m in $(MODULES); do \
		echo "==> $$m"; \
		(cd $$m && go mod tidy) || exit 1; \
	done

tidy-check: ## fail if go.mod / go.sum would change after `go mod tidy`
	@for m in $(MODULES); do \
		echo "==> $$m"; \
		cp $$m/go.mod $$m/go.mod.bak; \
		cp $$m/go.sum $$m/go.sum.bak 2>/dev/null || true; \
		(cd $$m && go mod tidy) || { mv $$m/go.mod.bak $$m/go.mod; exit 1; }; \
		if ! diff -q $$m/go.mod $$m/go.mod.bak >/dev/null 2>&1 || \
		   ! diff -q $$m/go.sum $$m/go.sum.bak >/dev/null 2>&1; then \
			echo "✗ $$m/go.mod or go.sum is not tidy"; \
			diff -u $$m/go.mod.bak $$m/go.mod || true; \
			mv $$m/go.mod.bak $$m/go.mod; \
			mv $$m/go.sum.bak $$m/go.sum 2>/dev/null || true; \
			exit 1; \
		fi; \
		mv $$m/go.mod.bak $$m/go.mod; \
		mv $$m/go.sum.bak $$m/go.sum 2>/dev/null || true; \
	done; \
	echo "✓ all go.mod files tidy"

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

vet: ## go vet ./... in every module
	@for m in $(MODULES); do \
		echo "==> $$m"; \
		(cd $$m && go vet ./...) || exit 1; \
	done

lint: ## golangci-lint over every module
	@command -v golangci-lint >/dev/null 2>&1 || { \
		echo "golangci-lint not installed; see https://golangci-lint.run/usage/install/"; exit 1; }
	@for m in $(MODULES); do \
		echo "==> $$m"; \
		(cd $$m && golangci-lint run --config $(CURDIR)/.golangci.yml ./...) || exit 1; \
	done

vuln: ## govulncheck over every module
	@command -v govulncheck >/dev/null 2>&1 || { \
		echo "govulncheck not installed; run: go install golang.org/x/vuln/cmd/govulncheck@latest"; exit 1; }
	@for m in $(MODULES); do \
		echo "==> $$m"; \
		(cd $$m && govulncheck ./...) || exit 1; \
	done

purity-check: ## ensure no module imports forbidden host-app paths
	@bad=$$(grep -rE '"github\.com/[^"]+/(internal|pkg/models|pkg/middleware)"' \
		$(MODULES) --include='*.go' 2>/dev/null | grep -v _test.go | wc -l | tr -d ' '); \
	if [ "$$bad" != "0" ]; then \
		echo "✗ purity violated: $$bad forbidden imports found"; \
		grep -rE '"github\.com/[^"]+/(internal|pkg/models|pkg/middleware)"' \
			$(MODULES) --include='*.go' | grep -v _test.go; \
		exit 1; \
	fi; \
	echo "✓ pkg purity ok"
