# 宿主拥有组合与用户模型：实施计划

> **执行要求：** 按任务逐项实现和验证，不保留与 ADR-0001 冲突的强制共享组装。

**目标：** 把项目调整为“共享模块提供能力和事件，quickstart 拥有用户模型、服务商选择、迁移与业务组合”的可运行架构。

**架构：** 保留 `foundation/*` 和独立 `modules/*`；将认证持久化适配器补入 auth 模块，将所有宿主用户访问改成显式端口；删除强制统一用户模型和 `stacks/saascore`。quickstart 作为唯一默认组合根，按配置创建模块并在宿主监听器中连接事件。

**技术栈：** Go 1.25、Gin、GORM、PostgreSQL/SQLite 测试、进程内事件总线、Stripe、Resend/Brevo、Google/GitHub OAuth。

---

### 任务一：固化模块边界

**文件：**

- 新建：`modules/auth/adapter/gormstore/store.go`
- 新建：`modules/auth/adapter/gormstore/store_test.go`
- 修改：`modules/billing/port/customer.go`
- 修改：`modules/billing/adapter/repo/billing_customer_gorm.go`
- 修改：`modules/billing/adapter/repo/user_resolver_gorm.go`
- 修改：`modules/billing/adapter/repo/*_test.go`

**步骤：**

1. 先为认证 GORM store 和 billing 的宿主账户投影编写失败测试。
2. 将验证码、每日限流和 OAuth 交换码表迁入 auth 自己的 GORM 适配器。
3. 给 billing 增加最小 `AccountLookup` 端口，移除对固定 `users` 表和 `plan` 字段的查询。
4. 运行 `go test ./modules/auth/... ./modules/billing/...`，预期全部通过。

### 任务二：提供可选的 GitHub 登录适配器

**文件：**

- 新建：`modules/auth/adapter/github/provider.go`
- 新建：`modules/auth/adapter/github/provider_test.go`
- 修改：`modules/auth/domain/identity.go`
- 修改：`modules/auth/README.md`

**步骤：**

1. 编写 GitHub 授权地址、状态校验、令牌交换和用户资料映射测试。
2. 实现 `auth/port.IdentityProvider`，并增加 `ProviderGitHub`。
3. 运行 auth 模块测试，预期全部通过。

### 任务三：让 quickstart 成为组合根

**文件：**

- 新建：`templates/quickstart/internal/user/model.go`
- 新建：`templates/quickstart/internal/user/repository.go`
- 新建：`templates/quickstart/internal/user/auth_store.go`
- 新建：`templates/quickstart/internal/user/billing_lookup.go`
- 新建：`templates/quickstart/internal/platform/modules.go`
- 新建：`templates/quickstart/internal/platform/routes.go`
- 新建：`templates/quickstart/internal/platform/*_provider.go`
- 修改：`templates/quickstart/internal/bootstrap/app.go`
- 修改：`templates/quickstart/internal/bootstrap/config.go`
- 修改：`templates/quickstart/internal/bootstrap/host_hooks.go`
- 修改：`templates/quickstart/internal/bootstrap/host_migrate.go`
- 修改：`templates/quickstart/internal/hostapi/hostapi.go`
- 修改：`templates/quickstart/internal/http/router.go`
- 修改：`templates/quickstart/deploy/config.yaml.example`
- 修改：`templates/quickstart/.env.example`

**步骤：**

1. 先编写宿主用户仓储/认证适配器和按配置启停路由的测试。
2. 把默认 `User`、仓储、`auth.UserStore`、`billing.AccountLookup` 放入模板。
3. 创建 `platform.Modules`，分别构造 email、auth、billing、referral；任何模块都不导入另一个模块。
4. 在 `bootstrap/host_hooks.go` 直接订阅模块事件，示范欢迎邮件、邀请归因和付费激活邀请。
5. 只迁移已启用模块拥有的表，并始终由模板迁移自己的 `User`。
6. 按配置挂载路由；关闭邮箱登录、OAuth、billing 或 referral 时不暴露对应接口。
7. 运行 quickstart 全部测试，预期全部通过。

### 任务四：移除错误的共享所有权

**文件：**

- 删除：`modules/user/**`
- 删除：`stacks/saascore/**`
- 修改：所有残留导入、测试、命令和示例

**步骤：**

1. 使用 `rg` 确认 quickstart 已不再导入两个旧包。
2. 删除旧包和仅验证旧强制组装的测试。
3. 调整 legacy billing backfill 测试，使它自己声明 legacy `users` 测试表，不依赖共享用户模块。
4. 运行根模块测试，预期全部通过且 `rg 'modules/user|stacks/saascore' --glob '*.go'` 无结果。

### 任务五：统一中文文档

**文件：**

- 新建或重写：`README.md`
- 新建或重写：`docs/README.md`
- 新建或重写：`docs/ARCHITECTURE.md`
- 新建或重写：`docs/SETUP_ZH.md`
- 新建或重写：`docs/CONFIG_STANDARD.md`
- 新建或重写：`docs/INTEGRATION.md`
- 新建或重写：`templates/quickstart/README.md`
- 修改：模块 README、前端 README 和版本说明

**步骤：**

1. 按代码实际行为写明三层边界、启动链路、用户字段修改方式和事件组合示例。
2. 给出只启用 GitHub、Google 登录后用 Resend 发邮件、关闭支付/邀请、替换支付服务商的具体入口。
3. 删除所有“saascore 拥有 users/统一业务组合”的过期描述。
4. 检查文档链接和配置键都能在代码或示例配置中找到。

### 任务六：全量验证与完成度审计

**步骤：**

1. 对所有 Go 文件执行 `gofmt`。
2. 在根模块和 quickstart 分别运行 `go test ./...`。
3. 运行 `go vet ./...` 和 `go build ./...`。
4. 使用 `rg` 检查旧边界、旧配置键和错误文档引用。
5. 对照 ADR-0001、用户目标和本计划逐项检查，修复所有未完成项后再交付。

