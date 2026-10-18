# 配置标准

## 配置优先级

```text
进程环境变量 > .env > deploy/config.yaml
```

后端默认读取 `deploy/config.yaml`。可以用 `CONFIG` 指向其他文件。
环境变量统一以 `APP_` 开头，嵌套层级使用下划线：

```text
auth.github.client_id  → APP_AUTH_GITHUB_CLIENT_ID
host.welcome_email.enabled → APP_HOST_WELCOME_EMAIL_ENABLED
```

密钥只放环境变量或未提交的 `.env`；非敏感默认值放 YAML。

历史版本可能在 `service_provider_settings` 保存了服务配置，此时整个服务的数据库快照
会覆盖上述部署配置。Stripe 可以显式迁回部署配置：

```dotenv
APP_BILLING_STRIPE_CONFIG_SOURCE=environment
```

此开关必须随支持它的后端版本部署，并重启全部后端实例。先在部署环境补齐同一 Stripe
账号及 test/live 模式的 secret key、Webhook secret、所有 Price ID、试用和积分包配置。
它统一切换 Stripe 的凭据、价格与支付开关，不混用数据库密钥和环境价格；启用支付但缺少
必要配置会阻止启动。Lifetime 的 Price 必须为启用的一次性价格。

未设置或留空时保留旧数据库覆盖行为，避免升级代码时自动切换生产凭据。数据库记录不会
删除或修改，Resend 的来源不受这个 Stripe 开关影响。移除开关并重启会恢复旧的来源规则，
包括旧 Price ID。启动日志 `Stripe configuration loaded` 的 `source` 显示实际来源；
管理员读取接口 `GET /api/v1/admin/integrations` 的 `active_source` 与其一致，不返回密钥。
旧 YAML 没有 `config_source` 字段时，环境变量开关仍能生效。

## 模块开关

| 配置 | 默认规则 | 结果 |
| --- | --- | --- |
| `auth.enabled` | 默认 true | false 时不创建 auth、不挂登录路由 |
| `auth.email.enabled` | 默认 true | false 时不挂邮箱验证码路由 |
| `auth.google.enabled` | 省略时根据凭据推断 | 注册或移除 Google Provider |
| `auth.github.enabled` | 省略时根据凭据推断 | 注册或移除 GitHub Provider |
| `billing.enabled` | 省略时根据 Stripe 凭据推断 | false 时不迁移 billing 表，支付接口返回明确的 503 |
| `referral.enabled` | 默认 true | false 时不迁移邀请表、不挂邀请路由 |
| `email.provider` | 默认 log | `none`、`log`、`resend`、`brevo` |

显式 `enabled: false` 的优先级高于“凭据存在”。

## 常用组合

### 只使用 GitHub 登录

```yaml
auth:
  enabled: true
  email:
    enabled: false
  google:
    enabled: false
  github:
    enabled: true
    client_id: "..."
    client_secret: "..."
    redirect_url: "https://api.example.com/api/v1/auth/github/callback"
    state_secret: "..."

email:
  provider: none
```

### Google 注册后用 Resend 发欢迎邮件

```yaml
auth:
  google:
    client_id: "..."
    client_secret: "..."

email:
  provider: resend
  resend:
    api_key: "..."
    sender_email: "no-reply@example.com"

host:
  welcome_email:
    enabled: true
    only_provider: google
    subject: "欢迎加入"
    text_body: "你的账号已经创建成功。"
```

实际发送规则在 `internal/bootstrap/host_hooks.go`，不是 auth 或 email 模块的固定行为。

### 不使用支付和邀请

```yaml
billing:
  enabled: false

referral:
  enabled: false
```

对应模块表不会创建。Auth 和 Referral 路由不会挂载；Billing 为了让客户端可诊断配置状态，
保留同路径的 503 响应。客户端可通过 `GET /api/v1/capabilities` 在请求前判断模块状态。

## 配置归属

| 配置 | 放在哪里 |
| --- | --- |
| 日志、数据库、追踪 | `bootstrap.AppConfig` |
| 模块和服务商配置 | `internal/platform.Config` 及其子结构 |
| 当前 SaaS 的业务规则 | `internal/hostcfg.Config` |

产品功能新增配置时，优先加入 `hostcfg`。只有多个 SaaS 都需要并且语义一致的服务商
参数，才加入 platform 配置。

### 后台网站设置的边界

`/admin/settings` 管理品牌、双语首页介绍、支持渠道及不含密钥的发件名称/邮箱。公开呈现由 Next.js 服务端读取并短期缓存；发件资料在后端启动时覆盖对应默认值，需要重启生效。Stripe、Webhook、邮件 API 密钥由部署配置管理；后台凭证写入接口返回 403。旧数据库服务覆盖配置保持启动读取兼容，Stripe 通过上述 `config_source=environment` 显式迁回部署配置，不会自动删除记录或切换其他服务。语言仍由前端代码定义为 EN/中文。

## 校验原则

- 启用 auth 时必须有 `auth.user_jwt_secret`；
- Billing 和 Referral 只提供用户路由，因此启用任一能力时必须同时启用 auth；
- 启用邮箱登录时 `email.provider` 不能是 `none`；
- 显式启用 OAuth 后，客户端 ID、密钥、回调地址和 state 密钥必须完整；
- 启用 billing 时当前只支持 `provider: stripe`，并要求 secret key 和 webhook secret；
- 启用 referral 时要求有效的 `base_link`，生产环境必须是 HTTPS；未填写时会从
  `auth.frontend_redirect` 的 origin 派生 `/invite?ref=`；
- 不认识的 provider 直接启动失败，不静默回退。

### 邀请奖励扩展点

邀请奖励不使用配置：在 `templates/quickstart/internal/bootstrap/host_hooks.go` 的
`onReferralRegistered`（绑定邀请后）或 `onReferralActivated`（激活后）直接编写规则。
两个函数默认无副作用，不发积分或会员；旧 `referral.activation_reward` /
`APP_REFERRAL_ACTIVATION_REWARD` 已移除。邀请开关、链接、激活期限配置仍然有效。

## Optional AI integration

Host-owned configuration lives in `host.ai` / `APP_HOST_AI_*`: `enabled` (default false), `base_url` (HTTPS API base), `api_key` (server secret), `model`, and `max_output_tokens` (default 1024, maximum 4096). Invalid enabled configuration fails startup validation. Restart after changes. Public configuration and browser variables never contain the key.

Configuration supplies the `internal/ai` factory; products explicitly inject the resulting `foundation/aix.Generator` into their own features. It opens no default AI routes and defines no conversation storage, daily quota, subscription or credit policy. See `templates/quickstart-nextjs/EXTENSIONS.md` and `foundation/aix/README.md` for integration and transport limits.
