"use client";

import Link from "next/link";
import { useState } from "react";
import { SiteShell } from "@/components/site-shell";
import { CTAButton, DetailRows, PageSection } from "@/components/ui";
import { FeatureCard, MetricCard } from "@/components/marketing";
import { appEnv } from "@/lib/env";
import { useI18n } from "@/lib/i18n";
import styles from "./overview.module.css";

const testCard = "4242 4242 4242 4242";

export default function HomePage() {
  const { t } = useI18n();
  const [copyState, setCopyState] = useState<"idle" | "copied" | "error">("idle");
  const demo = appEnv.demoMode;
  const content = t({
    zh: {
      eyebrow: "免费 SaaS 启动模板 · Next.js + Go",
      title: "用一套免费模板，上线你的 SaaS。",
      description: "Template 为独立开发者和小团队准备了 SaaS 的基础功能：登录、订阅支付、邀请推荐、积分和管理员后台。你拿到的是配套的 Next.js 前端与 Go 后端源码；加入自己的业务、配置服务并部署，就能把想法做成可以向用户收费的产品。",
      start: "先体验用户流程", configure: "用模板做自己的产品",
      included: "已整合的功能", analytics: "运营后台与数据", tryIt: "怎么体验", payment: "怎么测试支付", sharing: "怎么测试分享", setup: "怎么配置上线",
      features: [
        { label: "01 / 用户认证", title: "注册登录，接好就能用", description: "整合邮箱验证码、Google 和 GitHub 登录，支持账号创建、会话管理与退出登录。按需启用登录方式，让用户顺畅进入你的产品。" },
        { label: "02 / 订阅与计费", title: "为你的产品接上付费能力", description: "整合 Stripe Checkout，支持月付、年付、终身套餐与积分包，配套订阅管理和账单查询。根据业务选择合适的收费方式。" },
        { label: "03 / 邀请推荐", title: "从用户分享，到邀请转化", description: "专属邀请链接、注册归因、订阅激活与积分奖励已经串联。用户可以查看邀请记录和统计，为产品的推荐增长提供基础。" },
        { label: "04 / Resend 邮件", title: "把验证和欢迎邮件送到用户手中", description: "已接入 Resend 邮件发送能力。配置发信域名和密钥后即可发送验证码，还可以按需启用欢迎邮件，完善用户的首次体验。" },
        { label: "05 / 产品官网", title: "展示、定价、文档一起准备好", description: "配套首页、价格、可搜索的文档与博客、更新日志和联系页面。中英文内容使用统一的响应式界面，替换品牌与文案即可构建自己的产品门面。" },
        { label: "06 / 管理员后台", title: "让站点运营有自己的入口", description: "独立 /admin 后台查看全站用户、订阅、支付与邀请记录，管理积分和公开网站配置。管理员邮箱与密码由后端配置，普通账号只管理自己的账户。" }
      ],
      metrics: [
        { label: "已内置 / 用户", value: "累计注册用户", detail: "运营后台读取真实账号总数，并支持按邮箱、用户名或账号 ID 查询用户。" },
        { label: "已内置 / 订阅", value: "有效订阅", detail: "查看当前有效订阅数量、套餐状态和支付记录，掌握产品的基本运营情况。" },
        { label: "已内置 / 邀请", value: "待激活与已激活", detail: "查看邀请总量与激活进展，按状态查找记录，并核对已激活邀请的奖励。" },
        { label: "已内置 / 运营操作", value: "积分与网站配置", detail: "发放或退回积分，修改品牌与支持信息，并查询相关操作记录。服务商密钥通过后端环境变量配置。" }
      ],
      steps: [
        { title: "注册并登录", description: "点击登录，使用页面提供的邮箱或第三方登录方式。新用户首次完成验证时会自动创建账号。", result: "完成后：账户页可以看到自己的账号。", href: "/login", action: "注册 / 登录" },
        { title: "体验订阅与计费", description: "价格页展示模板支持的 SaaS 收费方式，不是模板售价。进入订阅管理选择套餐与周期，或体验一次性积分包。", result: "完成后：进入 Stripe 收银台。", href: "/pricing", action: "查看套餐示例" },
        { title: demo ? "完成测试购买" : "购买并管理订阅", description: demo ? "在 Stripe 测试收银台填写下方测试卡。测试模式下的订阅、买断和积分包付款不涉及真实资金。" : "在 Stripe 收银台确认订单与金额。返回后可查看账单，或管理套餐、取消与恢复订阅。", result: "完成后：返回订阅管理，等待支付结果同步后刷新查看。", href: demo ? "#test-payment" : "/billing", action: demo ? "获取测试卡" : "订阅管理" },
        { title: "分享给一个新用户", description: "到推荐中心复制自己的邀请链接，用另一个浏览器或无痕窗口打开，并使用尚未注册的账号完成注册。", result: "完成后：回到原账号查看邀请记录。", href: "/referrals", action: "打开推荐中心" }
      ],
      configuration: [
        { title: "获取前后端源码", description: "从仓库获取模板。templates/quickstart-nextjs 是前端，templates/quickstart 是后端；按各自 README 安装依赖并准备配置。", href: "/docs#overview", action: "从源码开始" },
        { title: "连接数据库、域名和邮件", description: "使用自己的 PostgreSQL 数据库，连接前端与 API，配置 Resend 发信和需要的登录方式，让用户可以真实注册。", href: "/docs#auth-email", action: "配置登录与邮件" },
        { title: "设置你的管理员账号", description: "在后端环境变量里设置管理员邮箱和密码，再登录 /admin 管理品牌、支持信息及全站记录。", href: "/docs#admin", action: "配置管理员登录" },
        { title: "确定收费和邀请规则", description: "选择订阅、终身套餐或积分计费，在 Stripe 测试环境配置价格，并设定符合条件的邀请奖励。", href: "/docs#payments", action: "配置套餐与测试支付" },
        { title: "接入自己的核心业务", description: "把用户真正需要的工具或服务接进来。可从笔记与积分导出示例开始修改，再替换官网、价格和帮助文案。", href: "/docs#features", action: "查看可复用的功能" },
        { title: "验收后部署上线", description: "验证注册、购买、权益和邀请流程，完善隐私与支持信息。正式收费时，将 Stripe 密钥、价格和 Webhook 一起切换到正式环境。", href: "/docs#launch", action: "查看上线检查" }
      ]
    },
    en: {
      eyebrow: "Free SaaS starter · Next.js + Go",
      title: "Launch your SaaS with a free template.",
      description: "Template gives independent developers and small teams the foundations of a SaaS: sign-in, subscriptions, referrals, credits, and an administrator console. Start with the Next.js frontend and Go backend source, add your own service, configure your providers, and deploy a product you can charge for.",
      start: "Try the customer journey", configure: "Build your own product",
      included: "Integrated features", analytics: "Operations and data", tryIt: "Try the template", payment: "Test a payment", sharing: "Test a referral", setup: "Configure and launch",
      features: [
        { label: "01 / Authentication", title: "Give customers a smooth way in", description: "Email verification, Google, and GitHub sign-in connect account creation, session management, and sign-out. Enable the sign-in options your product needs." },
        { label: "02 / Subscriptions and billing", title: "Start with payments already connected", description: "Stripe Checkout supports monthly and yearly subscriptions, lifetime access, and credit packages, with subscription management and invoices. Choose how your product earns revenue." },
        { label: "03 / Referrals", title: "Connect sharing to customer growth", description: "Personal invite links connect sign-up attribution, subscription activation, and credit rewards. Customers can review referral history and statistics." },
        { label: "04 / Resend email", title: "Reach customers from their first sign-in", description: "Resend email delivery is integrated. Configure your sending domain and API key to deliver verification codes, then enable welcome emails when you need them." },
        { label: "05 / Your public website", title: "Present, explain, and price your product", description: "A homepage, pricing, searchable docs and blog, release notes, and contact pages share a responsive English / Chinese interface. Update the brand and content for your product." },
        { label: "06 / Administration", title: "Run your site from a dedicated console", description: "The separate /admin console provides site-wide user, subscription, payment, and referral records, credit adjustments, and public site settings. Configure administrator credentials on the backend; customers manage their own accounts." }
      ],
      metrics: [
        { label: "Included / Users", value: "Registered accounts", detail: "Read the real account total in the operator console and find users by email, username, or account ID." },
        { label: "Included / Subscriptions", value: "Active subscriptions", detail: "Review the active subscription count, plan status, and payment records for day-to-day operations." },
        { label: "Included / Invitations", value: "Pending and activated", detail: "Follow invitation totals and activation progress, filter records by status, and reconcile activated rewards." },
        { label: "Included / Operations", value: "Credits and site settings", detail: "Grant or refund credits, update branding and support details, and review recorded operations. Provider secrets stay in backend environment variables." }
      ],
      steps: [
        { title: "Create your account", description: "Choose an available email or social sign-in option. Completing verification for the first time automatically creates your account.", result: "Then: find your identity on the account page.", href: "/login", action: "Sign up / sign in" },
        { title: "Explore subscriptions and billing", description: "Pricing demonstrates ways to charge for your own SaaS; the template itself is free. Choose a plan and interval in billing, or try a one-time credit package.", result: "Then: continue to Stripe Checkout.", href: "/pricing", action: "Explore example plans" },
        { title: demo ? "Make a test purchase" : "Buy and manage your subscription", description: demo ? "Enter the test card below in Stripe test Checkout. In test mode, subscription, lifetime, and credit package payments move no real money." : "Confirm the order and amount in Stripe Checkout. Return to view invoices, change plans, or cancel and resume a subscription.", result: "Then: return to billing and refresh after the payment result syncs.", href: demo ? "#test-payment" : "/billing", action: demo ? "Get the test card" : "Manage billing" },
        { title: "Invite a new customer", description: "Copy your invite link from the referral center. Open it in another browser or private window and register with an account that has never signed up here.", result: "Then: return to your original account to see the referral.", href: "/referrals", action: "Open referrals" }
      ],
      configuration: [
        { title: "Get the frontend and backend source", description: "Get the template from the repository. templates/quickstart-nextjs is the frontend; templates/quickstart is the backend. Follow each README to install dependencies and prepare configuration.", href: "/docs#overview", action: "Start with the source" },
        { title: "Connect data, domains, and email", description: "Use your own PostgreSQL database, connect the frontend to the API, and configure Resend and your chosen sign-in methods so customers can register.", href: "/docs#auth-email", action: "Set up sign-in and email" },
        { title: "Create your administrator access", description: "Set the administrator email and password in backend environment variables. Sign in at /admin to manage your branding, support information, and site-wide records.", href: "/docs#admin", action: "Configure admin sign-in" },
        { title: "Choose pricing and referral rules", description: "Choose subscriptions, lifetime access, or credit billing. Configure Stripe prices in a test environment and set rewards for qualifying referrals.", href: "/docs#payments", action: "Configure test payments" },
        { title: "Add the service you want to sell", description: "Build the tool or service your customers need. Adapt the notes and credit-based export example, then replace website, pricing, and help content with your own.", href: "/docs#features", action: "Explore the included features" },
        { title: "Verify the journey and deploy", description: "Check registration, purchases, access, and referrals. Complete your privacy and support pages. Switch Stripe keys, prices, and webhooks together to live mode when you are ready to charge.", href: "/docs#launch", action: "Check your launch setup" }
      ]
    }
  });

  async function copyTestCard() {
    try {
      await navigator.clipboard.writeText(testCard.replaceAll(" ", ""));
      setCopyState("copied");
    } catch {
      setCopyState("error");
    }
  }

  return (
    <SiteShell
      eyebrow={content.eyebrow}
      title={content.title}
      description={content.description}
      showEnvironment={false}
      sideTitle={t({ en: "What you get", zh: "你会得到什么" })}
      sideBody={<DetailRows rows={[
        { label: t({ en: "Source", zh: "模板源码" }), value: t({ en: "Free · frontend + backend", zh: "免费 · 配套前端与后端" }) },
        { label: t({ en: "Your website", zh: "产品官网" }), value: t({ en: "Overview, pricing, docs, and blog", zh: "介绍、价格、文档和博客" }) },
        { label: t({ en: "Customer area", zh: "用户工作台" }), value: t({ en: "Account, subscription, and referrals", zh: "自己的账户、订阅与邀请" }) },
        { label: t({ en: "Admin area", zh: "管理员后台" }), value: t({ en: "Site-wide records and settings", zh: "全站记录与网站配置" }) }
      ]} />}
      actions={<><CTAButton href="/login" primary>{content.start}</CTAButton><CTAButton href="#setup">{content.configure}</CTAButton></>}
      toc={[
        { id: "what-is-template", label: t({ en: "What is Template?", zh: "Template 是什么" }) },
        { id: "included", label: content.included }, { id: "who-uses-what", label: t({ en: "Customer and admin areas", zh: "用户与管理员怎么用" }) }, { id: "try-it", label: content.tryIt },
        ...(demo ? [{ id: "test-payment", label: content.payment }] : []),
        { id: "sharing", label: content.sharing }, { id: "setup", label: content.setup }
      ]}
    >
      <PageSection id="what-is-template" title={t({ en: "A starting point for the SaaS you want to build.", zh: "把通用功能准备好，让你从自己的业务开始。" })} description={t({ en: "A SaaS is a service people use online, often through a subscription or usage credits. Template gives you its common foundations as source code, ready for you to customize.", zh: "SaaS 就是用户在线使用、按订阅或用量付费的软件服务。Template 把这类产品常见的基础功能整合成源码，供你在自己的项目里使用。" })}>
        <div className={styles.introduction}>
          <article><span className="panel-kicker">{t({ en: "THE STARTING POINT", zh: "已经准备好的" })}</span><h3>{t({ en: "A website with working flows", zh: "官网，加上能实际使用的功能" })}</h3><p>{t({ en: "Customers can register, manage a subscription, and invite others. You get a separate console to operate the website.", zh: "用户可以注册、管理订阅、邀请他人；你有独立的管理员后台来运营网站。当前站点就是这些功能的体验入口。" })}</p></article>
          <article><span className="panel-kicker">{t({ en: "YOUR PRODUCT", zh: "可以基于它做什么" })}</span><h3>{t({ en: "Build the service you want to sell", zh: "做你的会员工具或订阅产品" })}</h3><p>{t({ en: "Use it as the foundation for a paid tool, content service, or AI app. Those product-specific capabilities are yours to develop and connect.", zh: "适合有开发能力的独立开发者和小团队，用来搭建付费工具、内容服务或 AI 应用。具体的工具、内容和 AI 能力由你接入。" })}</p></article>
          <article><span className="panel-kicker">{t({ en: "THE COST", zh: "模板与产品的价格" })}</span><h3>{t({ en: "Free source. Your own pricing.", zh: "模板免费，你的产品由你定价" })}</h3><p>{t({ en: "The pricing page demonstrates how your SaaS could charge customers. Hosting, domains, email, and payment services have their own provider costs.", zh: "本站价格页展示的是 SaaS 套餐示例，获取模板无需购买套餐。部署后，你决定自己的服务卖多少钱；域名、服务器、邮件等费用由服务商单独收取。" })}</p></article>
        </div>
      </PageSection>

      {demo && (
        <aside className={styles.demoNotice} aria-label={t({ en: "Demo payment information", zh: "演示支付说明" })}>
          <span className={styles.modeBadge}>{t({ en: "LIVE DEMO", zh: "在线体验版" })}</span>
          <div>
            <strong>{t({ en: "Try real accounts and referrals, with Stripe test payments.", zh: "真实体验注册与邀请，用 Stripe 测试支付走完购买流程。" })}</strong>
            <p>{t({ en: "Account and referral records are saved by the backend. Payments in a Stripe test environment simulate purchases without moving money. Check that Checkout shows test mode before using the test card. Available features depend on the services enabled for this site.", zh: "账号和邀请关系由后端实际保存。在标有测试模式的 Stripe 收银台中，可以模拟付款而不扣真实资金。使用测试卡前确认收银台模式；具体可体验的功能，以当前站点已启用的服务为准。" })}</p>
          </div>
          <a className="text-link" href="#test-payment">{t({ en: "View test card ↗", zh: "查看测试卡 ↗" })}</a>
        </aside>
      )}

      <PageSection id="included" title={t({ en: "The foundations of your SaaS, already integrated.", zh: "上线 SaaS 需要的基础能力，已经接好。" })} description={t({ en: "From a customer's first sign-in to payments and referrals, reuse the common features and focus development on what makes your product useful.", zh: "从用户第一次登录，到订阅付费和邀请分享，复用已经整合的通用功能，把开发精力放在产品真正提供的价值上。" })}>
        <div className="feature-grid">{content.features.map((feature) => <FeatureCard key={feature.label} {...feature} />)}</div>
        <div className={styles.examples}>
          <div><span className="panel-kicker">{t({ en: "BUSINESS EXAMPLES", zh: "附带可修改的业务示例" })}</span><h3>{t({ en: "See how a paid feature fits together", zh: "看看一个付费功能是怎么运作的" })}</h3><p>{t({ en: "Save a note for free, confirm its export cost, then spend credits to download Markdown. You can also upload and view private images. Adapt these examples to the service you build.", zh: "免费保存笔记，确认导出价格后消耗积分下载 Markdown，再到积分页查看余额与记录。还可体验属于自己账号的图片上传与查看。你可以把这些示例改成自己的业务功能。" })}</p></div>
          <div className={styles.exampleLinks}><Link href="/notes">{t({ en: "Notes and paid exports", zh: "笔记与积分导出" })} ↗</Link><Link href="/credits">{t({ en: "My credits and transactions", zh: "个人积分与流水" })} ↗</Link><Link href="/files">{t({ en: "Private images", zh: "个人图片文件" })} ↗</Link><small>{t({ en: "Sign in first. Uploads require storage to be enabled.", zh: "需先登录；图片上传需站点已启用存储。" })}</small></div>
        </div>
      </PageSection>

      <PageSection id="who-uses-what" title={t({ en: "Your customers have a workspace. You have an admin console.", zh: "用户管理自己的账户，你管理整个网站。" })} description={t({ en: "The template provides a public website and two separate areas with different permissions.", zh: "模板包含公开官网、用户工作台和管理员后台。三者用途不同，登录后能看到什么由账号权限决定。" })}>
        <div className={styles.areaTable}>
          <table><caption className={styles.srOnly}>{t({ en: "Who uses each part of the template", zh: "模板各页面的使用者与用途" })}</caption><thead><tr><th scope="col">{t({ en: "Who", zh: "谁来用" })}</th><th scope="col">{t({ en: "What they do", zh: "能做什么" })}</th><th scope="col">{t({ en: "Where to start", zh: "从哪里进入" })}</th></tr></thead><tbody>
            <tr><th scope="row">{t({ en: "Visitors", zh: "尚未登录的访客" })}</th><td>{t({ en: "Learn about the product, browse plans and guides, then sign up.", zh: "了解产品、查看套餐与使用指南，再决定是否注册。" })}</td><td><Link href="/pricing">{t({ en: "Website and pricing", zh: "官网与套餐示例" })} ↗</Link></td></tr>
            <tr><th scope="row">{t({ en: "Customers", zh: "注册用户" })}</th><td>{t({ en: "Manage their own account, subscription, invoices, and invitations. They cannot change site settings or access other users' records.", zh: "管理自己的账户、订阅、账单和邀请；无法修改网站配置或查看其他用户的数据。" })}</td><td><Link href="/account">{t({ en: "My workspace", zh: "进入用户工作台" })} ↗</Link></td></tr>
            <tr><th scope="row">{t({ en: "Website administrators", zh: "网站管理员" })}</th><td>{t({ en: "Review site-wide records, grant or refund credits, update public settings, and check operational activity.", zh: "查询全站用户与业务记录，发放或退回积分，修改公开网站配置，查看相关操作记录。" })}</td><td><code>/admin</code><br /><Link href="/docs#admin">{t({ en: "Configure admin access", zh: "查看管理员配置方法" })} ↗</Link></td></tr>
          </tbody></table>
        </div>
        <p className={styles.helper}>{t({ en: "Signing up creates a customer account. To operate your own deployment, configure a separate administrator email and password on the backend, then sign in at /admin.", zh: "普通注册只会获得用户账号。部署自己的站点后，在后端环境变量中配置管理员邮箱与密码，再从 /admin 登录后台。" })}</p>
      </PageSection>

      <PageSection
        id="analytics"
        title={t({ en: "See the essentials of your operation", zh: "掌握产品的基本运营情况" })}
        description={t({ en: "An administrator console brings together user, subscription, and invitation counts with the records needed to operate your SaaS.", zh: "管理员后台汇总用户、订阅和邀请计数，配合相关记录查询，支持 SaaS 日常运营。" })}
      >
        <p className={styles.helper}>
          <strong>{t({ en: "Core operations are included. ", zh: "基础运营后台已内置。" })}</strong>
          {t({ en: "Administrators can view real user, subscription, invitation, payment, and credit records, manage public site settings, and review recorded operations. The cards describe the available features; live operational data is visible only in the administrator console.", zh: "管理员可查询真实用户、订阅、邀请、支付和积分记录，调整公开网站配置，并查看操作记录。下方展示能力说明，实际运营数据仅在管理员后台中可见。" })}
        </p>
        <div className="metric-grid">
          {content.metrics.map((metric) => <MetricCard key={metric.label} {...metric} />)}
        </div>
        <p className={styles.helper}>{t({ en: "These are basic operational metrics. DAU (daily active users), MAU, retention, project counts, and revenue analytics require product-specific tracking and are not included yet.", zh: "这类数据专业上称为「基础运营指标」。DAU（日活跃用户数）、MAU（月活跃用户数）、留存、项目数量和收入分析尚未内置，需要根据你的业务另行接入。" })}</p>
      </PageSection>

      <PageSection id="try-it" title={t({ en: "Take the customer journey.", zh: "像真实用户一样，完整体验一次。" })} description={t({ en: "You do not need to deploy anything to try this site. Start with an account, then explore purchases and sharing.", zh: "体验本站不需要先部署或修改配置。先注册一个账号，再依次试用购买、订阅管理和邀请分享。" })}>
        <ol className={styles.journey}>
          {content.steps.map((step, index) => (
            <li className={styles.step} key={step.title}>
              <span className={styles.stepNumber} aria-hidden="true">0{index + 1}</span>
              <h3>{step.title}</h3><p>{step.description}</p><p className={styles.stepResult}>{step.result}</p>
              <Link className="text-link" href={step.href}>{step.action} <span aria-hidden="true">↗</span></Link>
            </li>
          ))}
        </ol>
        <p className={styles.helper}>{t({ en: "If a service is shown as unavailable, try the enabled features first. A site owner needs to configure payment or email providers to open those flows; the isolated local preview deliberately leaves payments disabled.", zh: "如果页面提示某项服务尚未开放，可以先体验已启用的功能。支付、邮件等流程需站点完成相应服务配置；独立的本地预览模式默认关闭支付。" })}</p>
      </PageSection>

      {demo && (
        <PageSection id="test-payment" title={t({ en: "Try a purchase in test mode.", zh: "用测试卡，体验购买流程。" })} description={t({ en: "Use this card in Stripe Checkout when it shows test mode. Use test details only; you do not need your own bank card.", zh: "在标有测试模式的 Stripe 收银台中，使用下面的测试卡完成付款。只填写测试信息，不需要使用自己的银行卡。" })}>
          <div className={styles.paymentGrid}>
            <div className={styles.testCard}>
              <div className={styles.cardHeading}><span>STRIPE / TEST</span><span>VISA</span></div>
              <span className={styles.cardLabel}>{t({ en: "Successful payment", zh: "支付成功测试卡" })}</span>
              <code className={styles.cardNumber}>{testCard}</code>
              <div className={styles.cardFields}>
                <div><span>{t({ en: "EXPIRY", zh: "有效期" })}</span><strong>{t({ en: "Any future date", zh: "任意未来日期" })}</strong></div>
                <div><span>CVC</span><strong>123</strong></div>
              </div>
              <button className="button primary wide" type="button" onClick={() => void copyTestCard()}>{copyState === "copied" ? t({ en: "Copied", zh: "已复制卡号" }) : t({ en: "Copy test card number", zh: "复制测试卡号" })}</button>
              <p className={styles.copyStatus} role="status">{copyState === "error" ? t({ en: "Could not copy. Select and copy the card number above.", zh: "复制失败，请选中上方卡号手动复制。" }) : copyState === "copied" ? t({ en: "Paste this number into Stripe test Checkout.", zh: "可以粘贴到 Stripe 测试收银台了。" }) : t({ en: "Test card only · no real funds", zh: "仅用于测试 · 不涉及真实资金" })}</p>
            </div>
            <div className={styles.paymentGuide}>
              <h3>{t({ en: "What to enter, and what happens next", zh: "怎么填，付完会发生什么" })}</h3>
              <DetailRows rows={[
                { label: t({ en: "Expiry / CVC", zh: "有效期 / 安全码" }), value: t({ en: "Any future month/year and any 3 digits, e.g. 123.", zh: "有效期选未来的月份和年份；安全码填任意 3 位数字，如 123。" }) },
                { label: t({ en: "Other details", zh: "其他信息" }), value: t({ en: "Use test name and address details if requested.", zh: "若收银台要求姓名、地址等信息，填写符合格式的测试内容即可。" }) },
                { label: t({ en: "After checkout", zh: "购买完成后" }), value: t({ en: "Return to billing. Allow the payment result to sync, then refresh your subscription and invoices.", zh: "返回订阅管理，等待支付结果同步，再刷新查看套餐和账单。" }) },
                { label: t({ en: "Try a decline", zh: "测试支付失败" }), value: <><code>4000 0000 0000 0002</code><br />{t({ en: "Simulates a declined card.", zh: "这张测试卡会模拟拒付。" })}</> }
              ]} />
              <p className={styles.helper}>{t({ en: "Your plan updates after the payment result arrives. If it has not updated yet, refresh shortly or contact the site owner.", zh: "套餐会在支付结果同步后更新。若暂时没有变化，可以稍后刷新或联系站点负责人。" })}</p>
              <div className="button-row"><CTAButton href="/billing" primary>{t({ en: "Try a purchase", zh: "去测试购买" })}</CTAButton><a className="text-link" href="https://docs.stripe.com/testing" target="_blank" rel="noreferrer">{t({ en: "Stripe test card guide ↗", zh: "Stripe 官方测试说明 ↗" })}</a></div>
            </div>
          </div>
        </PageSection>
      )}

      <PageSection id="sharing" title={t({ en: "Test the invitation journey with two accounts.", zh: "用两个账号，走完邀请流程。" })} description={t({ en: "The inviter does not need to purchase first. Use two accounts to follow sharing, sign-up, and activation; opening a link alone does not create a referral.", zh: "邀请人无需先购买。用两个账号，验证从分享链接到注册、激活和奖励的完整流程；仅打开链接不会产生注册记录。" })}>
        <div className={styles.referralFlow}>
          <article><span className="panel-kicker">{t({ en: "YOU / ACCOUNT A", zh: "你 / 账号 A" })}</span><h3>{t({ en: "Copy your invite link", zh: "复制你的专属邀请链接" })}</h3><p>{t({ en: "Sign in and open the referral center. Copy your link and share it, or open it yourself in a separate private window.", zh: "登录后进入推荐中心，复制链接。可以分享给其他人，也可以自己在独立的无痕窗口中打开。" })}</p><Link className="text-link" href="/referrals">{t({ en: "Get my invite link ↗", zh: "获取我的邀请链接 ↗" })}</Link></article>
          <article><span className="panel-kicker">{t({ en: "NEW CUSTOMER / ACCOUNT B", zh: "新用户 / 账号 B" })}</span><h3>{t({ en: "Register through that link", zh: "通过链接注册新账号" })}</h3><p>{demo ? t({ en: "Use a different email or social account that has never registered here. Complete sign-up, then make a subscription purchase using the test card above.", zh: "使用从未在本站注册过的另一邮箱或第三方账号完成注册。随后使用上面的测试卡，体验一次订阅购买。" }) : t({ en: "Use a different email or social account that has never registered here. After sign-up, the new customer can choose a subscription.", zh: "使用从未在本站注册过的另一邮箱或第三方账号完成注册。注册后，新用户可以选择订阅套餐。" })}</p></article>
          <article><span className="panel-kicker">{t({ en: "BACK TO ACCOUNT A", zh: "回到账号 A" })}</span><h3>{t({ en: "See the relationship and reward", zh: "查看邀请关系与奖励" })}</h3><p>{t({ en: "Reopen or refresh the referral center. Sign-up creates a pending record; the first qualifying subscription or lifetime payment activates it and grants the configured credit reward.", zh: "重新进入或刷新推荐中心。注册后先出现待激活记录；新用户首次符合条件的订阅或终身套餐付款后，邀请状态更新，并按配置发放积分奖励。" })}</p></article>
        </div>
        <p className={styles.helper}>{t({ en: "Free trials earn a reward after the first qualifying payment. Buying a credit package alone does not activate a referral. Rewards are in-product credits. Registering through a link must happen before activation and within the site's invitation rules.", zh: "免费试用需在首次符合条件的付款后才发放奖励；单独购买积分包不触发邀请奖励。奖励是产品内积分。请先通过链接完成新账号注册，再按站点公布的期限与规则完成激活。" })}</p>
      </PageSection>

      <PageSection id="setup" title={t({ en: "From a free template to your own SaaS.", zh: "从免费模板，到你自己的 SaaS。" })} description={t({ en: "For developers: use the included Next.js frontend and Go backend. Configure your brand, database, authentication, Resend email, Stripe billing, and referrals, then add your core service and deploy.", zh: "给开发者：使用配套的 Next.js 前端和 Go 后端，配置品牌、数据库、登录方式、Resend 邮件、Stripe 支付和邀请规则，接上核心业务后部署上线。" })}>
        <div className="button-row"><a className="button primary" href="https://github.com/brizenchi/go-modules" target="_blank" rel="noreferrer">{t({ en: "Get the template source", zh: "获取模板源码" })} ↗</a><CTAButton href="/docs">{t({ en: "Read the step-by-step guide", zh: "打开完整使用指南" })}</CTAButton></div>
        <ol className={styles.launchSteps}>
          {content.configuration.map((item, index) => (
            <li key={item.title}><span className={styles.stepNumber} aria-hidden="true">0{index + 1}</span><div><h3>{item.title}</h3><p>{item.description}</p><Link className="text-link" href={item.href}>{item.action} ↗</Link></div></li>
          ))}
        </ol>
        <p className={styles.setupNote}>{t({ en: "The guide contains the environment variables and setup checks for each step. Keep Stripe, Resend, database, and administrator credentials on the backend. The frontend demo flag changes copy only; it does not switch payment mode.", zh: "每一步的环境变量和检查方法都在使用指南中。Stripe、Resend、数据库与管理员密码只放后端配置；前端的演示开关只改变提示文案，不会切换支付模式。" })}</p>
        <p className={styles.setupNote}>{t({ en: "The template is free. Hosting, domains, and email services are billed separately by your chosen providers. If you purchase through a link marked as a referral, the author may earn a commission to support maintenance. You can choose other providers.", zh: "模板免费提供。部署、域名、邮件等第三方服务，由你选择的平台单独计费。如果你通过标注为推荐的链接购买服务，作者可能获得佣金，用于支持模板维护；你也可以选择其他服务商。" })}</p>
        <div className="cta-strip"><div><span className="panel-kicker">{t({ en: "FREE TEMPLATE. YOUR NEXT SAAS.", zh: "免费模板，开启你的下一个 SaaS。" })}</span><strong>{t({ en: "Spend your next sprint on your core product.", zh: "把下一轮开发，留给你的核心业务。" })}</strong></div><div className="cta-strip-actions"><CTAButton href="/login" primary>{content.start}</CTAButton><CTAButton href="/pricing">{t({ en: "Explore example plans", zh: "查看套餐示例" })}</CTAButton></div></div>
      </PageSection>
    </SiteShell>
  );
}
