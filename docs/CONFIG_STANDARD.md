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

## 校验原则

- 启用 auth 时必须有 `auth.user_jwt_secret`；
- Billing 和 Referral 只提供用户路由，因此启用任一能力时必须同时启用 auth；
- 启用邮箱登录时 `email.provider` 不能是 `none`；
- 显式启用 OAuth 后，客户端 ID、密钥、回调地址和 state 密钥必须完整；
- 启用 billing 时当前只支持 `provider: stripe`，并要求 secret key 和 webhook secret；
- 启用 referral 时要求有效的 `base_link`，生产环境必须是 HTTPS；未填写时会从
  `auth.frontend_redirect` 的 origin 派生 `/invite?ref=`；
- 不认识的 provider 直接启动失败，不静默回退。
