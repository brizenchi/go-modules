.PHONY: help hooks fmt fmt-check vet test test-race frontend-verify secrets-check alerts-check

help: ## Show targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-18s %s\n", $$1, $$2}'

hooks: ## Install local git hooks (needs lefthook; gitleaks recommended)
	@command -v lefthook >/dev/null 2>&1 || { echo "lefthook not installed: brew install lefthook"; exit 1; }
	@lefthook install
	@command -v gitleaks >/dev/null 2>&1 || echo "tip: brew install gitleaks to enable the local secret scan"

fmt: ## gofmt the backend
	@gofmt -s -w backend

fmt-check: ## Fail if backend code is not gofmt-clean
	@out=$$(gofmt -s -l backend); if [ -n "$$out" ]; then echo "✗ files need gofmt:"; echo "$$out"; exit 1; fi; echo "✓ gofmt clean"

vet: ## go vet the backend
	@cd backend && GOWORK=off go vet ./...

test: ## Backend tests
	@cd backend && GOWORK=off go test ./...

test-race: ## Backend tests with the race detector
	@cd backend && GOWORK=off go test -race ./...

frontend-verify: ## Frontend tests, lint, build
	@cd frontend && npm run verify

secrets-check: ## Scan git history for secrets
	@command -v gitleaks >/dev/null 2>&1 || { echo "gitleaks not installed: brew install gitleaks"; exit 1; }
	@gitleaks git . --no-banner --redact --config .gitleaks.toml

alerts-check: ## Validate and unit-test alert rules (needs promtool)
	@cd backend/deploy/alerts && promtool check rules rules.yaml && promtool test rules rules_test.yaml
