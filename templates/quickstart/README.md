# quickstart 后端模板

这是一个可以复制后直接开发的 SaaS 后端。它不是共享栈的薄壳：复制后，用户模型、
模块组合、配置和事件回调都属于当前 SaaS。

## 启动链路

```text
cmd/quickstart/main.go
  → bootstrap.New()
    → LoadConfig()                         读取 YAML / .env / 环境变量
    → foundation 日志与追踪
    → pgx.Open()                           连接数据库并健康检查
    → platform.Migrate()                   User + 已启用模块的表
    → platform.New()                       选择适配器并创建模块
    → subscribeModuleEvents()              连接注册、支付和邀请事件
    → http.NewRouter()                     挂模块路由和产品路由
    → buildHostJobs()                      创建后台任务
  → app.Run()                              HTTP、Runner、优雅退出
```

## 目录

```text
internal/
├── user/
│   ├── model.go            当前 SaaS 的 User 和 user_identities
│   ├── repository.go       用户数据访问
│   ├── auth_store.go       实现 auth.port.UserStore
│   └── billing_lookup.go   实现 billing.port.AccountLookup
├── platform/
│   ├── config.go           模块和服务商配置
│   ├── migrate.go          按启用状态迁移表
│   ├── modules.go          当前 SaaS 的模块集合
│   ├── auth_provider.go    Google/GitHub/JWT/验证码组装
│   ├── email_provider.go   none/log/Resend/Brevo
│   ├── billing_provider.go Stripe 组装；替换支付入口
│   └── routes.go           按启用状态挂路由
├── bootstrap/
│   ├── app.go              进程生命周期
│   ├── subscriptions.go    模块之间的事件连接
│   ├── host_hooks.go       当前 SaaS 的业务回调
│   ├── host_migrate.go     产品功能模型
│   └── host_jobs.go        后台任务
├── feature/
│   ├── account/            用户资料读取与更新 API
│   └── note/               service/repository/handler 参考功能
├── hostapi/                传给产品功能的依赖和路由组
└── hostcfg/                当前 SaaS 的业务配置
```

## 第一次启动

```bash
cp deploy/config.yaml.example deploy/config.yaml
cp .env.example .env

# 如果这是从尚未发布的本地 go-modules 分支复制出的同级目录：
go mod edit -replace github.com/brizenchi/go-modules=../go-modules
go mod tidy

# 准备 PostgreSQL 后：
go run ./cmd/quickstart
curl http://localhost:8080/health
```

服务器把 `.env.production.example` 复制为 `.env`，逐项替换其中的域名和
`CHANGE-ME`。

新版本发布后，删除本地 replace，并把 `go-modules` 依赖升级到发布标签。
生产环境变量、Google/GitHub 回调、Stripe Webhook 和上线验收见
[配置与上线指南](../../docs/SETUP_ZH.md)。`APP_ENV=production` 时，模板会在连接数据库
之前拒绝弱密钥、HTTP 回调、通配 CORS、调试验证码和不完整的 Stripe/邮件配置。

如果这个后端仍在原始 `go-modules` monorepo 内，Dokploy 必须以仓库根目录 `/` 为构建
上下文并使用根目录 `/Dockerfile`。当前目录中的 Dockerfile 是给复制后的独立项目使用的，
它只读取这里的 `go.mod`，因此只会构建已经发布并锁定的共享模块版本。

部署到前后端不同域名时，设置 `APP_HTTP_ALLOWED_ORIGINS` 为前端 origin（只有协议和域名，
不要包含 `/login` 路径），例如 `https://app.example.com`。未显式设置时，容器会从
`APP_AUTH_FRONTEND_REDIRECT` 自动提取该 origin；OAuth 启用时，启动校验也会确认两者一致。

OAuth 登录按钮会先在当前 tab 的 `sessionStorage` 生成一次性 verifier，然后顶层跳转到
后端 `/auth/:provider/authorize?redirect=1&challenge=...`。后端用 HttpOnly flow cookie
绑定 provider callback，最终 `/auth/exchange-token` 还会核对 verifier，防止 state 或
前端 `?code=` 被复制到另一浏览器造成 session swapping。生产环境必须保持
`APP_AUTH_OAUTH_COOKIE_SECURE=true`；本地 `http://localhost` 使用 false。旧前端若未发送
`challenge`/`oauth_verifier` 会收到 400，需要与本次后端一起升级。

## 修改用户字段

只修改 `internal/user/model.go`：

```go
type User struct {
    // 默认身份字段……

    WorkspaceID   string `gorm:"type:varchar(36);index"`
    Locale        string `gorm:"type:varchar(20)"`
    OnboardingStep int
}
```

本地快速开发仍然使用 GORM AutoMigrate。生产升级时建议把字段变更转成当前 SaaS 自己
的版本化 migration。不要把产品字段加进 `go-modules/modules/auth`。

auth 适配器只把宿主 User 映射为：

```go
authdomain.Identity{UserID, Email, Username, AvatarURL, Role}
```

因此新增字段不会影响共享模块或其他 SaaS。

不是所有“和用户有关”的数据都应该加进 `User`：

- 语言、时区、引导步骤等稳定的一对一资料，可以直接加字段；
- 工作区成员、积分流水、配额、偏好等规则容易变化的数据，放到
  `internal/feature/*` 的独立表，用 `user_id` 关联；
- Stripe customer、订阅和邀请关系由对应模块表保存；
- 只有暂时不稳定、无需索引和约束的小块数据才考虑 JSON，稳定后迁出。

生产环境新增、重命名或删除字段时，为当前 SaaS 写版本化 migration。其他 SaaS 不会
被迫跟着迁移，这正是把 User 放在模板而非共享模块里的原因。

## 决定使用哪些模块

配置示例：

```yaml
auth:
  enabled: true
  email:
    enabled: false
  google:
    enabled: false
  github:
    enabled: true

billing:
  enabled: false

referral:
  enabled: true
```

关闭模块会同时停止建表和挂路由。完整规则见
[配置标准](../../docs/CONFIG_STANDARD.md)。

## 编写业务组合

`internal/bootstrap/host_hooks.go` 的函数就是可修改的业务回调：

```go
func onUserSignedUp(
    ctx context.Context,
    deps hostapi.Deps,
    envelope authevent.Envelope,
    event authevent.UserSignedUp,
) error {
    // 当前 SaaS 自己决定：发欢迎邮件、送积分、创建 workspace……
    return nil
}
```

模板已经示范：

- `UserSignedUp`：可选注册积分和欢迎邮件；
- `CreditsPurchased`：把购买积分写入宿主 `User.Credits`；
- `SubscriptionActivated`：激活邀请关系；
- `ReferralActivated`：把邀请奖励写入宿主 `User.Credits`。

回调同时收到通用 `Envelope` 和强类型 payload。支付回调可从 envelope 取得
`UserID`、供应商和 Webhook event ID，再从 payload 读取订阅或金额等业务内容。

不使用积分的 SaaS 可以删除 `User.Credits` 和对应两个监听器。奖励是优惠券或现金时，
直接在 `onReferralActivated` 调用自己的 service。

## 替换服务商

| 需求 | 修改位置 |
| --- | --- |
| 只使用 GitHub | 配置关闭 email 和 Google，启用 GitHub |
| Resend 换 Brevo | `email.provider: brevo` |
| 邮件换其他服务 | 实现 `email.port.Sender`，修改 `buildEmail` |
| Stripe 换其他支付 | 实现 `billing.port.Provider`，修改 `buildBilling` |
| 自己的登录平台 | 实现 `auth.port.IdentityProvider`，在 `buildIdentityProviders` 注册 |

共享模块不需要知道当前 SaaS 选了哪一个服务商。

## 添加产品功能

参考 `internal/feature/note`：

```text
note.go        实体和 Register
repository.go  数据访问，只在这里接触 GORM
service.go     业务规则，不依赖 Gin
handler.go     HTTP 输入输出
```

新增功能后：

1. 在 `host_migrate.go` 注册模型；
2. 在 `internal/http/host_routes.go` 注册路由；
3. 需要模块能力时从 `hostapi.Deps` 读取，不使用包级全局变量。

路由组：

| 路由组 | 权限 |
| --- | --- |
| `g.Public` | 匿名 |
| `g.User` | 有效用户 JWT |
| `g.Admin` | 用户 JWT + admin 角色 |

auth 被关闭时，`g.User` 和 `g.Admin` 返回 503，不会退化成匿名访问。

模板内置两个前端可直接消费的宿主接口：

- `GET /api/v1/account/profile`：读取当前用户的公开资料和只读账户状态；
- `PATCH /api/v1/account/profile`：只允许更新 `username` 与 `avatar_url`；
- `GET /api/v1/capabilities`：匿名读取 account、billing、referral 的启用状态，以及实际
  配置的订阅 plan/interval、Lifetime、Credits offer；不会返回 Stripe price ID 或密钥。

Billing 未配置时仍保留 `/stripe/*` 路径并返回结构化 503，便于前端区分“功能未配置”与
“地址写错”。是否显示购买入口应以 `capabilities.billing.offers` 为准。

## 后台任务

在 `host_jobs.go` 返回实现 `Runner` 的任务。`App.Run` 会和 HTTP 一起启动，并在退出
时先停止接收请求，再取消 Runner，最长等待 30 秒。

## 测试

```bash
go test ./...
go vet ./...
go build ./...
```

关键参考测试：

- `internal/user/auth_store_test.go`：用户字段与 auth 边界；
- `internal/platform/modules_test.go`：模块启停和 GitHub-only；
- `internal/bootstrap/subscriptions_test.go`：注册、购买和邀请奖励组合。
