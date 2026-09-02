# 版本管理

## 两种更新不是一回事

### 共享模块升级

`foundation/*` 和 `modules/*` 通过 Go module 版本升级：

```bash
go get github.com/brizenchi/go-modules@<目标版本>
go mod tidy
go test ./...
```

适合接收认证安全修复、Stripe Webhook 修复和新适配器。

### 模板更新

`templates/quickstart` 被复制后就属于当前 SaaS。模板的新字段、新监听器和新示例需要
项目选择性合并，不能由 `go get` 自动覆盖。

这正是用户字段安全的来源：共享模块升级不会自动修改当前 SaaS 的 `User`。

## 版本规则

- 修复，不改变公开接口：补丁版本；
- 向后兼容的新能力或适配器：次版本；
- 删除包、修改端口或迁移所有权：主版本或升级说明明确的预发布版本。

本次删除 `stacks/saascore` 和 `modules/user` 是有意的边界修正。旧项目应先迁出
共享 User，再升级到采用 ADR-0001 的版本。

## 每次发布前

```bash
go test ./...
go vet ./...
go build ./...

cd templates/quickstart
go test ./...
go vet ./...
go build ./...
```

同时检查 quickstart 的 `go.mod` 是否指向即将发布的版本。

本仓库的模板和共享模块在同一个仓库里，因此新版本第一次发布按这个顺序：

```bash
# 1. 提交并推送共享模块改动，然后创建标签
git tag v0.3.0
git push origin v0.3.0

# 2. 标签可下载后，固定并验证模板依赖
make pin-template-version VERSION=v0.3.0
make verify-template-release VERSION=v0.3.0

# 3. 提交并推送 go.mod/go.sum 的模板指针更新
```

不要在标签还不可下载时把 quickstart 指向它；仓库内 `go.work` 会掩盖这个问题，而独立
项目或服务器构建会失败。

仓库内开发时 `go.work` 会让 quickstart 使用本地根模块；quickstart 的 `go.mod`
仍应指向一个已经存在的版本，避免脱离工作区后下载失败。发布本次重构后，再把它
更新到新标签。

从未发布分支复制模板做联调时，可以临时使用：

```bash
go mod edit -replace github.com/brizenchi/go-modules=../go-modules
go mod tidy
```

发布后必须执行 `go mod edit -dropreplace github.com/brizenchi/go-modules`，再升级到
发布标签。不要让 CI 或生产构建依赖开发机相对路径。
