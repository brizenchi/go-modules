# quickstart-nextjs

这是与 [`templates/quickstart`](../quickstart/) 配套的免费 Next.js SaaS 启动模板，
帮助独立开发者和小团队快速上线自己的 SaaS。配套 Go 后端已整合用户认证、
Stripe 订阅与积分计费、邀请推荐和 Resend 邮件能力；前端包含产品官网、可搜索的
中英文文档与博客、套餐展示、账户管理和运营后台。复用通用功能，将开发精力放在核心业务上。

模板本身免费；示例价格页展示的是你可以为自己的 SaaS 配置的收费方式。
数据库、部署、域名和邮件等第三方服务的费用由所选平台另行收取。

## 运营后台与产品分析

用户与管理员使用两套独立界面：用户账户中心只管理自己的账户、订阅和邀请；
运营后台使用独立的 `/admin` 路由与布局，负责全站用户、支付、权益和网站配置。
账户中心的桌面侧栏、移动导航和头像菜单都不会因为当前用户拥有管理员角色而
变成运营导航。管理员通过 `/admin` 单独进入后台。

后台使用独立的**管理员邮箱 + 密码**登录。仅在后端环境中设置
`APP_AUTH_ADMIN_EMAIL` 与 `APP_AUTH_ADMIN_PASSWORD` 后重启，密码长度为 12–72 字节；
不要将密码放入前端 `.env.local` 或任何 `NEXT_PUBLIC_*` 变量。前端根据
`/capabilities` 的 `auth.admin_password_enabled` 显示登录表单，登录成功后保留当前后台路径。
普通用户继续使用邮箱验证码或 OAuth。管理员登录不会提交邀请码，也不会记录密码到浏览器存储。

后台主导航固定为：**运营概览、用户管理、内容管理、网站设置、审计日志**。

- 运营概览展示今日注册、今日邀请、今日支付、总用户与付费用户，以及真实的 7/30 天增长和注册用户付费数据；可选择统计时区与支付环境。
- 用户管理包含用户列表、全局支付和邀请查询。用户弹窗只有「用户概况、支付记录、积分变更记录」三个标签；概况显示注册时间、会员级别与余额，点击对应按钮调整会员或积分。积分流水统一展示各类来源，记录页不重复放置编辑表单。手动会员权益不会修改 Stripe 的扣费计划。
- 支付记录包含升级后记录的收银台创建和已验证付款结果，按收银台与账单关联去重；未付款不会计入付费统计。未被记录的历史弃单无法补回。
- 内容管理支持首页与栏目介绍、功能与场景详情、文章与文档。支持 Markdown 导入、预览、草稿和中英文分别发布/下架。公开正文、SEO 和站点地图均由服务端读取数据库发布快照。
- 网站设置管理品牌、双语首页标题与介绍、支持渠道、发件邮箱和名称。公开页面使用短期服务端缓存，后续访问通常在一分钟内刷新；发件资料需要重启后端生效。
- 服务状态放在网站设置中，只读显示是否配置及运行环境。Stripe、Webhook 和邮件密钥由后端部署配置管理，后台不展示或修改。旧数据库服务覆盖配置暂时保留读取兼容，切换到环境配置前应明确迁移，不能直接删记录。
- 审计通过注入的统一应用服务记录会员/积分、文章、网站配置、奖励核对和失败的后台写请求。关键本地修改与成功审计共用事务。

**尚未内置访客归因漏斗、DAU、留存和产品项目分析**。图表中的付费比例针对注册用户及已验证的支付记录，不能当作访客转化率。产品行为分析需要实际的业务事件与去重定义。

## 后端契约

- 后端 API 挂在 `/api/v1`；
- 响应使用 `foundation/httpresp` 的 JSON envelope；
- 登录方式由后端 `/capabilities` 声明；前端环境变量只能进一步限制显示范围；
- 登录弹窗在当前标签页内缓存可用登录方式一分钟，并复用进行中的查询。刷新页面、缓存过期或点击错误重试时重新获取；Google/GitHub 授权地址仍在点击对应按钮时获取；
- 支付使用共享 billing 的 Stripe HTTP 接口；
- 邀请使用 referral HTTP 接口；
- 邮箱验证和 OAuth token exchange 都会提交 `referral_code`。

## 页面

官网内容入口同时出现在首页、栏目目录、相关内容和页脚。指南页提供文档、产品文章和更新日志的真实链接；产品栏目之间提供同级导航。首页场景和最新指南、详情页相关推荐读取当前语言的已发布目录，不会用示例文章补回空的线上目录。构建检查验证公开页面可从首页经导航和正文链接在三次点击内到达。

| 路径 | 内容 |
| --- | --- |
| `/` | 产品总览：已集成功能、运营能力、体验和配置指南 |
| `/pricing` | 定价页 |
| `/docs` | 中英文配置文档与搜索 |
| `/blog`、`/blog/[slug]` | 中英文指南、全文搜索与独立文章页面 |
| `/updates` | 更新日志 |
| `/contact` | 已配置的支持渠道与自助帮助 |
| `/privacy`、`/terms` | 需按实际业务完善的隐私与条款模板 |
| `/login` | 按配置显示邮箱、Google、GitHub 登录 |
| `/account` | 只读登录邮箱、会员等级与退出 |
| `/billing` | 当前套餐、剩余积分、支付记录、发票和套餐管理 |
| `/referrals` | 邀请链接、统计和历史 |
| `/invite` | 注册前保存 `?ref=...` |
| `/credits` | 可用积分、到期计划与收支明细 |
| `/admin`、`/admin/[section]` | 仅管理员可访问的运营概况、记录查询、权益操作与网站配置 |

内容、联系渠道和 SEO 的编辑说明见 [CONTENT.md](./CONTENT.md)。`/admin` 使用独立
管理员布局，不属于用户账户中心。用户主导航固定为账户中心、订阅与账单、推荐奖励，
积分明细从订阅与账单页进入。购买套餐和积分包从价格页发起；账户页只展示身份与会员等级。旧 `/notes`、`/files` 地址跳转到账户中心，不提供默认业务示例页面。实际权限由后端认证与管理员校验决定，
隐藏导航不能替代权限检查。

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

- 后台内容管理中的首页、栏目介绍与文章；网站设置中的品牌和支持渠道；代码中的布局、按钮和分享卡片；
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

邀请必须单独配置后端（以下激活期限 30 天为示例）：

```dotenv
APP_REFERRAL_ENABLED=true
APP_REFERRAL_BASE_LINK=https://your-site.example/invite?ref=
APP_REFERRAL_ACTIVATION_WINDOW_DAYS=30
```

分享测试使用两个不同账号：原账号复制推荐链接，新账号在另一浏览器或无痕窗口
通过链接注册；回到邀请页后自动更新待激活记录。当前 `internal/bootstrap/host_hooks.go`
在邀请注册时发放邀请人 20、受邀人 10 积分；首次符合条件的付费订阅或终身套餐付款后，
双方再各获 20 / 10 积分，每个阶段只奖励一次。免费试用转为付费才激活，单独购买积分包
不会触发付费激活。金额在宿主代码中定义，调整后需同步邀请页规则。
验收时对照双方积分变更记录，检查注册奖励、付款奖励和 Webhook 重试不重复发放。

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

手工验收至少覆盖：邀请码保存、所有已启用登录方式、账户菜单与直接退出、首次订阅、
套餐变更、Billing Portal、固定积分包购买和邀请统计。生产和 CI 使用 Node 22 LTS。

文章列表排版可在接入本地预览 API 完成构建后运行 `node scripts/check-published-layout.mjs`。
脚本用本机 Chrome/Chromium 离线渲染构建产物，检查中英文 12 个内容目录在
1440、768、390px 下的标题、摘要、搜索框宽度和横向溢出，并输出截图目录。
也可传入构建目录和浏览器路径：`node scripts/check-published-layout.mjs .next "/path/to/chrome"`。
它不调用业务 API，不使用已有浏览器账号或配置，也不安装浏览器依赖。

官网菜单与指南页可运行 `node scripts/check-public-ui.mjs`（可传入浏览器路径和截图目录）。
使用隔离的 Chrome 配置与离线内容，检查真实悬停、退出成功/失败/并发登录保护、纯文字导航及间距、指南筛选和桌面/手机溢出，并生成截图。

后台弹窗回归可运行 `node scripts/check-customer-dialog.mjs`（可依次传入浏览器路径和截图目录）。
脚本使用真实 React 弹窗、表格和下拉组件，模拟 API 边界，离线检查中英文、1440/390px 下
反复切换标签、单一编辑表单、订单与积分流水及横向溢出，输出 20 组验收截图；不连接现有服务或用户账号。

## Optional extension modules

See [EXTENSIONS.md](./EXTENSIONS.md) for reusable tables/forms, GA4 analytics, on-demand Crisp chat, and server-side AI integration. External integrations default off. AI provides a stateless adapter for your own features, without a default chat page or conversation model; no extra payment provider is included.

### 首页与品牌配置

配置新版 API 后，首页、栏目介绍、关于、更新日志和文章读取数据库的已发布记录。首次安装进入 `/admin/articles`，使用「初始化内容」预览并导入中英文内容库。未初始化时，首页只展示网站设置中的站点身份，栏目保留导航与空列表，不会因为没有介绍文案而整页 404；这类内容未齐的页面暂不索引。文章详情不存在或已下架时仍返回 404。导入跳过已有路径；不会覆盖运营方修改。网站设置维护品牌、页脚简介与支持渠道；首页标题、摘要、正文和 SEO 在内容管理的「页面与栏目介绍」维护。保存草稿不会改变公开页面，需按语言分别发布。

正文在服务端渲染，页面/数据缓存为 30 秒并由后续请求触发刷新，非即时清缓存。没有配置 API，或旧后端的内容接口返回路由级 404 时，保留离线展示兼容。新版接口成功返回空列表或未发布记录时，不会回退到示例文章；其他运行错误正常抛出。新文章不需要重新构建前端。布局、按钮和分享图片仍为代码配置。首页截图位于 `public/previews/`，使用虚构演示账号。

发布文章需要运行 Next.js 服务端（ISR），不支持纯静态导出。明确设置 `NEXT_PUBLIC_API_BASE_URL` 后，构建时应能访问后端；后台编辑发布通过后续请求触发短期缓存刷新。未配置 API 的静态演示保留代码内容。后端旧版本缺少内容接口时保留旧页面兼容；新版接口明确下架的内容不会回退到代码种子。
