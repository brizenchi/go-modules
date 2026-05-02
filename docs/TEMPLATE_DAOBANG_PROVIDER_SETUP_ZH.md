# Template 第三方接入联调

这份文档固定按下面这组域名说明：

- 前端：`https://template.daobang.tech`
- 后端：`https://api.template.daobang.tech`

适用对象：

- `templates/quickstart`
- `templates/quickstart-nextjs`
- `stacks/saascore`

## 一、域名与回调约定

这几个值必须统一：

```dotenv
# frontend
NEXT_PUBLIC_APP_URL=https://template.daobang.tech
NEXT_PUBLIC_API_BASE_URL=https://api.template.daobang.tech/api/v1

# backend
APP_AUTH_FRONTEND_REDIRECT=https://template.daobang.tech/login
APP_AUTH_GOOGLE_REDIRECT_URL=https://api.template.daobang.tech/api/v1/auth/google/callback
APP_REFERRAL_BASE_LINK=https://template.daobang.tech/invite?ref=
```

第三方平台里的地址也要完全一致：

- Google OAuth callback: `https://api.template.daobang.tech/api/v1/auth/google/callback`
- Stripe webhook: `https://api.template.daobang.tech/api/v1/stripe/webhook`
- Stripe success URL: `https://template.daobang.tech/billing?checkout=success`
- Stripe cancel URL: `https://template.daobang.tech/billing?checkout=cancelled`

## 二、后端建议配置

`templates/quickstart/.env` 可直接按这个版本填：

```dotenv
CONFIG=deploy/config.yaml

APP_SERVER_NAME=template
APP_SERVER_PORT=8080
APP_LOG_LEVEL=info
APP_LOG_FORMAT=json

APP_DB_HOST=YOUR_DB_HOST
APP_DB_PORT=5432
APP_DB_USER=YOUR_DB_USER
APP_DB_PASSWORD=YOUR_DB_PASSWORD
APP_DB_NAME=YOUR_DB_NAME
APP_DB_SSL_MODE=disable
APP_DB_TIME_ZONE=Asia/Shanghai

APP_AUTH_USER_JWT_SECRET=CHANGE_ME_TO_A_LONG_RANDOM_SECRET
APP_AUTH_FRONTEND_REDIRECT=https://template.daobang.tech/login

APP_AUTH_EMAIL_DEBUG=false
APP_AUTH_EMAIL_VERIFICATION_TEMPLATE_REF=

APP_AUTH_GOOGLE_CLIENT_ID=YOUR_GOOGLE_CLIENT_ID
APP_AUTH_GOOGLE_CLIENT_SECRET=YOUR_GOOGLE_CLIENT_SECRET
APP_AUTH_GOOGLE_REDIRECT_URL=https://api.template.daobang.tech/api/v1/auth/google/callback
APP_AUTH_GOOGLE_STATE_SECRET=CHANGE_ME_TO_ANOTHER_LONG_RANDOM_SECRET
APP_AUTH_GOOGLE_SCOPE=openid email profile

APP_EMAIL_PROVIDER=resend
APP_EMAIL_RESEND_API_KEY=YOUR_RESEND_API_KEY
APP_EMAIL_RESEND_SENDER_EMAIL=no-reply@YOUR_VERIFIED_DOMAIN
APP_EMAIL_RESEND_SENDER_NAME=Template

APP_BILLING_STRIPE_ENABLED=true
APP_BILLING_STRIPE_SECRET_KEY=sk_test_xxx
APP_BILLING_STRIPE_PUBLISHABLE_KEY=pk_test_xxx
APP_BILLING_STRIPE_WEBHOOK_SECRET=whsec_xxx
APP_BILLING_STRIPE_PRICES_STARTER_MONTHLY=price_xxx
APP_BILLING_STRIPE_PRICES_STARTER_YEARLY=price_xxx
APP_BILLING_STRIPE_PRICES_PRO_MONTHLY=price_xxx
APP_BILLING_STRIPE_PRICES_PRO_YEARLY=price_xxx

APP_REFERRAL_PREFIX=INV
APP_REFERRAL_BASE_LINK=https://template.daobang.tech/invite?ref=
APP_REFERRAL_ACTIVATION_REWARD=50
```

## 三、前端建议配置

`templates/quickstart-nextjs/.env.local`：

```dotenv
NEXT_PUBLIC_APP_URL=https://template.daobang.tech
NEXT_PUBLIC_API_BASE_URL=https://api.template.daobang.tech/api/v1
NEXT_PUBLIC_APP_NAME=Template
NEXT_PUBLIC_DEFAULT_PLAN=pro
NEXT_PUBLIC_DEFAULT_INTERVAL=monthly
NEXT_PUBLIC_DEFAULT_CREDITS_QUANTITY=1
NEXT_PUBLIC_CREDITS_PRICE_ID=
NEXT_PUBLIC_STRIPE_SUCCESS_PATH=/billing?checkout=success
NEXT_PUBLIC_STRIPE_CANCEL_PATH=/billing?checkout=cancelled
```

## 四、Stripe 测试

Stripe 可以本地测，也可以直接用公网域名测。

本地联调：

1. 启动 backend：`go run ./cmd/quickstart`
2. 启动 frontend：`npm run dev`
3. 登录 Stripe CLI
4. 执行：

```bash
stripe listen --forward-to http://localhost:8080/api/v1/stripe/webhook
```

5. 在前端 `/billing` 发起订阅
6. 用测试卡：
   - `4242 4242 4242 4242`
   - 任意未来日期
   - 任意 CVC

公网域名联调：

- Stripe Dashboard Webhook Endpoint 配成：
  `https://api.template.daobang.tech/api/v1/stripe/webhook`

必测清单：

1. 首次订阅 checkout
2. Billing 页面订阅信息读取
3. 月付 -> 年付切换
4. 年付 -> 月付切换
5. 降级是否按周期末生效
6. cancel
7. reactivate
8. Billing Portal 打开
9. invoices 列表读取

说明：

- 当前模板已经支持 `preview / change / portal / cancel / reactivate / webhook`
- 月付升年付默认立即切换
- 年付降月付默认周期末切换

## 五、Google OAuth 测试

Google 这套建议直接用公网域名测，不建议只测 localhost。

Google Cloud Console 里至少配置：

- OAuth Client 类型：Web application
- Authorized redirect URI：
  `https://api.template.daobang.tech/api/v1/auth/google/callback`

测试步骤：

1. 配好 `APP_AUTH_GOOGLE_CLIENT_ID`
2. 配好 `APP_AUTH_GOOGLE_CLIENT_SECRET`
3. 配好 `APP_AUTH_GOOGLE_STATE_SECRET`
4. 启动前后端
5. 打开 `https://template.daobang.tech/login`
6. 点击 `Continue with Google`
7. 确认浏览器跳转 Google
8. Google 回调后，浏览器应回到：
   `https://template.daobang.tech/login#code=...`
9. 前端继续 exchange token，最终拿到登录态

如果失败，优先检查这三处是否完全一致：

- 后端 env 里的 `APP_AUTH_GOOGLE_REDIRECT_URL`
- Google Console 里的 redirect URI
- 实际访问的后端域名

## 六、Resend 测试

当前模板已经补齐 `resend` 标准接入：

- `APP_EMAIL_PROVIDER=resend`
- `APP_EMAIL_RESEND_API_KEY`
- `APP_EMAIL_RESEND_SENDER_EMAIL`
- `APP_EMAIL_RESEND_SENDER_NAME`

并且验证码邮件不依赖 Resend 后台模板，shared stack 会直接发送默认验证码邮件内容。

前置条件：

1. Resend 里已经验证发信域名
2. `APP_EMAIL_RESEND_SENDER_EMAIL` 属于已验证域名
3. 后端机器能访问外网

测试步骤：

1. 启动 backend
2. 调用：

```bash
curl -X POST https://api.template.daobang.tech/api/v1/auth/send-code \
  -H 'Content-Type: application/json' \
  -d '{"email":"YOUR_TEST_EMAIL"}'
```

3. 收到验证码邮件
4. 再调用 `/api/v1/auth/login`
5. 用验证码换登录 token

本地调试阶段如果只想先走通登录流程，可以临时开：

```dotenv
APP_AUTH_EMAIL_DEBUG=true
```

这样 `send-code` 响应里会直接返回验证码，不依赖真实收信。

## 七、建议测试顺序

建议不要三套一起开测，顺序按这个来：

1. 先测 Resend 或 `APP_AUTH_EMAIL_DEBUG=true` 的邮箱验证码登录
2. 再测 Google OAuth
3. 最后测 Stripe

原因：

- 先把登录态拿稳，后面的 Billing/Referral 才好测
- Stripe 依赖用户登录和 webhook
- Google 依赖公网回调配置

## 八、当前结论

仓库模板侧现在应以这套标准为准：

- Google callback 归后端域名
- Stripe webhook 归后端域名
- Stripe success/cancel 回前端域名
- Resend 由后端直接调用，不需要 webhook
- `quickstart` 已经支持 `resend` 作为标准邮件 provider

如果你下一步要继续，我建议直接按这份文档把 `template.daobang.tech` 环境变量填上，然后先从 `Resend / 邮箱验证码登录` 开始验收。
