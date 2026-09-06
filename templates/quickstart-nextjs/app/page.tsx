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
      title: "快速上线你的 SaaS。",
      description: "Template 是一套面向独立开发者和小团队的免费 SaaS 启动模板。已整合用户登录、订阅支付、积分计费和邀请推荐，配套 Resend 邮件服务、产品官网与配置指南。复用这些基础能力，把时间留给你的核心业务，让想法更快成为可以运营的产品。",
      start: "体验在线演示", configure: "查看部署指南",
      included: "已整合的功能", analytics: "产品分析与运营看板", tryIt: "怎么体验", payment: "怎么测试支付", sharing: "怎么测试分享", setup: "怎么配置上线",
      features: [
        { label: "01 / 用户认证", title: "注册登录，接好就能用", description: "整合邮箱验证码、Google 和 GitHub 登录，支持账号创建、会话管理与退出登录。按需启用登录方式，让用户顺畅进入你的产品。" },
        { label: "02 / 订阅与计费", title: "为你的产品接上付费能力", description: "整合 Stripe Checkout，支持月付、年付、终身套餐与积分包，配套订阅管理和账单查询。根据业务选择合适的收费方式。" },
        { label: "03 / 邀请推荐", title: "从用户分享，到邀请转化", description: "专属邀请链接、注册归因、订阅激活与积分奖励已经串联。用户可以查看邀请记录和统计，为产品的推荐增长提供基础。" },
        { label: "04 / Resend 邮件", title: "把验证和欢迎邮件送到用户手中", description: "已接入 Resend 邮件发送能力。配置发信域名和密钥后即可发送验证码，还可以按需启用欢迎邮件，完善用户的首次体验。" },
        { label: "05 / 产品官网", title: "展示、定价、文档一起准备好", description: "配套首页、价格页、文档页和账户入口，使用统一的响应式界面。替换品牌与文案，沿用现有中英文结构，快速搭建产品门面。" },
        { label: "06 / 开发与部署", title: "在完整前后端上开发你的业务", description: "Next.js 前端与 Go 后端配套提供，包含环境配置示例和上线指南。按需启用模块、连接数据库、配置服务，再接入你的业务功能。" }
      ],
      metrics: [
        { label: "用户增长", value: "新增与累计用户", detail: "按时间观察注册增长，了解有多少人开始使用你的产品。" },
        { label: "活跃与留存", value: "DAU / MAU", detail: "按统一活跃行为统计每日、每月的去重用户，并观察用户是否持续回来使用。" },
        { label: "业务使用", value: "项目与关键操作", detail: "统计项目总数、新建项目数和关键功能使用量，了解用户真正完成了哪些工作。" },
        { label: "转化效果", value: "注册到订阅", detail: "分析注册、付费与邀请转化，找到用户流失的环节和可以改善的体验。" }
      ],
      steps: [
        { title: "注册并登录", description: "点击登录，使用页面提供的邮箱或第三方登录方式。新用户首次完成验证时会自动创建账号。", result: "完成后：账户页可以看到自己的账号。", href: "/login", action: "注册 / 登录" },
        { title: "体验订阅与计费", description: "价格页展示模板支持的 SaaS 收费方式，不是模板售价。进入订阅管理选择套餐与周期，或体验一次性积分包。", result: "完成后：进入 Stripe 收银台。", href: "/pricing", action: "查看套餐示例" },
        { title: demo ? "完成测试购买" : "购买并管理订阅", description: demo ? "在 Stripe 测试收银台填写下方测试卡。订阅、买断和积分包的付款都是模拟的，不会扣除真实资金。" : "在 Stripe 收银台确认订单与金额。返回后可查看账单，或管理套餐、取消与恢复订阅。", result: "完成后：返回订阅管理，等待支付结果同步后刷新查看。", href: demo ? "#test-payment" : "/billing", action: demo ? "获取测试卡" : "订阅管理" },
        { title: "分享给一个新用户", description: "到推荐中心复制自己的邀请链接，用另一个浏览器或无痕窗口打开，并使用尚未注册的账号完成注册。", result: "完成后：回到原账号查看邀请记录。", href: "/referrals", action: "打开推荐中心" }
      ],
      configuration: [
        { title: "品牌与访问地址", description: "把前端配置复制为 .env.local，填写产品名、网站地址和后端 API 地址，再替换首页、价格和文档中的产品内容。", file: "templates/quickstart-nextjs/.env.local", code: "NEXT_PUBLIC_APP_NAME=My SaaS\nNEXT_PUBLIC_APP_URL=https://app.example.com\nNEXT_PUBLIC_API_BASE_URL=https://api.example.com/api/v1", note: "前端从 .env.example 起步；部署时参考 .env.production.example。NEXT_PUBLIC_ 配置会进入浏览器，修改后需要重新构建。" },
        { title: "数据库与真实登录", description: "配置 PostgreSQL 和独立的认证密钥，选择邮箱、Google 或 GitHub 登录。沿用本项目的 Resend 方案配置发信域名和密钥；第三方登录填写相应平台的凭据和回调。", file: "templates/quickstart/.env + deploy/config.yaml", code: "APP_DB_HOST=your-database-host\nAPP_DB_NAME=my_saas\nAPP_AUTH_FRONTEND_REDIRECT=https://app.example.com/login\nAPP_HTTP_ALLOWED_ORIGINS=https://app.example.com\nAPP_AUTH_EMAIL_DEBUG=false", note: "先复制后端 .env.example 与 deploy/config.yaml.example，再补齐数据库、JWT 和服务商配置。本地默认邮件写日志；要实际收邮件，需要配置邮件服务。前端登录开关必须与后端一致。" },
        { title: "套餐与 Stripe 测试支付", description: "在 Stripe 测试环境创建订阅和积分包价格，把对应的 Price ID 填入后端。启用支付，配置测试密钥、Webhook，并在 Stripe 中启用 Customer Portal。", file: "templates/quickstart/.env", code: "APP_BILLING_ENABLED=true\nAPP_BILLING_STRIPE_SECRET_KEY=sk_test_...\nAPP_BILLING_STRIPE_WEBHOOK_SECRET=whsec_...\nAPP_BILLING_STRIPE_PRICES_PRO_MONTHLY=price_...\nAPP_BILLING_STRIPE_PRICES_CREDITS=price_...", note: "Webhook 地址为 https://api.example.com/api/v1/stripe/webhook。密钥、价格和 Webhook 必须属于同一测试环境；价格页金额也要与 Stripe 保持一致。密钥只放后端。" },
        { title: "邀请链接与奖励规则", description: "把邀请链接换成你的前端域名，设置被邀请人首次订阅激活后的奖励积分。注册归因和奖励记录会真实保存。", file: "templates/quickstart/.env", code: "APP_REFERRAL_ENABLED=true\nAPP_REFERRAL_BASE_LINK=https://app.example.com/invite?ref=\nAPP_REFERRAL_ACTIVATION_REWARD=50", note: "仅注册时，邀请关系处于待激活状态；被邀请人首次订阅激活后，才会按配置发放奖励。积分是产品内权益，不是现金返利。" },
        { title: "部署并开始正式销售", description: "接入自己的核心业务，完成注册、购买和邀请验收。部署前后端与 HTTPS 域名，再将 Stripe 密钥、价格和 Webhook 整套切换到正式环境。", file: ".env.production.example → 后端 .env / 前端 .env.local", code: "# 后端\nAPP_ENV=production\nAPP_AUTH_EMAIL_DEBUG=false\nAPP_BILLING_STRIPE_SECRET_KEY=sk_live_...\n\n# 前端\nNEXT_PUBLIC_DEMO_MODE=false", note: "NEXT_PUBLIC_DEMO_MODE 只控制总览中的演示说明和测试卡展示，不会切换 Stripe 或阻止扣款。正式销售前同步切换支付配置、替换演示内容，并重新构建前端。" }
      ]
    },
    en: {
      eyebrow: "Free SaaS starter · Next.js + Go",
      title: "Get your SaaS to launch, faster.",
      description: "Template is a free SaaS starter for independent developers and small teams. Authentication, subscriptions, credit billing, and referrals are already integrated, alongside Resend email, a public website, and setup guides. Build on these foundations and spend your time on the service your customers need.",
      start: "Try the live demo", configure: "View the setup guide",
      included: "Integrated features", analytics: "Product analytics", tryIt: "Try the template", payment: "Test a payment", sharing: "Test a referral", setup: "Configure and launch",
      features: [
        { label: "01 / Authentication", title: "Give customers a smooth way in", description: "Email verification, Google, and GitHub sign-in connect account creation, session management, and sign-out. Enable the sign-in options your product needs." },
        { label: "02 / Subscriptions and billing", title: "Start with payments already connected", description: "Stripe Checkout supports monthly and yearly subscriptions, lifetime access, and credit packages, with subscription management and invoices. Choose how your product earns revenue." },
        { label: "03 / Referrals", title: "Connect sharing to customer growth", description: "Personal invite links connect sign-up attribution, subscription activation, and credit rewards. Customers can review referral history and statistics." },
        { label: "04 / Resend email", title: "Reach customers from their first sign-in", description: "Resend email delivery is integrated. Configure your sending domain and API key to deliver verification codes, then enable welcome emails when you need them." },
        { label: "05 / Your public website", title: "Present, explain, and price your product", description: "A homepage, pricing, documentation, and account entry points share a responsive interface. Update your brand and copy, with an existing English / Chinese structure to build on." },
        { label: "06 / Development and deployment", title: "Build your service on a complete stack", description: "A Next.js frontend and Go backend come with configuration examples and a launch guide. Enable modules, connect your database and providers, then add your core features." }
      ],
      metrics: [
        { label: "User growth", value: "New and total users", detail: "Track sign-ups over time to understand how many people are starting to use your product." },
        { label: "Activity and retention", value: "DAU / MAU", detail: "Count distinct daily and monthly users using a consistent definition of activity, and see whether they return." },
        { label: "Product usage", value: "Projects and actions", detail: "Track total projects, new projects, and key feature usage to understand what customers actually accomplish." },
        { label: "Conversion", value: "Sign-up to subscription", detail: "Explore registration, payment, and referral conversion to find where customers leave and what to improve." }
      ],
      steps: [
        { title: "Create your account", description: "Choose an available email or social sign-in option. Completing verification for the first time automatically creates your account.", result: "Then: find your identity on the account page.", href: "/login", action: "Sign up / sign in" },
        { title: "Explore subscriptions and billing", description: "Pricing demonstrates ways to charge for your own SaaS; the template itself is free. Choose a plan and interval in billing, or try a one-time credit package.", result: "Then: continue to Stripe Checkout.", href: "/pricing", action: "Explore example plans" },
        { title: demo ? "Make a test purchase" : "Buy and manage your subscription", description: demo ? "Enter the test card below in Stripe test Checkout. Subscription, lifetime, and credit package payments are simulated and move no real money." : "Confirm the order and amount in Stripe Checkout. Return to view invoices, change plans, or cancel and resume a subscription.", result: "Then: return to billing and refresh after the payment result syncs.", href: demo ? "#test-payment" : "/billing", action: demo ? "Get the test card" : "Manage billing" },
        { title: "Invite a new customer", description: "Copy your invite link from the referral center. Open it in another browser or private window and register with an account that has never signed up here.", result: "Then: return to your original account to see the referral.", href: "/referrals", action: "Open referrals" }
      ],
      configuration: [
        { title: "Your brand and domains", description: "Copy the frontend configuration to .env.local. Set your product name, public URL, and backend API URL, then update the homepage, pricing, and documentation.", file: "templates/quickstart-nextjs/.env.local", code: "NEXT_PUBLIC_APP_NAME=My SaaS\nNEXT_PUBLIC_APP_URL=https://app.example.com\nNEXT_PUBLIC_API_BASE_URL=https://api.example.com/api/v1", note: "Start with .env.example locally or .env.production.example for deployment. NEXT_PUBLIC_ values are visible in the browser and require a new build when changed." },
        { title: "Database and real sign-in", description: "Configure PostgreSQL and separate authentication secrets. Choose email, Google, or GitHub sign-in. Follow this project’s Resend setup for your sending domain and API key; configure provider credentials and callbacks for social sign-in.", file: "templates/quickstart/.env + deploy/config.yaml", code: "APP_DB_HOST=your-database-host\nAPP_DB_NAME=my_saas\nAPP_AUTH_FRONTEND_REDIRECT=https://app.example.com/login\nAPP_HTTP_ALLOWED_ORIGINS=https://app.example.com\nAPP_AUTH_EMAIL_DEBUG=false", note: "Copy backend .env.example and deploy/config.yaml.example first, then complete database, JWT, and provider settings. Local email defaults to logs; delivery needs an email provider. Frontend sign-in options must match the backend." },
        { title: "Plans and Stripe test payments", description: "Create subscription and credit package prices in a Stripe test environment. Add the Price IDs to your backend, enable billing, configure the test key and webhook, and enable Stripe Customer Portal.", file: "templates/quickstart/.env", code: "APP_BILLING_ENABLED=true\nAPP_BILLING_STRIPE_SECRET_KEY=sk_test_...\nAPP_BILLING_STRIPE_WEBHOOK_SECRET=whsec_...\nAPP_BILLING_STRIPE_PRICES_PRO_MONTHLY=price_...\nAPP_BILLING_STRIPE_PRICES_CREDITS=price_...", note: "Use https://api.example.com/api/v1/stripe/webhook. Keys, prices, and webhooks must belong to the same test environment. Keep pricing copy consistent with Stripe, and keep secrets on the backend." },
        { title: "Referral links and rewards", description: "Set your frontend domain as the invite destination and choose the credits awarded when a new referral's first subscription activates. Attribution and rewards are saved to real records.", file: "templates/quickstart/.env", code: "APP_REFERRAL_ENABLED=true\nAPP_REFERRAL_BASE_LINK=https://app.example.com/invite?ref=\nAPP_REFERRAL_ACTIVATION_REWARD=50", note: "Sign-up creates a pending referral. The first subscription activation triggers the configured reward. Credits are product entitlements, not cash payouts." },
        { title: "Deploy and start selling", description: "Add your core service and verify sign-up, purchases, and referrals. Deploy both apps with HTTPS, then switch Stripe keys, prices, and webhooks together to your live environment.", file: ".env.production.example → backend .env / frontend .env.local", code: "# Backend\nAPP_ENV=production\nAPP_AUTH_EMAIL_DEBUG=false\nAPP_BILLING_STRIPE_SECRET_KEY=sk_live_...\n\n# Frontend\nNEXT_PUBLIC_DEMO_MODE=false", note: "NEXT_PUBLIC_DEMO_MODE only controls overview demo copy and test cards. It does not switch Stripe modes or prevent charges. Before selling, switch payment configuration, replace demo content, and rebuild the frontend." }
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
      sideTitle={t({ en: "Try it live", zh: "真实体验" })}
      sideBody={<DetailRows rows={[
        { label: t({ en: "Accounts", zh: "注册 / 登录" }), value: t({ en: "Real accounts", zh: "真实创建账号" }) },
        { label: t({ en: "Referrals", zh: "分享 / 邀请" }), value: t({ en: "Real attribution", zh: "真实记录关系" }) },
        { label: t({ en: "Payments", zh: "购买 / 订阅" }), value: demo ? t({ en: "Test payments · no charges", zh: "测试支付 · 不扣真钱" }) : t({ en: "Stripe Checkout", zh: "Stripe 收银台" }) },
        { label: t({ en: "The template", zh: "模板本身" }), value: t({ en: "Free to use and customize", zh: "免费使用，自由定制" }) }
      ]} />}
      actions={<><CTAButton href="/login" primary>{content.start}</CTAButton><CTAButton href="#setup">{content.configure}</CTAButton></>}
      toc={[
        { id: "included", label: content.included }, { id: "analytics", label: content.analytics }, { id: "try-it", label: content.tryIt },
        ...(demo ? [{ id: "test-payment", label: content.payment }] : []),
        { id: "sharing", label: content.sharing }, { id: "setup", label: content.setup }
      ]}
    >
      {demo && (
        <aside className={styles.demoNotice} aria-label={t({ en: "Demo payment information", zh: "演示支付说明" })}>
          <span className={styles.modeBadge}>{t({ en: "LIVE DEMO", zh: "在线体验版" })}</span>
          <div>
            <strong>{t({ en: "Real accounts and referrals. Only payments are simulated.", zh: "注册、登录、分享都是真的，只有支付是模拟的。" })}</strong>
            <p>{t({ en: "The template is free. Prices on this site demonstrate SaaS billing, and all demo purchases use Stripe test payments with no real charges. Accounts and referral records are actually saved.", zh: "模板免费提供。本站价格用于演示 SaaS 的收费方式，所有购买均使用 Stripe 测试支付，不扣除真实资金。账号和邀请关系会真实保存。" })}</p>
          </div>
          <a className="text-link" href="#test-payment">{t({ en: "View test card ↗", zh: "查看测试卡 ↗" })}</a>
        </aside>
      )}

      <PageSection id="included" title={t({ en: "The foundations of your SaaS, already integrated.", zh: "上线 SaaS 需要的基础能力，已经接好。" })} description={t({ en: "From a customer's first sign-in to payments and referrals, reuse the common features and focus development on what makes your product useful.", zh: "从用户第一次登录，到订阅付费和邀请分享，复用已经整合的通用功能，把开发精力放在产品真正提供的价值上。" })}>
        <div className="feature-grid">{content.features.map((feature) => <FeatureCard key={feature.label} {...feature} />)}</div>
      </PageSection>

      <PageSection
        id="analytics"
        title={t({ en: "Product analytics and business metrics", zh: "产品分析与运营看板" })}
        description={t({ en: "An extension for your product: bring user growth, activity, project usage, and conversion into one dashboard to understand how your SaaS performs after launch.", zh: "可按业务扩展：把用户增长、活跃度、项目使用和转化表现放到同一张数据看板里，帮助你了解 SaaS 上线后的使用情况。" })}
      >
        <p className={styles.helper}>
          <strong>{t({ en: "Extension · not included yet. ", zh: "可扩展能力 · 尚未内置。" })}</strong>
          {t({ en: "The template currently provides individual subscription and invoice views plus referral statistics. DAU, retention, project counts, and an operator dashboard require activity tracking and your business data. The items below describe metrics to add, not live figures.", zh: "当前模板已提供个人订阅、账单查询和邀请统计。DAU、留存、项目数量及运营端看板需要接入用户行为与业务数据；下方介绍的是可扩展的指标，不是演示站的实时统计。" })}
        </p>
        <div className="metric-grid">
          {content.metrics.map((metric) => <MetricCard key={metric.label} {...metric} />)}
        </div>
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
      </PageSection>

      {demo && (
        <PageSection id="test-payment" title={t({ en: "Try checkout. Keep your money.", zh: "走完购买流程，不花一分钱。" })} description={t({ en: "Use this card in Stripe Checkout when it shows test mode. Use test details only; you do not need your own bank card.", zh: "在标有测试模式的 Stripe 收银台中，使用下面的测试卡完成付款。只填写测试信息，不需要使用自己的银行卡。" })}>
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
              <p className={styles.helper}>{t({ en: "Returning from checkout does not by itself activate a plan. If it has not updated, refresh later; the site owner can check the Stripe webhook configuration.", zh: "跳转回来不代表套餐已经生效。如果状态尚未更新，可以稍后刷新；部署者需要检查 Stripe Webhook 是否配置并送达。" })}</p>
              <div className="button-row"><CTAButton href="/billing" primary>{t({ en: "Try a purchase", zh: "去测试购买" })}</CTAButton><a className="text-link" href="https://docs.stripe.com/testing" target="_blank" rel="noreferrer">{t({ en: "Stripe test card guide ↗", zh: "Stripe 官方测试说明 ↗" })}</a></div>
            </div>
          </div>
        </PageSection>
      )}

      <PageSection id="sharing" title={t({ en: "Your next test: invite someone new.", zh: "购买之后，再试试分享和注册。" })} description={t({ en: "Use two different accounts to see the full referral journey. Opening the link alone does not create a referral.", zh: "用两个不同账号，可以亲自验证从分享链接到注册、激活和奖励的完整流程。仅打开链接不会产生注册记录。" })}>
        <div className={styles.referralFlow}>
          <article><span className="panel-kicker">{t({ en: "YOU / ACCOUNT A", zh: "你 / 账号 A" })}</span><h3>{t({ en: "Copy your invite link", zh: "复制你的专属邀请链接" })}</h3><p>{t({ en: "Sign in and open the referral center. Copy your link and share it, or open it yourself in a separate private window.", zh: "登录后进入推荐中心，复制链接。可以分享给其他人，也可以自己在独立的无痕窗口中打开。" })}</p><Link className="text-link" href="/referrals">{t({ en: "Get my invite link ↗", zh: "获取我的邀请链接 ↗" })}</Link></article>
          <article><span className="panel-kicker">{t({ en: "NEW CUSTOMER / ACCOUNT B", zh: "新用户 / 账号 B" })}</span><h3>{t({ en: "Register through that link", zh: "通过链接注册新账号" })}</h3><p>{demo ? t({ en: "Use a different email or social account that has never registered here. Complete sign-up, then make a subscription purchase using the test card above.", zh: "使用从未在本站注册过的另一邮箱或第三方账号完成注册。随后使用上面的测试卡，体验一次订阅购买。" }) : t({ en: "Use a different email or social account that has never registered here. After sign-up, the new customer can choose a subscription.", zh: "使用从未在本站注册过的另一邮箱或第三方账号完成注册。注册后，新用户可以选择订阅套餐。" })}</p></article>
          <article><span className="panel-kicker">{t({ en: "BACK TO ACCOUNT A", zh: "回到账号 A" })}</span><h3>{t({ en: "See the relationship and reward", zh: "查看邀请关系与奖励" })}</h3><p>{t({ en: "Reopen or refresh the referral center. Sign-up creates a pending record; the first subscription activation updates it and grants the configured credit reward.", zh: "重新进入或刷新推荐中心。注册后先出现待激活记录；新用户首次订阅激活后，邀请状态更新，并按配置发放积分奖励。" })}</p></article>
        </div>
      </PageSection>

      <PageSection id="setup" title={t({ en: "From a free template to your own SaaS.", zh: "从免费模板，到你自己的 SaaS。" })} description={t({ en: "For developers: use the included Next.js frontend and Go backend. Configure your brand, database, authentication, Resend email, Stripe billing, and referrals, then add your core service and deploy.", zh: "给开发者：使用配套的 Next.js 前端和 Go 后端，配置品牌、数据库、登录方式、Resend 邮件、Stripe 支付和邀请规则，接上核心业务后部署上线。" })}>
        <div className={styles.configuration}>
          {content.configuration.map((item, index) => (
            <details className={styles.configItem} key={item.title}>
              <summary><span className={styles.configNumber}>0{index + 1}</span><span>{item.title}</span><span className={styles.expandIcon} aria-hidden="true">+</span></summary>
              <div className={styles.configBody}><p>{item.description}</p><span className={styles.fileLabel}>{item.file}</span><pre><code>{item.code}</code></pre><p className={styles.helper}>{item.note}</p></div>
            </details>
          ))}
        </div>
        <p className={styles.setupNote}>{t({ en: "These are the key settings, not complete configuration files. Full startup instructions are included in both template READMEs and docs/SETUP_ZH.md. Disabled features need their backend configuration completed before you can try them.", zh: "以上是关键配置示例，不是完整配置文件。启动步骤见两个模板目录中的 README.md，完整配置指南见 docs/SETUP_ZH.md。未启用的功能需要先补齐后端配置，才能正常体验。" })}</p>
        <p className={styles.setupNote}>{t({ en: "The template is free. Hosting, domains, and email services are billed separately by your chosen providers. If you purchase through a link marked as a referral, the author may earn a commission to support maintenance. You can choose other providers.", zh: "模板免费提供。部署、域名、邮件等第三方服务，由你选择的平台单独计费。如果你通过标注为推荐的链接购买服务，作者可能获得佣金，用于支持模板维护；你也可以选择其他服务商。" })}</p>
        <div className="cta-strip"><div><span className="panel-kicker">{t({ en: "FREE TEMPLATE. YOUR NEXT SAAS.", zh: "免费模板，开启你的下一个 SaaS。" })}</span><strong>{t({ en: "Spend your next sprint on your core product.", zh: "把下一轮开发，留给你的核心业务。" })}</strong></div><div className="cta-strip-actions"><CTAButton href="/login" primary>{content.start}</CTAButton><CTAButton href="/pricing">{t({ en: "Explore example plans", zh: "查看套餐示例" })}</CTAButton></div></div>
      </PageSection>
    </SiteShell>
  );
}
