# quickstart 配置与上线指南

本文只描述当前代码真正支持的能力。模块边界先看
[整体架构](./ARCHITECTURE.md)，配置开关看 [配置标准](./CONFIG_STANDARD.md)。

## 0. 准备配置文件

```bash
cd templates/quickstart
cp deploy/config.yaml.example deploy/config.yaml
cp .env.example .env
```

服务器使用 `cp .env.production.example .env`；前端使用
`cp .env.production.example .env.local`。两份 production 示例正好对应下面的最小集合。

默认开发组合：

- auth 启用；
- 邮箱验证码启用，邮件写日志；
- Google/GitHub 未配置，不启用；
- Stripe 不启用；
- referral 启用；
- welcome email 和注册送积分不启用。

约定以下示例使用：

- 前端：`https://app.example.com`
- 后端：`https://api.example.com`
- 后端 API 前缀：`/api/v1`

### 最小生产环境变量

这是“邮箱 + Google + Pro 月付/年付 + 固定积分包”的推荐最小集合。未使用的套餐、
GitHub、邀请奖励和 tracing 可以不配置。

```dotenv
# process / HTTP
CONFIG=deploy/config.yaml
APP_PROJECT=my-saas
APP_ENV=production
APP_SERVER_NAME=my-saas
APP_SERVER_PORT=8080
APP_LOG_LEVEL=info
APP_LOG_FORMAT=json
APP_HTTP_ALLOWED_ORIGINS=https://app.example.com
APP_HTTP_READ_HEADER_TIMEOUT_SECONDS=10
APP_HTTP_READ_TIMEOUT_SECONDS=15
APP_HTTP_WRITE_TIMEOUT_SECONDS=30
APP_HTTP_IDLE_TIMEOUT_SECONDS=60

# PostgreSQL；也可改用一条 APP_DB_DSN
APP_DB_HOST=db.example.com
APP_DB_PORT=5432
APP_DB_USER=my_saas
APP_DB_PASSWORD=数据库密码
APP_DB_NAME=my_saas
APP_DB_SSL_MODE=require
APP_DB_TIME_ZONE=UTC
APP_DB_LOG_LEVEL=warn
APP_DB_SLOW_QUERY_MS=200

# auth；三条 secret 必须分别生成，不能复用
APP_AUTH_ENABLED=true
APP_AUTH_USER_JWT_SECRET=至少32位随机值
APP_AUTH_USER_JWT_EXPIRE_HOURS=168
APP_AUTH_WS_TICKET_TTL_SECONDS=300
APP_AUTH_FRONTEND_REDIRECT=https://app.example.com/login
APP_AUTH_EMAIL_ENABLED=true
APP_AUTH_EMAIL_DEBUG=false
APP_AUTH_EMAIL_CODE_TTL_MINUTES=10
APP_AUTH_EMAIL_CODE_MIN_RESEND_GAP_SECONDS=60
APP_AUTH_EMAIL_CODE_DAILY_CAP=10
APP_AUTH_EMAIL_CODE_MAX_ATTEMPTS=5

# Resend
APP_EMAIL_PROVIDER=resend
APP_EMAIL_RESEND_API_KEY=re_...
APP_EMAIL_RESEND_SENDER_EMAIL=no-reply@example.com
APP_EMAIL_RESEND_SENDER_NAME=My SaaS

# Google OAuth；不用就显式 APP_AUTH_GOOGLE_ENABLED=false 并删除其余项
APP_AUTH_GOOGLE_ENABLED=true
APP_AUTH_GOOGLE_CLIENT_ID=...
APP_AUTH_GOOGLE_CLIENT_SECRET=...
APP_AUTH_GOOGLE_REDIRECT_URL=https://api.example.com/api/v1/auth/google/callback
APP_AUTH_GOOGLE_STATE_SECRET=另一条至少32位随机值
APP_AUTH_GOOGLE_STATE_TTL_MINUTES=20
APP_AUTH_GOOGLE_SCOPE=openid email profile

# Stripe test/live 必须整套匹配；不接支付就设 APP_BILLING_ENABLED=false
APP_BILLING_ENABLED=true
APP_BILLING_PROVIDER=stripe
APP_BILLING_STRIPE_SECRET_KEY=sk_live_...
APP_BILLING_STRIPE_WEBHOOK_SECRET=whsec_...
APP_BILLING_STRIPE_PRICES_PRO_MONTHLY=price_...
APP_BILLING_STRIPE_PRICES_PRO_YEARLY=price_...
APP_BILLING_STRIPE_PRICES_CREDITS=price_...
APP_BILLING_STRIPE_CREDITS_PER_PACKAGE=100

# 宿主业务默认保持关闭，按产品规则再开启
APP_HOST_SIGNUP_CREDITS=0
APP_HOST_WELCOME_EMAIL_ENABLED=false
APP_REFERRAL_ENABLED=false
```

前端只需要公开配置，不放任何 secret：

```dotenv
NEXT_PUBLIC_APP_URL=https://app.example.com
NEXT_PUBLIC_API_BASE_URL=https://api.example.com/api/v1
NEXT_PUBLIC_APP_NAME=My SaaS
NEXT_PUBLIC_AUTH_EMAIL_ENABLED=true
NEXT_PUBLIC_AUTH_OAUTH_PROVIDERS=google
NEXT_PUBLIC_DEFAULT_PLAN=pro
NEXT_PUBLIC_DEFAULT_INTERVAL=monthly
NEXT_PUBLIC_DEFAULT_CREDITS_QUANTITY=1
NEXT_PUBLIC_CREDITS_PRICE_ID=price_...
NEXT_PUBLIC_STRIPE_SUCCESS_PATH=/billing?checkout=success
NEXT_PUBLIC_STRIPE_CANCEL_PATH=/billing?checkout=cancelled
```

生成随机密钥：

```bash
openssl rand -hex 32
```

`APP_ENV=production` 时，后端会在连接数据库之前校验 HTTPS/CORS、弱密钥、调试验证码、
邮件发送配置和 Stripe 必填项。只使用 Google/GitHub 时必须显式关闭邮箱登录，否则生产
校验会要求 Resend 或 Brevo。

## 1. 配置数据库

准备一个 PostgreSQL 数据库和建表权限，填写：

```dotenv
APP_DB_HOST=localhost
APP_DB_PORT=5432
APP_DB_USER=app
APP_DB_PASSWORD=app
APP_DB_NAME=my_saas
APP_DB_SSL_MODE=disable
APP_DB_TIME_ZONE=UTC
```

托管数据库通常要求 `APP_DB_SSL_MODE=require`。

启动时会迁移：

- 当前 SaaS 的 `users`、`user_identities`；
- 已启用模块自己拥有的表；
- `host_migrate.go` 注册的产品表。

AutoMigrate 适合快速起步。生产字段删除、重命名和大表变更应改用当前 SaaS 自己的
版本化 migration，并在升级前备份。

## 2. 生成认证密钥

```bash
openssl rand -base64 32
```

至少生成 JWT 密钥：

```dotenv
APP_AUTH_ENABLED=true
APP_AUTH_USER_JWT_SECRET=第一条随机值
```

Google 和 GitHub 各自再生成一条不同的 state 密钥。不要共用，不要提交到 Git。
修改 JWT 密钥会让已有登录令牌失效。

## 3. 选择登录方式

### 邮箱验证码

开发环境：

```dotenv
APP_AUTH_EMAIL_ENABLED=true
APP_AUTH_EMAIL_DEBUG=true
APP_EMAIL_PROVIDER=log
```

线上必须把 `APP_AUTH_EMAIL_DEBUG=false`。

不需要邮箱登录：

```dotenv
APP_AUTH_EMAIL_ENABLED=false
```

### Google OAuth

按 Google 官方的
[Web Server OAuth 指南](https://developers.google.com/identity/protocols/oauth2/web-server)
创建 Web application 客户端，并把授权回调地址设置为：

```text
https://api.example.com/api/v1/auth/google/callback
```

填写：

```dotenv
APP_AUTH_GOOGLE_ENABLED=true
APP_AUTH_GOOGLE_CLIENT_ID=...
APP_AUTH_GOOGLE_CLIENT_SECRET=...
APP_AUTH_GOOGLE_REDIRECT_URL=https://api.example.com/api/v1/auth/google/callback
APP_AUTH_GOOGLE_STATE_SECRET=独立随机值
APP_AUTH_GOOGLE_STATE_TTL_MINUTES=20
APP_AUTH_GOOGLE_SCOPE=openid email profile
```

代码传给 Google 的 redirect URI 必须和控制台登记值一致。

### GitHub OAuth

创建 GitHub OAuth App，参考官方
[OAuth Web Application Flow](https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/authorizing-oauth-apps)。
Authorization callback URL 填：

```text
https://api.example.com/api/v1/auth/github/callback
```

填写：

```dotenv
APP_AUTH_GITHUB_ENABLED=true
APP_AUTH_GITHUB_CLIENT_ID=...
APP_AUTH_GITHUB_CLIENT_SECRET=...
APP_AUTH_GITHUB_REDIRECT_URL=https://api.example.com/api/v1/auth/github/callback
APP_AUTH_GITHUB_STATE_SECRET=独立随机值
APP_AUTH_GITHUB_STATE_TTL_MINUTES=20
APP_AUTH_GITHUB_SCOPE=read:user user:email
```

模板使用 `user:email` 读取隐藏的已验证邮箱。只需要 GitHub 时，关闭邮箱和 Google：

```dotenv
APP_AUTH_EMAIL_ENABLED=false
APP_AUTH_GOOGLE_ENABLED=false
APP_EMAIL_PROVIDER=none
```

## 4. 配置邮件服务

### Resend

在 Resend 添加自己拥有的域名，按官方
[域名验证说明](https://resend.com/docs/dashboard/domains/introduction)
添加其显示的 SPF 和 DKIM 等 DNS 记录，域名状态变成 verified 后创建 API Key。

```dotenv
APP_EMAIL_PROVIDER=resend
APP_EMAIL_RESEND_API_KEY=re_...
APP_EMAIL_RESEND_SENDER_EMAIL=no-reply@example.com
APP_EMAIL_RESEND_SENDER_NAME=My SaaS
```

发送地址必须属于已经验证的域名或子域。

### Brevo

验证发信域名并创建 API Key，然后：

```dotenv
APP_EMAIL_PROVIDER=brevo
APP_EMAIL_BREVO_API_KEY=...
APP_EMAIL_BREVO_SENDER_EMAIL=no-reply@example.com
APP_EMAIL_BREVO_SENDER_NAME=My SaaS
```

### 欢迎邮件

邮件 Provider 只是发送能力。是否在注册后发送由宿主配置和回调决定：

```dotenv
APP_HOST_WELCOME_EMAIL_ENABLED=true
APP_HOST_WELCOME_EMAIL_ONLY_PROVIDER=google
APP_HOST_WELCOME_EMAIL_SUBJECT=欢迎加入
APP_HOST_WELCOME_EMAIL_TEXT_BODY=你的账号已经创建成功。
```

实际代码在 `internal/bootstrap/host_hooks.go:onUserSignedUp`，可以替换成自己的模板、
队列或营销规则。

## 5. 配置 Stripe

在 Stripe 测试环境创建需要的订阅或一次性价格，取得 `price_...`。再取得后端
secret key，并根据官方 [Webhook 指南](https://docs.stripe.com/webhooks)
创建回调：

```text
https://api.example.com/api/v1/stripe/webhook
```

填写：

```dotenv
APP_BILLING_ENABLED=true
APP_BILLING_PROVIDER=stripe
APP_BILLING_STRIPE_SECRET_KEY=sk_test_...
APP_BILLING_STRIPE_WEBHOOK_SECRET=whsec_...

APP_BILLING_STRIPE_PRICES_STARTER_MONTHLY=price_...
APP_BILLING_STRIPE_PRICES_STARTER_YEARLY=price_...
APP_BILLING_STRIPE_PRICES_PRO_MONTHLY=price_...
APP_BILLING_STRIPE_PRICES_PRO_YEARLY=price_...
APP_BILLING_STRIPE_PRICES_PREMIUM_MONTHLY=price_...
APP_BILLING_STRIPE_PRICES_PREMIUM_YEARLY=price_...
APP_BILLING_STRIPE_PRICES_LIFETIME=price_...
APP_BILLING_STRIPE_PRICES_CREDITS=price_...
APP_BILLING_STRIPE_CREDITS_PER_PACKAGE=100
```

当前前端只创建后端托管的 Stripe Checkout Session，不在浏览器创建 PaymentIntent，
所以不需要 `NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY`。后端保留的 publishable key 字段也是
可选兼容项，不属于最小配置。

`APP_BILLING_STRIPE_PRICES_CREDITS` 对应 `[]string`，环境变量包含多个值时按当前
配置库支持的逗号分隔格式填写；不使用积分包就留空。

Stripe Workbench/Dashboard 的 Webhook 只订阅下面六个事件：

```text
checkout.session.completed
customer.subscription.created
customer.subscription.updated
customer.subscription.deleted
invoice.paid
invoice.payment_failed
```

不要再同时订阅 `invoice.payment_succeeded`；适配器为旧项目保留了兼容解析，但新项目
只用 `invoice.paid`，避免同一张续费账单触发两次业务监听器。Webhook 使用与 API key
相同的 test/live 模式，并把该 endpoint 生成的 signing secret 填入
`APP_BILLING_STRIPE_WEBHOOK_SECRET`。反向代理不能改写请求 body。

另外在 Stripe Customer Portal 中启用允许用户执行的套餐切换、取消和支付方式更新；
这不是 Webhook，但 Billing Portal 页面依赖该配置。

本地 Webhook 可用 Stripe CLI：

```bash
stripe listen --forward-to http://localhost:8080/api/v1/stripe/webhook
```

命令输出的临时 `whsec_...` 用作本地 webhook secret。

当前 billing HTTP 路径仍以 `/stripe` 命名。替换支付商时在当前 SaaS 的
`internal/platform/billing_provider.go` 实现新的 `billing.port.Provider` 并决定新路由。

### 平台侧需要配置什么

| 平台 | 类型 | 配置值 |
| --- | --- | --- |
| Google Cloud OAuth Client | Authorized redirect URI | `https://api.example.com/api/v1/auth/google/callback` |
| GitHub OAuth App | Authorization callback URL | `https://api.example.com/api/v1/auth/github/callback` |
| Stripe Webhook | Endpoint URL | `https://api.example.com/api/v1/stripe/webhook`，订阅上面的六个事件 |
| Resend / Brevo | 发信域名 DNS | 按平台给出的 SPF、DKIM 等记录验证；基础登录不需要邮件 Webhook |

OAuth 回调和 Stripe Webhook 都指向后端。前端域名只用于登录落地、Checkout 返回地址和
CORS。Google/GitHub 的测试、预发、生产建议使用独立 OAuth 应用，避免多个环境互相覆盖
回调地址。

## 6. 配置邀请奖励

```dotenv
APP_REFERRAL_ENABLED=true
APP_REFERRAL_PREFIX=INV
APP_REFERRAL_BASE_LINK=https://example.com/invite?ref=
APP_REFERRAL_ACTIVATION_REWARD=50
APP_REFERRAL_ACTIVATION_WINDOW_DAYS=0
```

默认流程：

1. 注册请求把 `referral_code` 传给 `/auth/verify-code` 或 `/auth/exchange-token`；
2. `UserSignedUp` 监听器创建邀请归因；
3. 被邀请人第一次订阅激活后，邀请关系变成 activated；
4. `ReferralActivated` 监听器给邀请人增加宿主 `User.Credits`。

不使用积分时直接替换 `host_hooks.go:onReferralActivated`；例如改成发优惠券、写现金
返利记录或只做统计。`subscriptions.go` 只负责把邀请事件交给这个回调。

## 7. 配置域名和前端

后端 OAuth 回调域名、前端登录落地地址、邀请链接必须使用当前 SaaS 的域名：

```dotenv
APP_AUTH_FRONTEND_REDIRECT=https://app.example.com/login
APP_REFERRAL_BASE_LINK=https://app.example.com/invite?ref=
```

前端：

```dotenv
NEXT_PUBLIC_APP_URL=https://app.example.com
NEXT_PUBLIC_API_BASE_URL=https://api.example.com/api/v1
NEXT_PUBLIC_AUTH_EMAIL_ENABLED=true
NEXT_PUBLIC_AUTH_OAUTH_PROVIDERS=google,github
```

前端登录开关只控制显示，必须和后端启用的登录方式一致。只用 GitHub 时把邮箱设为
`false`，Provider 列表设为 `github`。

## 8. 链路追踪（可选）

`APP_TRACING_ENDPOINT` 留空就不导出。启用 OTLP：

```dotenv
APP_TRACING_ENDPOINT=collector.example.com:4318
APP_TRACING_PROTOCOL=http
APP_TRACING_INSECURE=false
APP_TRACING_SAMPLE_RATE=0.1
APP_TRACING_AUTHORIZATION=
```

## 9. 发布、服务器启动与测试

### 首次发布共享模块

quickstart 已使用本次新增的 GitHub/GORM auth 适配器，所以必须先发布新模块版本，再让
模板脱离 `go.work`：

```bash
# 在仓库根目录，先提交并 push 当前代码
git tag v0.3.0
git push origin v0.3.0

# tag 可从远端下载后
make pin-template-version VERSION=v0.3.0
make verify-template-release VERSION=v0.3.0

# 再提交并 push templates/quickstart/go.mod 和 go.sum 的更新
```

以后创建项目：

```bash
make init-quickstart \
  DEST=../my-saas \
  MODULE=github.com/your-name/my-saas \
  APP=my-saas \
  VERSION=v0.3.0
```

### push 到服务器后启动

后端使用 Go 1.25.13，前端使用 Node 22 LTS。先准备 `.env`/`.env.local`，不要提交它们：

```bash
# backend
cd backend
GOWORK=off go test ./...
GOWORK=off go build -o my-saas ./cmd/quickstart
./my-saas

# frontend（另一个进程）
cd frontend
nvm use 22
npm ci
npm run verify
npm run start -- --hostname 0.0.0.0 --port 3000
```

`npm run start` 前必须执行过 `npm run build`；`npm run verify` 已包含 build。生产可把两个
命令分别放进 systemd、Docker 或你的进程管理器，并在前面终止 TLS。

### 启动后的最小验收

```bash
# 健康检查
curl -fsS https://api.example.com/health

# 邮箱验证码发送（生产不会返回验证码本身）
curl -fsS https://api.example.com/api/v1/auth/send-code \
  -H 'Content-Type: application/json' \
  --data '{"email":"you@example.com"}'

# OAuth authorize 接口应返回 redirect_url
curl -fsS https://api.example.com/api/v1/auth/google/authorize
```

Stripe 先用 test key 和 test prices。在本地另开终端：

```bash
stripe listen \
  --events checkout.session.completed,customer.subscription.created,customer.subscription.updated,customer.subscription.deleted,invoice.paid,invoice.payment_failed \
  --forward-to http://localhost:8080/api/v1/stripe/webhook
```

把 CLI 输出的 `whsec_...` 放入本地环境，然后从前端真实走一遍 test Checkout。最后确认：

1. `billing_events` 中事件被标记已处理；
2. `billing_subscriptions` 有最新订阅快照；
3. 重发同一个 Stripe event 不会重复发积分；
4. Portal 能打开，取消/恢复后本地快照更新；
5. 未登录访问用户路由返回 401。

## 验收顺序

1. `GET /health` 返回 200；
2. 选中的登录方式可以创建用户并返回 JWT；
3. 未带 JWT 的用户路由被拒绝；
4. 欢迎邮件仅在配置的注册方式触发；
5. Stripe Webhook 验签成功，重复事件不会重复处理；
6. 邀请注册后能查到 pending 关系；
7. 被邀请人付费后关系 activated，奖励只入账一次；
8. 关闭的模块没有对应路由和数据表。

## 上线检查

- [ ] `APP_AUTH_EMAIL_DEBUG=false`
- [ ] JWT、Google state、GitHub state 使用不同随机密钥
- [ ] `.env` 没有进入镜像和 Git
- [ ] 数据库启用了正确 TLS 模式并已备份
- [ ] OAuth 回调使用正式 HTTPS 域名
- [ ] Stripe 测试密钥、价格和 Webhook 已全部换成正式环境的一套
- [ ] 关闭不使用的模块和登录方式
- [ ] 欢迎邮件、注册送积分、邀请奖励规则已按当前 SaaS 审核

## 常见错误

| 现象 | 检查 |
| --- | --- |
| auth 启动失败 | JWT secret 是否存在；启用的 OAuth 配置是否完整 |
| 邮箱路由 404 | `auth.email.enabled` 是否为 true |
| 邮箱登录启动失败 | `email.provider` 是否被设成 none |
| OAuth provider unavailable | 对应 Provider 是否真正启用并注册 |
| `redirect_uri_mismatch` | 代码配置与服务商控制台回调地址是否一致 |
| 支付路由 404 | `billing.enabled` 是否为 true |
| Stripe Webhook 失败 | webhook secret、公开 URL、事件模式是否匹配 |
| 邀请码没归因 | 注册请求是否把 `referral_code` 传给最终登录接口 |
