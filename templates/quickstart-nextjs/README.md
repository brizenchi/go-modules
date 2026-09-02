# quickstart-nextjs

这是与 [`templates/quickstart`](../quickstart/) 配套的可复制 Next.js SaaS 前端模板，
已经包含营销页、文档页、定价、登录、账户、支付管理和邀请页面。

## 后端契约

- 后端 API 挂在 `/api/v1`；
- 响应使用 `foundation/httpresp` 的 JSON envelope；
- 登录方式由前端环境变量选择，并与后端 auth 配置保持一致；
- 支付使用共享 billing 的 Stripe HTTP 接口；
- 邀请使用 referral HTTP 接口；
- 邮箱验证和 OAuth token exchange 都会提交 `referral_code`。

## 页面

| 路径 | 内容 |
| --- | --- |
| `/` | 产品首页 |
| `/pricing` | 定价页 |
| `/docs` | 文档示例页 |
| `/login` | 按配置显示邮箱、Google、GitHub 登录 |
| `/account` | 会话刷新、退出、WebSocket ticket |
| `/billing` | 固定价格 Checkout、订阅变更、Portal、账单、积分包 |
| `/referrals` | 邀请链接、统计和历史 |
| `/invite` | 注册前保存 `?ref=...` |

## 复制和启动

```bash
cp -R templates/quickstart-nextjs ../my-saas-frontend
cd ../my-saas-frontend
cp .env.example .env.local
nvm use
npm ci
npm run dev
```

服务器改用 `.env.production.example` 作为 `.env.local` 的起点。

默认地址：

- 前端：`http://localhost:3000`
- 后端：`http://localhost:8080/api/v1`

## 登录组合

```dotenv
NEXT_PUBLIC_AUTH_EMAIL_ENABLED=true
NEXT_PUBLIC_AUTH_OAUTH_PROVIDERS=google,github
```

`NEXT_PUBLIC_AUTH_OAUTH_PROVIDERS` 支持 `google`、`github`，逗号分隔；留空表示不显示
OAuth 登录。例如只使用 GitHub：

```dotenv
NEXT_PUBLIC_AUTH_EMAIL_ENABLED=false
NEXT_PUBLIC_AUTH_OAUTH_PROVIDERS=github
```

它必须与后端保持一致：前端开关只控制界面，不会启用后端能力。

## 主要环境变量

```dotenv
NEXT_PUBLIC_APP_URL=http://localhost:3000
NEXT_PUBLIC_API_BASE_URL=http://localhost:8080/api/v1
NEXT_PUBLIC_APP_NAME=My SaaS

NEXT_PUBLIC_AUTH_EMAIL_ENABLED=true
NEXT_PUBLIC_AUTH_OAUTH_PROVIDERS=google

NEXT_PUBLIC_DEFAULT_PLAN=pro
NEXT_PUBLIC_DEFAULT_INTERVAL=monthly
NEXT_PUBLIC_DEFAULT_CREDITS_QUANTITY=1
NEXT_PUBLIC_CREDITS_PRICE_ID=
NEXT_PUBLIC_STRIPE_SUCCESS_PATH=/billing?checkout=success
NEXT_PUBLIC_STRIPE_CANCEL_PATH=/billing?checkout=cancelled
```

后端 OAuth 配置示例：

```dotenv
APP_AUTH_FRONTEND_REDIRECT=http://localhost:3000/login
APP_AUTH_GOOGLE_REDIRECT_URL=http://localhost:8080/api/v1/auth/google/callback
APP_AUTH_GITHUB_REDIRECT_URL=http://localhost:8080/api/v1/auth/github/callback
APP_REFERRAL_BASE_LINK=http://localhost:3000/invite?ref=
```

OAuth 控制台登记的是后端 callback；后端完成回调后，再把浏览器带回
`APP_AUTH_FRONTEND_REDIRECT`。Stripe Webhook 也指向后端，Checkout 的 success/cancel
地址才指向前端。

## 通常要改什么

- `NEXT_PUBLIC_APP_NAME` 和品牌文案；
- 登录方式开关；
- 首页、定价和文档内容；
- 套餐和固定积分包默认值；
- 账户菜单中的产品功能入口。

## 验证

```bash
npm test
npm run lint
npm run build
# 或一次执行：
npm run verify
```

手工验收至少覆盖：邀请码保存、所有已启用登录方式、会话刷新与退出、首次订阅、
套餐变更、Billing Portal、固定积分包购买和邀请统计。生产和 CI 使用 Node 22 LTS。
