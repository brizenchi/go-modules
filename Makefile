.PHONY: help test test-race tidy tidy-check fmt fmt-check vet build purity-check lint vuln list verify-templates

MODULES := \
	foundation/slog \
	foundation/jwt \
	foundation/ossx \
	foundation/httpresp \
	foundation/ginx \
	foundation/config \
	foundation/httpx \
	foundation/tracing \
	foundation/pgx \
	foundation/randx \
	foundation/rdx \
	foundation/resilience \
	modules/auth \
	modules/billing \
	modules/email \
	modules/referral \
	modules/user \
	stacks/saascore \
	templates/quickstart

help: ## Show targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

list: ## List all modules
	@printf '%s\n' $(MODULES)

verify-templates: ## verify backend and frontend starter templates
	@echo "==> templates/quickstart"
	@(cd templates/quickstart && go test ./... && go build ./...)
	@echo "==> templates/quickstart-nextjs"
	@(cd templates/quickstart-nextjs && npm run verify)

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
		had_sum=0; \
		if [ -f $$m/go.sum ]; then \
			cp $$m/go.sum $$m/go.sum.bak; \
			had_sum=1; \
		fi; \
		(cd $$m && go mod tidy) || { \
			mv $$m/go.mod.bak $$m/go.mod; \
			if [ $$had_sum -eq 1 ]; then mv $$m/go.sum.bak $$m/go.sum; else rm -f $$m/go.sum; fi; \
			exit 1; \
		}; \
		mod_dirty=0; \
		sum_dirty=0; \
		if ! diff -q $$m/go.mod $$m/go.mod.bak >/dev/null 2>&1; then mod_dirty=1; fi; \
		if [ $$had_sum -eq 1 ]; then \
			if ! diff -q $$m/go.sum $$m/go.sum.bak >/dev/null 2>&1; then sum_dirty=1; fi; \
		elif [ -f $$m/go.sum ]; then \
			sum_dirty=1; \
		fi; \
		if [ $$mod_dirty -eq 1 ] || [ $$sum_dirty -eq 1 ]; then \
			echo "✗ $$m/go.mod or go.sum is not tidy"; \
			diff -u $$m/go.mod.bak $$m/go.mod || true; \
			if [ $$had_sum -eq 1 ] && [ $$sum_dirty -eq 1 ]; then \
				diff -u $$m/go.sum.bak $$m/go.sum || true; \
			elif [ $$had_sum -eq 0 ] && [ $$sum_dirty -eq 1 ]; then \
				echo "new $$m/go.sum would be created by go mod tidy"; \
			fi; \
			mv $$m/go.mod.bak $$m/go.mod; \
			if [ $$had_sum -eq 1 ]; then mv $$m/go.sum.bak $$m/go.sum; else rm -f $$m/go.sum; fi; \
			exit 1; \
		fi; \
		mv $$m/go.mod.bak $$m/go.mod; \
		if [ $$had_sum -eq 1 ]; then mv $$m/go.sum.bak $$m/go.sum; else rm -f $$m/go.sum; fi; \
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
