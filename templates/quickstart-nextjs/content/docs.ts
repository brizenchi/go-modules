import type { ArticleSection, LocalizedText } from "./articles";

export type DocumentationSection = ArticleSection & {
  items?: { title: LocalizedText; body: LocalizedText }[];
  ordered?: boolean;
  code?: string;
  links?: { href: string; label: LocalizedText }[];
};

export const documentation: DocumentationSection[] = [
  {
    id: "overview",
    title: { en: "What you can build with this template", zh: "这个模板能帮你做什么" },
    paragraphs: [
      { en: "This is a free Next.js + Go starter for building your own SaaS: a website where customers register, use your service, and manage a paid subscription. It connects the public website, customer account center, and administrator console so you can focus on the service you want to sell.", zh: "这是一个免费的 Next.js + Go SaaS 启动模板，用来搭建你自己的在线产品：用户注册、使用服务、购买和管理订阅。模板已经连接好公开网站、用户工作台和管理员后台，你可以把开发精力放在真正要出售的服务上。" },
      { en: "You get the source code and connected example workflows. Add your own core feature, replace the brand and sample plans, and connect your service accounts before launching. The prices shown here demonstrate how your future product can charge customers; they are not a price for downloading the template.", zh: "你拿到的是源码和已经串联好的示例流程。上线前，需要加入自己的核心业务、替换品牌和示例套餐，并接入自己的服务账号。这里展示的套餐价格，是你未来的产品如何收费的示例；下载模板本身免费。" }
    ],
    links: [
      { href: "https://github.com/brizenchi/go-modules/tree/main/templates", label: { en: "Get the template source", zh: "获取模板源码" } },
      { href: "/docs#try-demo", label: { en: "Try the customer journey", zh: "先体验用户流程" } },
      { href: "/docs#domains", label: { en: "Set up your own project", zh: "开始配置自己的项目" } }
    ]
  },
  {
    id: "features",
    title: { en: "What is included, and where to find it", zh: "已经有什么功能，分别在哪里" },
    paragraphs: [
      { en: "Each part has a different audience. Your customers manage their own accounts; the site owner uses a separate administrator console to run the service.", zh: "不同页面服务不同的人。你的客户在工作台管理自己的账号；网站所有者使用独立的管理员后台运营整个网站。" }
    ],
    items: [
      { title: { en: "Public website", zh: "公开网站 · 向访客介绍产品" }, body: { en: "Overview, pricing, setup guide, searchable blog, updates, contact, privacy, and terms pages. Chinese and English copy, sharing metadata, and a sitemap are included.", zh: "总览、价格、使用指南、可搜索的博客、更新记录、联系、隐私与条款页面，带有中英文文案、分享信息和站点地图。" } },
      { title: { en: "Customer account center", zh: "用户工作台 · 管理自己的账户" }, body: { en: "Account settings at /account, subscriptions and invoices at /billing, and invitation links, progress, and rewards at /referrals. Email-code login and optional Google/GitHub sign-in connect to the Go API.", zh: "在 /account 管理账户资料，在 /billing 管理自己的订阅和账单，在 /referrals 分享邀请链接、查看进展与奖励。支持邮箱验证码登录，也可配置 Google / GitHub 登录。" } },
      { title: { en: "Administrator console", zh: "管理员后台 · 运营整个网站" }, body: { en: "At /admin, authorized administrators can view users, payment records, subscriptions, referrals, credit records, and audit logs, and edit public brand, support, and example export settings. Ordinary accounts cannot access it.", zh: "在 /admin 查看全站用户、支付记录、订阅、邀请、积分和操作审计，并修改品牌、支持渠道和示例导出定价。需要管理员权限，普通用户无法进入。" } },
      { title: { en: "Examples you can extend", zh: "业务示例 · 改造成你自己的产品" }, body: { en: "Use /notes for free note creation and credit-based exports, /credits for balance and transaction history, and /files for private image uploads when enabled. They demonstrate how to connect a customer action to ownership, billing, and records.", zh: "在 /notes 体验免费创建笔记、用积分导出，在 /credits 查看余额和流水，在启用上传后使用 /files 保存私有图片。这些示例展示如何把用户操作、数据归属和计费记录接起来，你可以据此开发自己的功能。" } }
    ],
    links: [{ href: "/docs#launch", label: { en: "See what to add before launch", zh: "查看上线前还需要补什么" } }]
  },
  {
    id: "try-demo",
    title: { en: "Try a complete customer journey", zh: "按这三步，体验完整用户流程" },
    paragraphs: [
      { en: "Registration, account changes, and invitation records use the backend and its database. Payment testing uses Stripe’s test environment when configured. The isolated local preview keeps temporary data and has payments disabled, so a checkout cannot be completed there.", zh: "注册、账户修改和邀请记录会经过后端并写入数据库。支付体验需要接入 Stripe 测试环境。隔离的本地 preview 使用临时数据，支付默认关闭，因此不能在其中完成购买。" }
    ],
    ordered: true,
    items: [
      { title: { en: "Create your first account", zh: "注册并进入工作台" }, body: { en: "Open the account center and use an available sign-in method. With email login, request a code and enter it; a new address creates an account. Check your profile and try signing out and back in.", zh: "进入用户工作台，选择页面提供的登录方式。使用邮箱时，发送并输入验证码；新邮箱会创建账号。查看账户资料，再退出并重新登录，确认完整流程可用。" } },
      { title: { en: "Try a subscription in test mode", zh: "在测试环境购买一次订阅" }, body: { en: "Open subscriptions, choose a plan, and continue to Checkout. Only after confirming Stripe test mode, use 4242 4242 4242 4242, a future expiry date, and any three-digit CVC. Return to subscriptions to check the result. If payments are unavailable, the owner must configure them first.", zh: "打开订阅页面，选择套餐并进入 Checkout。确认使用 Stripe 测试模式后，填写 4242 4242 4242 4242、未来有效期和任意三位 CVC。支付后回到订阅页查看结果；如果支付不可用，需要先由站点所有者完成配置。" } },
      { title: { en: "Invite a separate new account", zh: "邀请一个独立的新账号" }, body: { en: "Copy your link from referrals. Open it in a separate browser profile or private window and register with a different email. Check the inviter’s pending record, then have the new account complete a qualifying test purchase and check the activation and reward.", zh: "到邀请页面复制链接，在另一浏览器个人资料或无痕窗口中打开，用不同邮箱注册新账号。先在邀请人账号确认“待激活”记录，再让被邀请人完成符合条件的测试购买，检查激活状态和奖励。" } }
    ],
    links: [
      { href: "/account", label: { en: "Start with an account", zh: "从注册账号开始" } },
      { href: "/docs#invitations", label: { en: "Read the reward conditions", zh: "查看邀请奖励条件" } },
      { href: "https://docs.stripe.com/testing", label: { en: "Stripe’s official test cards", zh: "Stripe 官方测试卡说明" } }
    ]
  },
  {
    id: "domains",
    title: { en: "1. Connect your frontend and backend", zh: "1. 准备项目，连接前后端" },
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
    title: { en: "2. Enable customer sign-in with Resend", zh: "2. 接入 Resend，配置用户登录" },
    paragraphs: [
      { en: "Resend is already integrated for email delivery. Add your own API key and verified sender on the backend, then enable email-code sign-in. The local example uses the log provider and debug codes; use Resend and turn debug codes off for a deployed service.", zh: "模板已经集成 Resend 邮件发送。你需要在后端填写自己的 API key 和经过验证的发件人，再开启邮箱验证码登录。本地示例使用日志邮件和调试验证码；部署服务时改用 Resend 并关闭调试验证码。" },
      { en: "Google and GitHub sign-in are optional. Configure their client credentials and callback URLs on the backend when you need them. The frontend shows methods enabled by the API. Verify sign-in, account creation, and sign-out with an email you control before inviting customers.", zh: "Google 和 GitHub 登录按需启用，在后端填写对应应用凭证和回调地址即可。前端会读取 API 提供的登录方式。邀请真实客户之前，先用自己的邮箱验证收码、注册、登录和退出。" }
    ],
    code: "APP_AUTH_EMAIL_ENABLED=true\nAPP_AUTH_EMAIL_DEBUG=false\nAPP_EMAIL_PROVIDER=resend\nAPP_EMAIL_RESEND_API_KEY=<your-resend-api-key>\nAPP_EMAIL_RESEND_SENDER_EMAIL=<your-verified-sender>",
    links: [{ href: "/account", label: { en: "Check customer sign-in", zh: "检查用户登录" } }]
  },
  {
    id: "admin",
    title: { en: "3. Set your administrator email and password", zh: "3. 设置管理员邮箱和密码" },
    paragraphs: [
      { en: "Set both variables below in the backend environment, restart the backend, and open /admin. Sign in with that email and password; you do not need to register the administrator account first. There is no shared default administrator password. Use a generated password of 12–72 bytes; ASCII characters make the byte length easy to check.", zh: "在后端环境变量中同时填写下面两项，重启后端，再打开 /admin，用设置的邮箱和密码登录，无需提前注册管理员账号。模板没有通用的默认管理员密码。密码要求为 12–72 字节，建议使用随机生成的英数符号密码，便于确认长度。" },
      { en: "In site settings, update the public brand, description, support channels, and example export cost. Provider credentials, database settings, and administrator passwords stay in backend configuration. The customer account center is for each customer’s own profile, subscriptions, and invitations.", zh: "登录后，在网站设置中修改公开品牌、简介、支持渠道和示例导出价格。服务密钥、数据库配置和管理员密码继续放在后端配置中。用户工作台则用于每个客户管理自己的资料、订阅和邀请。" }
    ],
    code: "APP_AUTH_ADMIN_EMAIL=<your-admin-email>\nAPP_AUTH_ADMIN_PASSWORD=<your-generated-password>",
    links: [{ href: "/admin", label: { en: "Open administrator sign-in", zh: "打开管理员登录" } }]
  },
  {
    id: "payments",
    title: { en: "4. Configure your plans and Stripe", zh: "4. 配置自己的套餐与 Stripe" },
    paragraphs: [
      { en: "Create your products and prices in Stripe’s test environment. Add the matching secret key, price IDs, and webhook signing secret to the backend .env. Keep the frontend plan descriptions aligned with the catalog. Set the webhook destination to https://your-api-domain/api/v1/stripe/webhook and enable Stripe Customer Portal. Payment events must reach the backend before subscriptions and invitation rewards can update.", zh: "先在 Stripe 测试环境创建商品与价格。在后端 .env 填写同一环境的 secret key、价格 ID 和 webhook 签名密钥，并让前端套餐文案与商品配置一致。Webhook 地址为 https://你的后端域名/api/v1/stripe/webhook，并在 Stripe 启用 Customer Portal。支付事件送达后端后，订阅状态和邀请奖励才会更新。" },
      { en: "Use Stripe’s official test card only in its test environment: 4242 4242 4242 4242, a future expiry, and any three-digit CVC. When asked for billing details, complete the required test form fields. A test-mode purchase does not charge a real card.", zh: "仅在 Stripe 测试环境使用官方测试卡：4242 4242 4242 4242，填写未来有效期和任意三位 CVC；如果表单要求账单信息，补齐相应测试字段。测试模式的购买不会扣取真实银行卡资金。" },
      { en: "NEXT_PUBLIC_DEMO_MODE controls the on-page demo notice only. The backend Stripe credentials determine whether payments are in test or live mode. Missing payment configuration disables checkout; switching to live credentials can create real charges. Complete the test journey before configuring live products and keys.", zh: "NEXT_PUBLIC_DEMO_MODE 只控制页面的演示提示。支付处于测试还是正式模式，由后端 Stripe 凭证决定；缺少支付配置时无法发起结账，换成正式凭证后可能产生真实扣款。先完成测试流程，再配置正式商品和密钥。" }
    ],
    code: "APP_BILLING_ENABLED=true\nAPP_BILLING_STRIPE_SECRET_KEY=sk_test_...\nAPP_BILLING_STRIPE_WEBHOOK_SECRET=whsec_...\nAPP_BILLING_STRIPE_PRICES_PRO_MONTHLY=price_...\nAPP_BILLING_STRIPE_PRICES_CREDITS=price_...",
    links: [
      { href: "/billing", label: { en: "Check subscriptions and billing", zh: "检查订阅和账单" } },
      { href: "https://docs.stripe.com/testing", label: { en: "Stripe testing reference", zh: "Stripe 测试说明" } }
    ]
  },
  {
    id: "invitations",
    title: { en: "5. Set invitation rules and verify rewards", zh: "5. 配置邀请规则，验证奖励到账" },
    paragraphs: [
      { en: "Enable referrals with APP_REFERRAL_ENABLED, set APP_REFERRAL_BASE_LINK to your frontend /invite?ref= address, and define APP_REFERRAL_ACTIVATION_REWARD and APP_REFERRAL_ACTIVATION_WINDOW_DAYS. These are product credits awarded for a qualifying invitation.", zh: "通过 APP_REFERRAL_ENABLED 开启邀请，将 APP_REFERRAL_BASE_LINK 设为自己前端的 /invite?ref= 地址，再用 APP_REFERRAL_ACTIVATION_REWARD 设置奖励积分、APP_REFERRAL_ACTIVATION_WINDOW_DAYS 设置激活期限。奖励发放的是你产品内的积分。" },
      { en: "A new account created through an invitation starts as pending. Its first qualifying paid subscription or lifetime purchase can activate the invitation. A free trial must convert to a paid subscription; buying a credit pack alone does not qualify. The activation deadline and configured rules also apply.", zh: "新用户通过邀请注册后，关系先记为“待激活”。首次符合条件的订阅付款或终身套餐付款可以激活邀请；免费试用需要转为付费，单独购买积分包不会触发奖励，同时还需满足激活期限和站点配置的规则。" },
      { en: "Test with two separate accounts and compare the inviter’s referral status and credit history after the qualifying purchase. Publish your reward amount and conditions so customers know what to expect. Third-party affiliate fees the template author may receive from marked recommendations are a separate mechanism.", zh: "使用两个独立账号测试，在符合条件的购买完成后，对照邀请人的邀请状态和积分流水。上线时公开奖励数额与条件，让用户知道什么情况下会获得奖励。模板作者通过已标注的第三方服务推荐链接可能获得的佣金，是另一种独立机制。" }
    ],
    code: "APP_REFERRAL_ENABLED=true\nAPP_REFERRAL_BASE_LINK=https://app.example.com/invite?ref=\nAPP_REFERRAL_ACTIVATION_REWARD=50\nAPP_REFERRAL_ACTIVATION_WINDOW_DAYS=30",
    links: [{ href: "/referrals", label: { en: "Open invitations", zh: "打开邀请页面" } }]
  },
  {
    id: "content",
    title: { en: "6. Replace the public content", zh: "6. 换成你的产品介绍与支持信息" },
    paragraphs: [
      { en: "Explain the problem your product solves on the overview page and describe exactly what each plan includes. Replace the template’s example business workflow with your own. In the administrator’s site settings, add a support email or HTTPS help page that you maintain.", zh: "在总览页写清楚自己的产品解决什么问题，在套餐页说明每档价格包含什么，并把模板的示例业务替换成自己的功能。在管理员的网站设置中填写有人维护的支持邮箱或 HTTPS 帮助页面。" },
      { en: "Edit content/articles.ts for bilingual blog posts, content/updates.ts for releases, and content/policies.ts for your actual privacy and terms. CONTENT.md covers publishing and metadata. Chinese and English share the same page URL; public pages include sharing metadata and a sitemap, while account and admin routes are excluded.", zh: "在 content/articles.ts 编辑中英文博客，在 content/updates.ts 发布更新，在 content/policies.ts 按实际业务完善隐私与条款。CONTENT.md 说明发布与元信息配置。中英文使用同一页面 URL；公开页面带有分享信息和站点地图，账户及管理路由不会出现在站点地图中。" }
    ],
    links: [
      { href: "/blog", label: { en: "Read the example guides", zh: "阅读示例指南" } },
      { href: "/contact", label: { en: "Check support information", zh: "检查支持信息" } }
    ]
  },
  {
    id: "launch",
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
