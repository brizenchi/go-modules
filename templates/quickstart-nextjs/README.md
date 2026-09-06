# quickstart-nextjs

这是与 [`templates/quickstart`](../quickstart/) 配套的免费 Next.js SaaS 启动模板，
帮助独立开发者和小团队快速上线自己的 SaaS。配套 Go 后端已整合用户认证、
Stripe 订阅与积分计费、邀请推荐和 Resend 邮件能力；前端包含产品官网、文档、
套餐展示、账户与订阅管理页面。复用通用功能，将开发精力放在核心业务上。

模板本身免费；示例价格页展示的是你可以为自己的 SaaS 配置的收费方式。
数据库、部署、域名和邮件等第三方服务的费用由所选平台另行收取。

## 产品分析与运营看板（可扩展）

产品分析（Product Analytics）关注用户如何使用产品；运营看板将这些行为与
用户数量、项目数量、付费转化等业务指标汇总展示：

- 用户增长：新增注册用户、累计用户数。
- 活跃与留存：DAU（日活跃用户）、MAU（月活跃用户）、用户留存。
- 业务使用：项目总数、新建项目数、关键功能使用量。
- 转化效果：注册到订阅、邀请到注册或付费的转化。

当前模板已有个人订阅、账单查询和邀请统计，**尚未内置 DAU、留存、项目分析
或运营端数据看板**。这些指标需要接入用户行为采集和产品自己的业务数据。
应先定义活跃行为、统计时区与周期，并按用户去重；不能将访问量或登录次数直接
当作 DAU。项目数量也应以实际业务模型为准。

## 后端契约

- 后端 API 挂在 `/api/v1`；
- 响应使用 `foundation/httpresp` 的 JSON envelope；
- 登录方式由后端 `/capabilities` 声明；前端环境变量只能进一步限制显示范围；
- 支付使用共享 billing 的 Stripe HTTP 接口；
- 邀请使用 referral HTTP 接口；
- 邮箱验证和 OAuth token exchange 都会提交 `referral_code`。

## 页面

| 路径 | 内容 |
| --- | --- |
| `/` | 产品总览：免费 SaaS 启动模板、已集成功能、分析扩展方向、体验和配置指南 |
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

`NEXT_PUBLIC_AUTH_OAUTH_PROVIDERS` 支持 `google`、`github`，逗号分隔。后端
`GET /capabilities` 是登录能力的最终事实来源，前端列表只能从后端已启用的 Provider 中
进一步筛选，不能启用后端未配置的 OAuth。显式留空表示不显示 OAuth；完全不设置时采用
后端返回的列表。例如只允许 GitHub：

```dotenv
NEXT_PUBLIC_AUTH_EMAIL_ENABLED=false
NEXT_PUBLIC_AUTH_OAUTH_PROVIDERS=github
```

`NEXT_PUBLIC_AUTH_EMAIL_ENABLED=false` 同样只能隐藏邮箱登录，不能启用后端已关闭的邮箱登录。

所有 `NEXT_PUBLIC_*` 都是构建时配置。修改域名或能力开关后要重新执行生产构建；发布时
先确认同一版本后端的 `GET /api/v1/capabilities` 返回 200，再发布前端，避免新页面连接
缺少账户、订阅或推荐路由的旧 API。

## 主要环境变量

```dotenv
NEXT_PUBLIC_APP_URL=http://localhost:3000
NEXT_PUBLIC_API_BASE_URL=http://localhost:8080/api/v1
NEXT_PUBLIC_APP_NAME=My SaaS
NEXT_PUBLIC_DEMO_MODE=true

# 可选：仅在需要隐藏后端已启用的登录方式时设置
# NEXT_PUBLIC_AUTH_EMAIL_ENABLED=false
# NEXT_PUBLIC_AUTH_OAUTH_PROVIDERS=github

NEXT_PUBLIC_DEFAULT_PLAN=pro
NEXT_PUBLIC_DEFAULT_INTERVAL=monthly
NEXT_PUBLIC_DEFAULT_CREDITS_QUANTITY=1
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

## 在线体验与正式销售

Template 是帮助开发者快速上线产品的免费 SaaS 启动模板。总览提供中英文产品导览，
引导用户依次注册、购买、管理订阅和分享邀请。账号与邀请关系使用真实后端数据；
演示站的支付使用 Stripe 测试环境，不涉及真实资金。

在 Stripe **测试模式**收银台使用 `4242 4242 4242 4242`、任意未来有效期和
任意三位 CVC（例如 `123`）可模拟成功；`4000 0000 0000 0002` 可模拟拒付。
参见 [Stripe 官方测试说明](https://docs.stripe.com/testing)。支付完成后等待
Webhook 同步，再在订阅管理中查看结果。

分享测试使用两个不同账号：原账号复制推荐链接，新账号在另一浏览器或无痕窗口
通过链接注册，原账号刷新邀请中心查看待激活记录。新账号在奖励期限内首次
完成符合条件的付费订阅后，再查看激活状态与配置的积分奖励；免费试用需要
转为付费订阅才会触发奖励。邀请中心支持复制链接、刷新、状态说明、奖励截止
时间和历史分页。积分奖励发给邀请人，续费和重复回调不会重复产生邀请奖励。

邀请落地页可直接打开注册弹窗，邮箱和第三方登录都会使用已保存的邀请码。
页面上的“已记录邀请码”仅代表浏览器已保存，最终有效性在新账号注册时校验。
已有账号不能重新绑定邀请，访问邀请链接时也不会为已有登录账号保存新邀请码。

`NEXT_PUBLIC_DEMO_MODE=true` 显示总览中的演示说明与测试卡，以及邀请页的测试说明，默认值为 `true`。
正式销售时设置为 `false` 并重新构建前端，隐藏不扣款文案和测试卡。
**此开关只影响页面展示，不切换 Stripe 环境，也不阻止真实扣款。**
部署者必须使展示与实际支付模式一致：演示环境使用 test 密钥、价格和 Webhook；
正式收款时整套换成 live 配置，并将品牌、定价和示例内容替换为自己的产品。
完整配置见 [配置与上线指南](../../docs/SETUP_ZH.md)。

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
