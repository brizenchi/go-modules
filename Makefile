.PHONY: help test test-race tidy fmt vet build purity-check list

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

fmt: ## gofmt -s -w
	@gofmt -s -w .

vet: ## go vet ./... in every module
	@for m in $(MODULES); do \
		echo "==> $$m"; \
		(cd $$m && go vet ./...) || exit 1; \
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
