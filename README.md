# go-modules

这是一个面向多个独立 SaaS 的 Go 复用仓库。目标不是让十个 SaaS 共用一张用户表，
而是让它们复用稳定能力，同时各自拥有用户字段、服务商选择和业务规则。

## 三层架构

```text
foundation/*
  通用技术能力：配置、数据库连接、日志、追踪、HTTP 中间件等

modules/*
  独立业务能力：auth、billing、email、referral
  每个模块提供 domain / port / adapter / app / http

templates/quickstart/*
  可复制的 SaaS 后端：拥有 User、数据库迁移、模块选择和事件回调
```

最重要的边界是：

```text
共享模块 = 能力和事件
quickstart = 默认组合方式
复制后的 SaaS = 用户字段和具体业务规则
```

项目不再提供强制的 `stacks/saascore`，也不再提供代表所有 SaaS 的共享
`modules/user`。认证模块通过 `auth/port.UserStore` 使用当前 SaaS 的用户表；
支付模块通过 `billing/port.AccountLookup` 只读取用户 ID 和邮箱。

## 模块清单

| 目录 | 提供什么 | 不负责什么 |
| --- | --- | --- |
| `modules/auth` | 邮箱验证码、Google/GitHub OAuth、JWT、登录事件 | 完整用户表、欢迎邮件、注册送积分 |
| `modules/email` | Resend、Brevo、SMTP、日志发送器 | 决定什么时候发什么邮件 |
| `modules/billing` | Stripe 结算、Webhook 幂等、订阅快照、支付事件 | 用户套餐字段、产品额度和邀请奖励 |
| `modules/referral` | 邀请码、归因、激活状态、邀请事件 | 奖励实际入账方式 |
| `foundation/*` | 通用基础设施 | SaaS 业务流程 |

## 新项目起步

发布版本后，用初始化脚本同时复制后端和前端：

```bash
make init-quickstart \
  DEST=../my-saas \
  MODULE=github.com/me/my-saas \
  APP=my-saas \
  VERSION=v0.3.0
```

脚本会替换 Go module、项目 slug 和共享模块版本，结果位于 `backend/`、`frontend/`。
需要联调当前尚未发布的本地代码时，再手工复制后端并使用临时 replace：

```bash
cp -R templates/quickstart ../my-saas-backend
cd ../my-saas-backend
cp deploy/config.yaml.example deploy/config.yaml
cp .env.example .env

# 当前改动还没有发布标签时，让新项目临时使用旁边的本地仓库：
go mod edit -replace github.com/brizenchi/go-modules=../go-modules
go mod tidy

go run ./cmd/quickstart
```

发布新的 `go-modules` 标签后，删除这个本地 replace，并把依赖升级到该标签。CI 和生产
环境应使用已发布版本，不依赖开发机上的相对路径。

如果要直接从这个 monorepo 部署内置 quickstart（例如 Dokploy），使用仓库根目录的
`Dockerfile`，并把构建上下文设为仓库根目录 `/`。这个入口通过 `go.work` 把当前提交里的
`foundation/*`、`modules/*` 和 `templates/quickstart` 一起构建，因此不会误用模板
`go.mod` 中锁定的旧发布版。`templates/quickstart/Dockerfile` 仍保留给已经复制出去、且
依赖已发布 `go-modules` 版本的独立项目。

复制后的第一批修改通常只有：

1. 在 `internal/user/model.go` 决定当前 SaaS 的用户字段。
2. 在 `deploy/config.yaml` 启用需要的登录、支付和邀请模块。
3. 在 `internal/bootstrap/host_hooks.go` 编写注册、登录、支付、邮件和奖励规则。
4. 在 `internal/feature/*` 添加当前 SaaS 的业务功能。

例如“Google 注册成功后由 Resend 发欢迎邮件”完全发生在模板中：Google 和
Resend 只是注入的适配器，`onUserSignedUp` 才是当前 SaaS 的规则。

## 用户字段怎么改

用户模型在：

```text
templates/quickstart/internal/user/model.go
```

给某个 SaaS 增加 `WorkspaceID`、`Locale` 或 `OnboardingStep` 时，只修改这个
SaaS 的模型、迁移和业务代码。共享 auth 只看到最小 `Identity`，其他 SaaS
不需要升级。

易变的业务状态不要都堆进 `users`：工作区成员、积分流水、配额和偏好更适合放在
当前 SaaS 的独立 feature 表，通过 `user_id` 关联。详细判断规则见
[整体架构：用户字段变更规则](docs/ARCHITECTURE.md#用户字段变更规则)。

支付客户 ID 和订阅状态保存在 billing 自己的表；外部登录身份保存在
`user_identities`，所以同一用户可以同时连接 Google 和 GitHub。

## 推荐阅读顺序

1. [文档入口](docs/README.md)
2. [整体架构](docs/ARCHITECTURE.md)
3. [quickstart 开发指南](templates/quickstart/README.md)
4. [配置标准](docs/CONFIG_STANDARD.md)
5. [第三方服务开通](docs/SETUP_ZH.md)
6. [模块接入方式](docs/INTEGRATION.md)

## 验证

```bash
go test ./...
go vet ./...

cd templates/quickstart
go test ./...
go vet ./...
```

架构决策见 [ADR-0001](docs/adr/0001-template-owned-composition-and-user-schema.md)。
