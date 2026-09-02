.PHONY: help test test-race tidy tidy-check fmt fmt-check vet build purity-check lint vuln list verify-templates verify-template-copy verify-template-release pin-template-version init-quickstart

PACKAGE_ROOTS := foundation modules
GO_PACKAGES := ./foundation/... ./modules/...
GO_SOURCE_DIRS := foundation modules cmd templates/quickstart
GO := GOWORK=off GOCACHE=$(CURDIR)/.cache/go-build GOMODCACHE=$(CURDIR)/.cache/gomod go

help: ## Show targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

list: ## List tracked Go package roots
	@printf '%s\n' $(PACKAGE_ROOTS)

verify-templates: ## 验证后端和前端模板
	@echo "==> templates/quickstart"
	@(cd templates/quickstart && go test ./... && go build ./...)
	@echo "==> templates/quickstart-nextjs"
	@(cd templates/quickstart-nextjs && npm run verify)

verify-template-copy: ## 脱离 go.work 验证当前 quickstart 与本地模块
	@./scripts/verify-quickstart-release.sh local

verify-template-release: ## 验证已发布版本：make verify-template-release VERSION=v0.3.0
	@test -n "$(VERSION)" || { echo "VERSION is required"; exit 2; }
	@./scripts/verify-quickstart-release.sh "$(VERSION)"

pin-template-version: ## 新标签发布后固定模板依赖：make pin-template-version VERSION=v0.3.0
	@test -n "$(VERSION)" || { echo "VERSION is required"; exit 2; }
	@(cd templates/quickstart && GOWORK=off go mod download "github.com/brizenchi/go-modules@$(VERSION)")
	@(cd templates/quickstart && GOWORK=off go mod edit -require="github.com/brizenchi/go-modules@$(VERSION)" && GOWORK=off go mod tidy)

init-quickstart: ## 创建项目：make init-quickstart DEST=../app MODULE=github.com/me/app APP=app VERSION=v0.3.0
	@test -n "$(DEST)" -a -n "$(MODULE)" -a -n "$(APP)" -a -n "$(VERSION)" || { echo "DEST, MODULE, APP and VERSION are required"; exit 2; }
	@./scripts/init-quickstart.sh "$(DEST)" "$(MODULE)" "$(APP)" "$(VERSION)"

build: ## 构建根模块
	@$(GO) build $(GO_PACKAGES)

test: ## 测试根模块
	@$(GO) test $(GO_PACKAGES)

test-race: ## 使用 race detector 测试根模块
	@$(GO) test -race $(GO_PACKAGES)

tidy: ## 整理根模块依赖
	@$(GO) mod tidy

tidy-check: ## 检查 go.mod / go.sum 是否已整理
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

fmt: ## 格式化 Go 代码
	@gofmt -s -w $(GO_SOURCE_DIRS)

fmt-check: ## 检查 Go 代码格式
	@out=$$(gofmt -s -l $(GO_SOURCE_DIRS)); \
	if [ -n "$$out" ]; then \
		echo "✗ files need gofmt:"; \
		echo "$$out"; \
		exit 1; \
	fi; \
	echo "✓ gofmt clean"

vet: ## 静态检查根模块
	@$(GO) vet $(GO_PACKAGES)

lint: ## golangci-lint over the repo root module
	@command -v golangci-lint >/dev/null 2>&1 || { \
		echo "golangci-lint not installed; see https://golangci-lint.run/usage/install/"; exit 1; }
	@golangci-lint run --config $(CURDIR)/.golangci.yml $(GO_PACKAGES)

vuln: ## govulncheck over the repo root module
	@command -v govulncheck >/dev/null 2>&1 || { \
		echo "govulncheck not installed; run: go install golang.org/x/vuln/cmd/govulncheck@latest"; exit 1; }
	@GOWORK=off GOCACHE=$(CURDIR)/.cache/go-build GOMODCACHE=$(CURDIR)/.cache/gomod govulncheck $(GO_PACKAGES)

purity-check: ## 检查共享包是否错误导入宿主代码
	@bad=$$(grep -rE '"github\.com/[^"]+/(internal|pkg/models|pkg/middleware)"' \
		$(PACKAGE_ROOTS) --include='*.go' 2>/dev/null | grep -v _test.go | wc -l | tr -d ' '); \
	if [ "$$bad" != "0" ]; then \
		echo "✗ purity violated: $$bad forbidden imports found"; \
		grep -rE '"github\.com/[^"]+/(internal|pkg/models|pkg/middleware)"' \
			$(PACKAGE_ROOTS) --include='*.go' | grep -v _test.go; \
		exit 1; \
	fi; \
	echo "✓ pkg purity ok"
