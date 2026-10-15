/** Public presentation defaults. Edit here, then rebuild both language routes. */
export const siteConfig = {
  brand: {
    name: "SaaS Starter",
    subtitle: { en: "Next.js + Go", zh: "Next.js + Go" },
    attribution: { en: "Built with go-modules", zh: "基于 go-modules 构建" }
  },
  repositoryURL: "https://github.com/brizenchi/go-modules",
  description: {
    en: "A free Next.js + Go SaaS starter with authentication, Stripe billing, credits, referrals, and server-side AI integration. Start with connected workflows and build your own product.",
    zh: "免费的 Next.js + Go SaaS 模板，整合登录、Stripe 计费、积分、邀请和服务端 AI 接入。从已接通的流程开始，构建你自己的产品。"
  },
  homepageTitle: { en: "Free Next.js + Go SaaS starter", zh: "免费的 Next.js + Go SaaS 模板" },
  hero: {
    eyebrow: { en: "FREE SAAS STARTER · NEXT.JS + GO", zh: "免费 SaaS 模板 · Next.js + Go" },
    title: { en: "Your next SaaS starts here.", zh: "从这里，开始你的 SaaS。" },
    description: {
      en: "Authentication, billing and integrations, already connected. Start with the free source and make room for the product only you can build.",
      zh: "登录、计费、服务接入，通用能力已经备好。从免费源码开始，把时间留给自己的产品。"
    },
    primaryAction: { href: "/account", label: { en: "Try the account center", zh: "体验账户中心" } },
    secondaryAction: { href: "#setup", label: { en: "Get the template", zh: "获取模板" } },
    note: { en: "Free source · Self-hosted · English & Chinese", zh: "免费源码 · 独立部署 · 中英双语" }
  },
  preview: {
    en: { src: "/previews/account-en.png", alt: "Actual customer account center with subscription, credit and referral services. Demonstration account." },
    zh: { src: "/previews/account-zh.png", alt: "真实账户中心，包含订阅、积分与推荐奖励入口，使用演示账号。" },
    caption: { en: "The included account center, with a demonstration account.", zh: "模板自带的账户中心 · 演示账号截图" }
  },
  footer: {
    description: { en: "Less groundwork. More room for your next idea. A free Next.js + Go SaaS starter.", zh: "少一些重复搭建，多一些专注业务。免费的 Next.js + Go SaaS 模板。" }
  },
  sharing: {
    title: "Your next SaaS starts here.",
    description: "Authentication · Billing · Credits · AI Integration",
    label: "Free SaaS starter",
    stack: "Next.js + Go"
  },
  faq: [
    { question: { en: "Is the template free?", zh: "模板免费吗？" }, answer: { en: "Yes. Get the source and deploy your own product. Hosting, email, payments and AI providers may charge for their services.", zh: "是的，可以获取源码并自行部署产品。服务器、邮件、支付和 AI 服务商可能收取各自的服务费用。" } },
    { question: { en: "What do the prices on this site represent?", zh: "价格页展示的是什么？" }, answer: { en: "They are sample offers for a SaaS built from the template. You do not need to purchase a plan to obtain the source.", zh: "价格页演示使用模板构建的 SaaS 可以如何收费。获取模板源码无需购买套餐。" } },
    { question: { en: "Does the template include a chat product?", zh: "模板提供现成的聊天产品吗？" }, answer: { en: "It provides server-side model integration. Your product defines its interface, conversation history, prompts and usage policy. The template does not expose a default chat product.", zh: "模板提供服务端模型调用接入。聊天界面、对话记录、提示词和用量规则由具体产品实现，模板不默认提供聊天业务。" } },
    { question: { en: "How do I make it my own?", zh: "如何换成自己的产品？" }, answer: { en: "Set your brand and homepage copy in content/site.ts, configure your services, and add your business feature. NEXT_PUBLIC_APP_NAME can override the default brand at build time.", zh: "在 content/site.ts 修改品牌和首页主文案，配置所需服务，再接入自己的核心业务。NEXT_PUBLIC_APP_NAME 可在构建时覆盖默认品牌。" } }
  ]
};

/** Homepage feature and journey copy, kept out of the page component. */
export function getHomeContent(demo: boolean) {
  return {
    zh: {
      features: [
        { label: "01", title: "用户认证", description: "邮箱验证码，以及配置后可用的 Google、GitHub 登录。账户信息只读，日常操作保持简洁。", detail: "邮箱 · OAuth · 账户弹窗", href: "/features/authentication" },
        { label: "02", title: "订阅与积分", description: "Stripe 订阅、一次性购买和积分账本，连接购买、账单与用量记录。", detail: "月付 / 年付 · 积分流水", href: "/features/subscriptions" },
        { label: "03", title: "邀请与邮件", description: "邀请链接连接注册归因与奖励，Resend 负责配置后的验证码和通知。", detail: "邀请奖励 · 邮件通知", href: "/docs/invitations" },
        { label: "04", title: "公开产品官网", description: "功能、场景、示例、文档与博客，配套双语路由与分享信息。", detail: "中英双语 · 内容与 SEO", href: "/resources" },
        { label: "05", title: "AI 接入能力", description: "配置服务商与模型，在自己的业务中调用。对话、生成任务与收费规则由产品定义。", detail: "模型调用 · 业务自行实现", href: "/docs/ai-integration" },
        { label: "06", title: "管理与扩展", description: "独立管理员后台，配套通用表格、表单，以及可选 GA4 和 Crisp 接入。", detail: "后台管理 · 可选统计与客服", href: "/docs/admin" }
      ],
      steps: [
        { title: "创建账号", description: "通过可用的邮箱或第三方登录方式，进入自己的账户。", href: "/login", action: "注册 / 登录" },
        { title: "选择套餐", description: "查看订阅、终身访问和积分包示例，了解产品可以如何收费。", href: "/pricing", action: "查看套餐示例" },
        { title: demo ? "体验测试购买" : "管理订阅", description: demo ? "确认 Stripe 收银台显示测试模式后，按指南使用测试卡走完购买流程。" : "在收银台确认订单和金额，返回后查看账单与订阅状态。", href: demo ? "/docs/try-demo#details" : "/billing", action: demo ? "阅读体验指南" : "打开订阅管理" },
        { title: "邀请新用户", description: "复制个人邀请链接，分享给尚未注册的用户，再查看邀请记录。", href: "/referrals", action: "打开推荐中心" }
      ],
      routes: [
        { title: "账户", href: "/account" },
        { title: "订阅管理", href: "/billing" },
        { title: "积分", href: "/credits" },
        { title: "邀请推荐", href: "/referrals" }
      ],
      flow: [
        { title: "接入公共能力", description: "复用登录、计费、AI 与存储接入。", label: "连接服务" },
        { title: "实现核心业务", description: "由你定义对话、生成任务或其他功能。", label: "构建产品" },
        { title: "定义收费规则", description: "按业务接入会员权益或积分服务。", label: "连接计费" }
      ]
    },
    en: {
      features: [
        { label: "01", title: "Authentication", description: "Email codes and optional Google or GitHub sign-in. A read-only account dialog keeps everyday account access simple.", detail: "Email · OAuth · Account dialog", href: "/features/authentication" },
        { label: "02", title: "Billing & credits", description: "Stripe subscriptions, one-time purchases and a credit ledger connect checkout, invoices and usage records.", detail: "Monthly / yearly · Credit history", href: "/features/subscriptions" },
        { label: "03", title: "Referrals & email", description: "Invite links connect signup and rewards. Configure Resend for verification codes and notifications.", detail: "Referral rewards · Email delivery", href: "/docs/invitations" },
        { label: "04", title: "Your public website", description: "Features, use cases, examples, docs and a blog, with bilingual routes and sharing metadata.", detail: "Two languages · Content & SEO", href: "/resources" },
        { label: "05", title: "AI integration", description: "Connect your own provider and call the model from your business service. Your product owns conversations, tasks and charging.", detail: "Model calls · Your own product rules", href: "/docs/ai-integration" },
        { label: "06", title: "Admin & extensions", description: "A separate admin console, shared tables and forms, plus optional GA4 analytics and Crisp support.", detail: "Administration · Optional integrations", href: "/docs/admin" }
      ],
      steps: [
        { title: "Create an account", description: "Use an available email or social sign-in option to open your own account.", href: "/login", action: "Sign up / sign in" },
        { title: "Explore the plans", description: "See examples of subscriptions, lifetime access, and credits for your own product.", href: "/pricing", action: "View example plans" },
        { title: demo ? "Try a test purchase" : "Manage a subscription", description: demo ? "Check that Stripe Checkout shows test mode, then follow the guide to pay with a test card." : "Confirm your order and amount at Checkout, then return to view invoices and subscription status.", href: demo ? "/docs/try-demo#details" : "/billing", action: demo ? "Read the demo guide" : "Open billing" },
        { title: "Invite someone new", description: "Share your personal link with someone who has not signed up, then review your referral history.", href: "/referrals", action: "Open referrals" }
      ],
      routes: [
        { title: "Account", href: "/account" },
        { title: "Billing", href: "/billing" },
        { title: "Credits", href: "/credits" },
        { title: "Referrals", href: "/referrals" }
      ],
      flow: [
        { title: "Connect shared capabilities", description: "Reuse authentication, billing, AI and storage adapters.", label: "Connect services" },
        { title: "Build your core feature", description: "Define conversations, generation tasks or another useful workflow.", label: "Build the product" },
        { title: "Define your charging rules", description: "Apply entitlements or credits to your own operations.", label: "Connect billing" }
      ]
    }
  };
}
