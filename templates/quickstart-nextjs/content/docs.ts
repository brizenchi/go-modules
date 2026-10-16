import type { ArticleSection, LocalizedText } from "./articles";
import type { Locale } from "../lib/locale";

export const documentationGroups = [
  { id: "start", title: { en: "Start here", zh: "开始使用" }, description: { en: "Get to know the template and try the customer journey.", zh: "了解模板能力，先体验一次完整用户流程。" } },
  { id: "configure", title: { en: "Connect your services", zh: "连接你的服务" }, description: { en: "Bring your own domains, authentication, email, and payments.", zh: "接入域名、登录、邮件和支付，搭建自己的产品。" } },
  { id: "launch", title: { en: "Make it your product", zh: "准备上线" }, description: { en: "Shape your content and check the journey before launch.", zh: "完善产品内容与用户流程，做好上线准备。" } }
] as const;

export type DocumentationSection = ArticleSection & {
  group: typeof documentationGroups[number]["id"];
  summary: LocalizedText;
  items?: { title: LocalizedText; body: LocalizedText }[];
  ordered?: boolean;
  code?: string;
  links?: { href: string; label: LocalizedText }[];
};

export const documentation: DocumentationSection[] = [
  {
    id: "overview",
    group: "start",
    summary: { en: "Understand the free Next.js and Go starter, what it includes, and what you build yourself.", zh: "了解免费的 Next.js 与 Go 模板包含什么，以及哪些业务需要自己开发。" },
    title: { en: "What you can build with this template", zh: "这个模板能帮你做什么" },
    paragraphs: [
      { en: "This is a free Next.js + Go starter for building your own SaaS: a website where customers register, use your service, and manage a paid subscription. It connects the public website, customer account center, and administrator console so you can focus on the service you want to sell.", zh: "这是一个免费的 Next.js + Go SaaS 启动模板，用来搭建你自己的在线产品：用户注册、使用服务、购买和管理订阅。模板已经连接好公开网站、账户中心和管理员后台，你可以把开发精力放在真正要出售的服务上。" },
      { en: "You get the source code and connected example workflows. Add your own core feature, replace the brand and sample plans, and connect your service accounts before launching. The prices shown here demonstrate how your future product can charge customers; they are not a price for downloading the template.", zh: "你拿到的是源码和已经串联好的示例流程。上线前，需要加入自己的核心业务、替换品牌和示例套餐，并接入自己的服务账号。这里展示的套餐价格，是你未来的产品如何收费的示例；下载模板本身免费。" }
    ],
    links: [
      { href: "https://github.com/brizenchi/go-modules/tree/main/templates", label: { en: "Get the template source", zh: "获取模板源码" } },
      { href: "/docs/try-demo", label: { en: "Try the customer journey", zh: "先体验用户流程" } },
      { href: "/docs/domains", label: { en: "Set up your own project", zh: "开始配置自己的项目" } }
    ]
  },
  {
    id: "features",
    group: "start",
    summary: { en: "Explore the public website, customer account center, admin console, and reusable examples.", zh: "浏览产品官网、账户中心、管理员后台与可复用的功能示例。" },
    title: { en: "What is included, and where to find it", zh: "已经有什么功能，分别在哪里" },
    paragraphs: [
      { en: "Each part has a different audience. Your customers manage their own accounts; the site owner uses a separate administrator console to run the service.", zh: "不同页面服务不同的人。你的客户在账户中心管理自己的账号；网站所有者使用独立的管理员后台运营整个网站。" }
    ],
    items: [
      { title: { en: "Public website", zh: "公开网站 · 向访客介绍产品" }, body: { en: "Overview, pricing, setup guide, searchable blog, updates, contact, privacy, and terms pages. Chinese and English copy, sharing metadata, and a sitemap are included.", zh: "总览、价格、使用指南、可搜索的博客、更新记录、联系、隐私与条款页面，带有中英文文案、分享信息和站点地图。" } },
      { title: { en: "Customer account center", zh: "账户中心 · 管理自己的账户" }, body: { en: "The account center at /account, a compact account menu with read-only identity and direct sign-out, subscriptions and invoices at /billing, credit history at /credits, and invitations and rewards at /referrals. Email-code login and optional Google/GitHub sign-in connect to the Go API.", zh: "在 /account 进入账户中心，在账户菜单查看只读登录邮箱或直接退出，在 /billing 管理订阅和账单，在 /credits 查看积分，在 /referrals 管理邀请与奖励。支持邮箱验证码登录，也可配置 Google / GitHub 登录。" } },
      { title: { en: "Administrator console", zh: "管理员后台 · 运营整个网站" }, body: { en: "The administrator console has five destinations: overview, users, articles, website settings and audit log. Manage membership, credits, payments and invitations from a user dialog; keep a global payment view for reconciliation. Credentials remain in deployment configuration.", zh: "管理员后台保留运营概览、用户管理、文章管理、网站设置和审计日志五个入口。在用户弹窗中管理会员、积分、支付与邀请，全站支付记录便于对账。密钥保留在部署配置中。" } },
      { title: { en: "Examples you can extend", zh: "业务示例 · 改造成你自己的产品" }, body: { en: "The customer area supplies common account services only. Add your own product feature and connect it to the existing authentication, credit and storage interfaces. Backend note/export and upload code remains available as implementation references, without default frontend pages.", zh: "用户区仅提供通用账户服务。你可以添加自己的核心业务，再接入已有的认证、积分与存储接口。后端笔记导出和上传代码保留为实现参考，不提供默认前端页面。" } }
    ],
    links: [{ href: "/docs/launch", label: { en: "See what to add before launch", zh: "查看上线前还需要补什么" } }]
  },
  {
    id: "try-demo",
    group: "start",
    summary: { en: "Walk through registration, a Stripe test purchase, and an invitation with two accounts.", zh: "完成注册、Stripe 测试购买，并用两个账号体验邀请流程。" },
    title: { en: "Try a complete customer journey", zh: "按这三步，体验完整用户流程" },
    paragraphs: [
      { en: "Registration, customer actions, and invitation records use the backend and its database. Payment testing uses Stripe’s test environment when configured. The isolated local preview keeps temporary data and has payments disabled, so a checkout cannot be completed there.", zh: "注册、用户操作和邀请记录会经过后端并写入数据库。支付体验需要接入 Stripe 测试环境。隔离的本地 preview 使用临时数据，支付默认关闭，因此不能在其中完成购买。" }
    ],
    ordered: true,
    items: [
      { title: { en: "Create your first account", zh: "注册并进入账户中心" }, body: { en: "Open the account center and use an available sign-in method. With email login, request a code and enter it; a new address creates an account. Check the read-only email in the account menu, then sign out directly and back in.", zh: "进入账户中心，选择页面提供的登录方式。使用邮箱时，发送并输入验证码；新邮箱会创建账号。在账户菜单确认只读邮箱，再直接退出并重新登录，确认完整流程可用。" } },
      { title: { en: "Try a subscription in test mode", zh: "在测试环境购买一次订阅" }, body: { en: "Open /pricing, choose monthly or yearly billing and a plan, then continue directly to Checkout. Only after confirming Stripe test mode, use 4242 4242 4242 4242, a future expiry date, and any three-digit CVC. Return to subscriptions to check the result. If payments are unavailable, the owner must configure them first.", zh: "打开 /pricing，选择月付或年付及套餐，直接进入 Checkout。确认使用 Stripe 测试模式后，填写 4242 4242 4242 4242、未来有效期和任意三位 CVC。支付后回到订阅页查看结果；如果支付不可用，需要先由站点所有者完成配置。" } },
      { title: { en: "Invite a separate new account", zh: "邀请一个独立的新账号" }, body: { en: "Copy your link from referrals. Open it in a separate browser profile or private window and register with a different email. Check the inviter’s pending record, then have the new account complete a qualifying test purchase and check the activation and reward.", zh: "到邀请页面复制链接，在另一浏览器个人资料或无痕窗口中打开，用不同邮箱注册新账号。先在邀请人账号确认“待激活”记录，再让被邀请人完成符合条件的测试购买，检查激活状态和奖励。" } }
    ],
    links: [
      { href: "/account", label: { en: "Start with an account", zh: "从注册账号开始" } },
      { href: "/docs/invitations", label: { en: "Read the reward conditions", zh: "查看邀请奖励条件" } },
      { href: "https://docs.stripe.com/testing", label: { en: "Stripe’s official test cards", zh: "Stripe 官方测试卡说明" } }
    ]
  },
  {
    id: "domains",
    group: "configure",
    summary: { en: "Connect the frontend and API domains, prepare environment variables, and configure CORS.", zh: "连接前端与 API 域名，准备环境变量并配置跨域访问。" },
    title: { en: "Connect your frontend and backend", zh: "准备项目，连接前后端" },
    paragraphs: [
      { en: "The source contains two projects: templates/quickstart-nextjs for the Next.js frontend and templates/quickstart for the Go API. Use both for the full SaaS workflow. Follow their READMEs for the matching runtime, dependency, and database setup.", zh: "源码包含两个配套项目：templates/quickstart-nextjs 是 Next.js 前端，templates/quickstart 是 Go API。完整的 SaaS 流程需要两者一起运行。先按各自 README 准备运行环境、依赖和自己的数据库。" },
      { en: "For local development, copy the frontend .env.example to .env.local and the backend .env.example to .env; also copy the backend deploy/config.yaml.example to deploy/config.yaml. Replace example secrets. For deployment, start from the .env.production.example files and configure your own HTTPS domains and database.", zh: "本地开发时，将前端 .env.example 复制为 .env.local，后端 .env.example 复制为 .env，并将后端 deploy/config.yaml.example 复制为 deploy/config.yaml，替换示例密钥。部署时从 .env.production.example 开始，配置自己的 HTTPS 域名和数据库。" },
      { en: "The frontend API address must include /api/v1. The backend CORS origin must match the frontend address. Start the Go API with go run ./cmd/quickstart in its directory, then run npm ci and npm run dev in the frontend directory. OAuth callbacks and Stripe return URLs must use the same configured domains.", zh: "前端的 API 地址需要包含 /api/v1；后端 CORS 填写前端来源地址。在后端目录用 go run ./cmd/quickstart 启动 API，再在前端目录运行 npm ci 和 npm run dev。OAuth 回调和 Stripe 返回地址也要匹配实际域名。" }
    ],
    code: "# Frontend: .env.local\nNEXT_PUBLIC_APP_URL=http://localhost:3000\nNEXT_PUBLIC_API_BASE_URL=http://localhost:8080/api/v1\n\n# Backend: .env\nAPP_HTTP_ALLOWED_ORIGINS=http://localhost:3000",
    links: [
      { href: "https://github.com/brizenchi/go-modules/tree/main/templates/quickstart-nextjs#readme", label: { en: "Frontend setup README", zh: "前端启动说明" } },
      { href: "https://github.com/brizenchi/go-modules/tree/main/templates/quickstart#readme", label: { en: "Backend setup README", zh: "后端启动说明" } }
    ]
  },
  {
    id: "auth-email",
    group: "configure",
    summary: { en: "Set up Resend email delivery and enable the sign-in methods your product needs.", zh: "配置 Resend 发信，启用产品需要的邮箱或第三方登录方式。" },
    title: { en: "Enable customer sign-in with Resend", zh: "接入 Resend，配置用户登录" },
    paragraphs: [
      { en: "Configure the Resend API key and email-code sign-in in the deployment environment. Website settings can manage the verified sender address and name; restart the API after sender changes. Provider keys cannot be edited in the operator console.", zh: "在部署环境中配置 Resend API Key 和邮箱验证码登录。网站设置可以维护经过验证的发件邮箱和名称，修改发件信息后重启后端。运营后台不编辑服务商密钥。" },
      { en: "Google and GitHub sign-in are optional. Configure their client credentials and callback URLs on the backend when you need them. The frontend shows methods enabled by the API. Verify sign-in, account creation, and sign-out with an email you control before inviting customers.", zh: "Google 和 GitHub 登录按需启用，在后端填写对应应用凭证和回调地址即可。前端会读取 API 提供的登录方式。邀请真实客户之前，先用自己的邮箱验证收码、注册、登录和退出。" }
    ],
    code: "APP_AUTH_EMAIL_ENABLED=true\nAPP_AUTH_EMAIL_DEBUG=false\nAPP_EMAIL_PROVIDER=resend\nAPP_EMAIL_RESEND_API_KEY=<your-resend-api-key>\nAPP_EMAIL_RESEND_SENDER_EMAIL=<your-verified-sender>",
    links: [
      { href: "/docs/integrations", label: { en: "How service settings take effect", zh: "服务配置如何生效" } },
      { href: "https://resend.com/docs/dashboard/api-keys/introduction", label: { en: "Get a Resend API key", zh: "获取 Resend API Key" } },
      { href: "/account", label: { en: "Check customer sign-in", zh: "检查用户登录" } }
    ]
  },
  {
    id: "admin",
    group: "configure",
    summary: { en: "Configure administrator credentials and understand which settings belong in the console.", zh: "配置管理员账号，了解后台设置与后端环境变量各自负责的内容。" },
    title: { en: "Set your administrator email and password", zh: "设置管理员邮箱和密码" },
    paragraphs: [
      { en: "Set both variables below in the backend environment, restart the backend, and open /admin. Sign in with that email and password; you do not need to register the administrator account first. There is no shared default administrator password. Use a generated password of 12–72 bytes; ASCII characters make the byte length easy to check.", zh: "在后端环境变量中同时填写下面两项，重启后端，再打开 /admin，用设置的邮箱和密码登录，无需提前注册管理员账号。模板没有通用的默认管理员密码。密码要求为 12–72 字节，建议使用随机生成的英数符号密码，便于确认长度。" },
      { en: "Website settings manage the public brand, bilingual introduction and homepage heading, support contacts and sender details. Public pages refresh through their short-lived cache. Use article management for localized guides and product articles. Database, authentication, payment and mail credentials remain deployment configuration.", zh: "在网站设置维护公开品牌、中英文服务介绍和首页标题、客服渠道及发件资料，公开页面通过短时缓存刷新。在文章管理编辑和发布多语言指南与产品文章。数据库、登录、支付及邮件密钥由部署环境管理。" }
    ],
    code: "APP_AUTH_ADMIN_EMAIL=<your-admin-email>\nAPP_AUTH_ADMIN_PASSWORD=<your-generated-password>",
    links: [{ href: "/admin", label: { en: "Open administrator sign-in", zh: "打开管理员登录" } }]
  },
  {
    id: "integrations",
    group: "configure",
    summary: { en: "Understand service credentials, configuration overrides, and when a backend restart is needed.", zh: "了解服务密钥、配置覆盖规则，以及哪些修改需要重启后端。" },
    title: { en: "Configure deployment services and review their status", zh: "配置部署服务并查看状态" },
    paragraphs: [
      { en: "Configure the database, JWT, administrator credentials, API domains and provider keys in your deployment environment. Website settings shows provider configuration status and lets you maintain non-secret sender information. Existing installations with database overrides retain compatibility until the operator explicitly migrates those values.", zh: "在部署环境配置数据库、JWT、管理员凭据、前后端域名及服务商密钥。网站设置展示服务配置状态，并维护非敏感的发件信息。已有数据库覆盖配置的项目保持兼容，迁移需由站长明确选择。" },
      { en: "The operator console does not edit provider credentials. Existing database overrides are retained for compatibility; move them to your deployment secret configuration through an explicit operator-controlled migration before removing them.", zh: "运营后台不编辑服务商凭据。已有数据库覆盖配置保持兼容；移除前，应由站长明确将其迁入部署密钥配置，避免影响已接入的支付和邮件服务。" }
    ],
    items: [
      { title: { en: "Where the settings come from", zh: "配置从哪里读取" }, body: { en: "New installations use deployment configuration for provider credentials. Website settings maintains public identity, support contacts and non-secret sender details. Existing provider overrides keep their previous precedence until explicitly migrated.", zh: "新项目通过部署配置管理服务商凭据，网站设置维护公开品牌、客服渠道和非敏感的发件资料。历史服务覆盖配置保持原有优先级，直到站长明确迁移。" } },
      { title: { en: "When changes apply", zh: "什么时候生效" }, body: { en: "Public branding and article publication refresh through the website cache, usually within one minute of subsequent visits. Sender or deployment credential changes require a backend restart; a configured badge does not prove a successful connection.", zh: "公开品牌和文章发布通过网站缓存刷新，通常在后续访问触发后一分钟内生效。发件资料和部署凭据变更需要重启后端；已配置状态不代表连通性已验证。" } },
      { title: { en: "What saving verifies", zh: "保存会检查什么" }, body: { en: "Status reports configuration, not external API permissions, email delivery or successful charges. Verify delivery and test payments in an isolated test environment. The local preview never activates external email or payments.", zh: "状态展示配置情况，不代表外部 API 权限、邮件投递或真实支付已验收。请在隔离测试环境验证发信与测试购买；本地 preview 始终不启用外部邮件或支付。" } }
    ],
    code: "# First boot only; also configure your database, JWT, admin credentials and domains.\nAPP_AUTH_ENABLED=true\nAPP_AUTH_EMAIL_ENABLED=false\nAPP_AUTH_EMAIL_DEBUG=false\nAPP_AUTH_GOOGLE_ENABLED=false\nAPP_AUTH_GITHUB_ENABLED=false\nAPP_EMAIL_PROVIDER=none\nAPP_BILLING_ENABLED=false",
    links: [{ href: "/admin/settings", label: { en: "Open website settings", zh: "打开网站设置" } }]
  },
  {
    id: "ai-integration",
    group: "configure",
    summary: { en: "Connect a model provider on the server and inject it into your own product features.", zh: "在服务端接入模型服务商，并将调用能力注入自己的业务功能。" },
    title: { en: "Connect AI to your own product", zh: "为自己的产品接入 AI" },
    paragraphs: [
      { en: "The template supplies a stateless model adapter in foundation/aix and a backend configuration factory in internal/ai. It accepts explicit messages and returns a text completion. There is no default customer chat page, chat API, conversation database or daily usage policy.", zh: "模板在 foundation/aix 提供无状态模型适配器，在后端 internal/ai 提供配置工厂。输入明确的消息，返回文本结果；默认不提供用户聊天页面、聊天接口、对话数据库或每日用量规则。" },
      { en: "Configure a provider supporting the Chat Completions protocol, then create the adapter in your product composition code with the injected HTTP client. Configuration alone does not expose an AI endpoint. Keep the API key on the server and pass the request context into every model call.", zh: "配置支持 Chat Completions 协议的服务商后，在产品组装代码中使用注入的 HTTP 客户端创建适配器。仅填写配置不会自动开放 AI 接口。密钥保留在服务端，每次模型调用都传递请求上下文。" },
      { en: "Your product owns its prompts, customer authorization, conversation history, task lifecycle and charging policy. Reuse the credit ledger if your feature charges credits; decide when to reserve, charge or refund in that business service. Product-specific screens and workflows are not included in the default customer area.", zh: "提示词、用户权限、对话历史、任务流程与收费规则属于具体产品。如果功能按积分收费，可以复用积分账本，在业务服务中决定何时预占、扣费或退款。具体业务的界面与流程不包含在默认用户区。" }
    ],
    code: "APP_HOST_AI_ENABLED=true\nAPP_HOST_AI_BASE_URL=https://api.openai.com/v1\nAPP_HOST_AI_API_KEY=<your-server-side-provider-key>\nAPP_HOST_AI_MODEL=<your-compatible-chat-model>\nAPP_HOST_AI_MAX_OUTPUT_TOKENS=1024",
    links: [{ href: "https://github.com/brizenchi/go-modules/tree/main/foundation/aix", label: { en: "Read the adapter and usage example", zh: "查看适配器与接入示例" } }, { href: "/docs/launch", label: { en: "Define your product rules", zh: "定义自己的产品规则" } }]
  },
  {
    id: "payments",
    group: "configure",
    summary: { en: "Connect Stripe prices and webhooks, test checkout, and prepare for live payments.", zh: "连接 Stripe 价格与 Webhook，测试结账流程，再准备正式收款。" },
    title: { en: "Configure your plans and Stripe", zh: "配置自己的套餐与 Stripe" },
    paragraphs: [
      { en: "Create Stripe test products and prices, then configure matching keys, price IDs and the webhook signing secret in the deployment environment. Keep plan descriptions aligned with the catalog, configure the webhook endpoint and Customer Portal, and restart the backend after deployment configuration changes. Website settings shows configuration status without exposing keys.", zh: "先在 Stripe 测试环境创建商品与价格，再通过部署环境配置同一环境的密钥、价格编号和 Webhook 签名密钥。让套餐文案与商品一致，配置 Webhook 接收地址和 Customer Portal，修改部署配置后重启后端。网站设置展示配置状态，不展示密钥。" },
      { en: "Use Stripe’s official test card only in its test environment: 4242 4242 4242 4242, a future expiry, and any three-digit CVC. When asked for billing details, complete the required test form fields. A test-mode purchase does not charge a real card.", zh: "仅在 Stripe 测试环境使用官方测试卡：4242 4242 4242 4242，填写未来有效期和任意三位 CVC；如果表单要求账单信息，补齐相应测试字段。测试模式的购买不会扣取真实银行卡资金。" },
      { en: "NEXT_PUBLIC_DEMO_MODE controls the on-page demo notice only. The backend Stripe credentials determine whether payments are in test or live mode. Missing payment configuration disables checkout; switching to live credentials can create real charges. Complete the test journey before configuring live products and keys.", zh: "NEXT_PUBLIC_DEMO_MODE 只控制页面的演示提示。支付处于测试还是正式模式，由后端 Stripe 凭证决定；缺少支付配置时无法发起结账，换成正式凭证后可能产生真实扣款。先完成测试流程，再配置正式商品和密钥。" }
    ],
    items: [
      { title: { en: "After checkout", zh: "购买完成后" }, body: { en: "Return to billing and allow the payment result to sync before refreshing your subscription and invoices. If your plan has not updated, refresh shortly or contact the site owner.", zh: "返回订阅管理，等待支付结果同步，再刷新套餐与账单。如果暂时没有更新，可以稍后重试或联系站点负责人。" } },
      { title: { en: "Test a declined payment", zh: "测试支付失败" }, body: { en: "In Stripe test mode only, use 4000 0000 0000 0002 to simulate a declined card. Check that the failed payment does not grant paid access.", zh: "仅在 Stripe 测试模式使用 4000 0000 0000 0002 模拟银行卡拒付，并检查失败的支付没有开通付费权益。" } }
    ],
    code: "APP_BILLING_ENABLED=true\nAPP_BILLING_STRIPE_SECRET_KEY=sk_test_...\nAPP_BILLING_STRIPE_WEBHOOK_SECRET=whsec_...\nAPP_BILLING_STRIPE_PRICES_PRO_MONTHLY=price_...\nAPP_BILLING_STRIPE_PRICES_CREDITS=price_...",
    links: [
      { href: "/billing", label: { en: "Check subscriptions and billing", zh: "检查订阅和账单" } },
      { href: "https://docs.stripe.com/keys", label: { en: "Get Stripe API keys", zh: "获取 Stripe API 密钥" } },
      { href: "https://docs.stripe.com/testing", label: { en: "Stripe testing reference", zh: "Stripe 测试说明" } }
    ]
  },
  {
    id: "invitations",
    group: "configure",
    summary: { en: "Configure referral conditions and verify attribution, activation, and credit rewards.", zh: "设置邀请条件，核对注册归因、激活状态与积分奖励。" },
    title: { en: "Set invitation rules and verify rewards", zh: "配置邀请规则，验证奖励到账" },
    paragraphs: [
      { en: "Enable referrals with APP_REFERRAL_ENABLED, set APP_REFERRAL_BASE_LINK to your frontend /invite?ref= address, and set the activation window with APP_REFERRAL_ACTIVATION_WINDOW_DAYS. In internal/bootstrap/host_hooks.go, onReferralRegistered grants 20 / 10 credits immediately, and onReferralActivated grants another 20 / 10 on first qualifying payment. Amounts are fixed in code.", zh: "通过 APP_REFERRAL_ENABLED 开启邀请，将 APP_REFERRAL_BASE_LINK 设为自己前端的 /invite?ref= 地址，用 APP_REFERRAL_ACTIVATION_WINDOW_DAYS 设置激活期限。internal/bootstrap/host_hooks.go 的 onReferralRegistered 在注册后给双方 20 / 10，onReferralActivated 在首次符合条件的付费后再给双方 20 / 10，无需金额配置。" },
      { en: "A new account created through an invitation starts as pending. Its first qualifying paid subscription or lifetime purchase can activate the invitation. A free trial must convert to a paid subscription; buying a credit pack alone does not qualify. The activation deadline also applies. Each qualifying invitation grants 20 credits to the inviter and 10 to the invitee, once.", zh: "新用户通过邀请注册后，关系先记为“待激活”。首次符合条件的订阅付款或终身套餐付款可以激活邀请；免费试用需要转为付费，单独购买积分包不会触发奖励，同时还需满足激活期限。符合条件的邀请激活后，邀请人获得 20 积分，被邀请人获得 10 积分，仅发一次。" },
      { en: "Test with two separate accounts and compare the inviter’s referral status and credit history after the qualifying purchase. Publish your reward amount and conditions so customers know what to expect. Third-party affiliate fees the template author may receive from marked recommendations are a separate mechanism.", zh: "使用两个独立账号测试，在符合条件的购买完成后，对照邀请人的邀请状态和积分流水。上线时公开奖励数额与条件，让用户知道什么情况下会获得奖励。模板作者通过已标注的第三方服务推荐链接可能获得的佣金，是另一种独立机制。" }
    ],
    code: "APP_REFERRAL_ENABLED=true\nAPP_REFERRAL_BASE_LINK=https://app.example.com/invite?ref=\nAPP_REFERRAL_ACTIVATION_WINDOW_DAYS=30",
    links: [{ href: "/referrals", label: { en: "Open invitations", zh: "打开邀请页面" } }]
  },
  {
    id: "content",
    group: "launch",
    summary: { en: "Replace starter branding and publish your own guides, articles, and support content.", zh: "替换模板品牌，发布自己的指南、文章与支持内容。" },
    title: { en: "Replace the public content", zh: "换成你的产品介绍与支持信息" },
    paragraphs: [
      { en: "Use website settings for your brand, bilingual heading, service introduction and support contacts. The source presentation defaults remain in content/site.ts; customize product-specific buttons, FAQ and artwork there. Public branding and metadata use the saved website identity.", zh: "在网站设置维护品牌、中英文首页标题、服务介绍和支持渠道。content/site.ts 保留默认展示资料，产品专属按钮、FAQ 与图片可在源码中定制。公开品牌和元信息使用已保存的网站资料。" },
      { en: "Use article management to import Markdown, edit drafts and publish each language independently. English uses the original route and Chinese uses /zh. Only published translations appear in language links and the sitemap; drafts stay private. Release notes and legal policies retain their dedicated source files.", zh: "在文章管理导入 Markdown，编辑草稿，并按语言分别发布。英文保留原路径，中文使用 /zh；语言链接和站点地图只收录已发布的翻译，草稿保持私有。更新日志与法律条款仍使用专用源码文件。" }
    ],
    links: [
      { href: "/blog", label: { en: "Read the example guides", zh: "阅读示例指南" } },
      { href: "/contact", label: { en: "Check support information", zh: "检查支持信息" } }
    ]
  },
  {
    id: "launch",
    group: "launch",
    summary: { en: "Add your core business feature and review the customer journey before going live.", zh: "接入核心业务功能，并在上线前检查完整用户流程与生产配置。" },
    title: { en: "What remains before your product launches", zh: "上线自己的产品，还需要完成什么" },
    paragraphs: [
      { en: "The template includes basic operational metrics: total users, subscriptions, payments, and invitation records. It does not yet include DAU, retention analysis, or project analytics. Those need your product’s own activity events and project model before they can be measured meaningfully.", zh: "模板已提供基础运营指标，可以查看用户总数、订阅、支付和邀请记录。目前还没有内置 DAU（日活跃用户数）、留存分析或项目数量分析。这些需要先定义你的产品中什么算活跃、什么是项目，再接入对应业务数据。" },
      { en: "The source is free to obtain. Hosting, database, email delivery, and payment-provider usage may have their own costs. Connect accounts you own and choose the services that fit your deployment.", zh: "模板源码免费获取，部署、数据库、邮件发送和支付服务可能产生各自的费用。接入你自己的服务账号，并按实际部署需要选择服务。" }
    ],
    items: [
      { title: { en: "Deliver your core feature", zh: "让核心业务可以使用" }, body: { en: "Give a new customer a clear route from signup to the result you sell. Define which features require a subscription or consume credits.", zh: "让新用户从注册到获得你出售的结果有一条清楚的路径，并定义哪些功能需要订阅、哪些操作消耗积分。" } },
      { title: { en: "Check customer and administrator access", zh: "验收用户与管理员流程" }, body: { en: "Check signup, sign-in, subscriptions, invoices, invitations, rewards, and logout with separate test accounts. Confirm that ordinary users can only access their own data and cannot enter /admin.", zh: "用独立测试账号验证注册、登录、订阅、账单、邀请、奖励和退出。确认普通用户只能访问自己的数据，不能进入 /admin。" } },
      { title: { en: "Publish with your own configuration", zh: "使用自己的配置发布" }, body: { en: "Use production configuration for your domains, database, mail, and secrets. Verify the payment environment, complete your public policies and support details, and replace the sample content before opening the product to customers.", zh: "使用适合正式环境的域名、数据库、邮件和密钥配置，确认支付环境，完善公开条款与支持信息，替换示例内容后再向客户开放。" } }
    ]
  }
];

export function findDocumentation(slug: string): DocumentationSection | undefined {
  return documentation.find((guide) => guide.id === slug);
}

export function searchDocumentation(query: string, locale: Locale): DocumentationSection[] {
  const terms = query.trim().toLocaleLowerCase().split(/\s+/).filter(Boolean);
  return documentation.filter((guide) => {
    const text = [
      guide.title[locale], guide.summary[locale],
      ...guide.paragraphs.map((paragraph) => paragraph[locale]),
      ...(guide.items ?? []).flatMap((item) => [item.title[locale], item.body[locale]]),
      ...(guide.links ?? []).map((link) => link.label[locale]),
      guide.code ?? ""
    ].join(" ").toLocaleLowerCase();
    return terms.every((term) => text.includes(term));
  });
}
