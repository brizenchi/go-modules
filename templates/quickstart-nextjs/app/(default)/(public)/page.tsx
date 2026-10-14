"use client";

import { LocalizedLink as Link } from "@/components/localized-link";
import { SiteShell } from "@/components/site-shell";
import { CTAButton, PageSection } from "@/components/ui";
import { documentation } from "@/content/docs";
import { appEnv } from "@/lib/env";
import { useI18n } from "@/lib/i18n";
import styles from "./overview.module.css";

export default function HomePage() {
  const { t } = useI18n();
  const demo = appEnv.demoMode;
  const content = t({
    zh: {
      eyebrow: "免费 SaaS 启动模板 · Next.js + Go",
      title: "通用功能已备好。开始做你的 SaaS。",
      description: "登录、订阅支付、积分与邀请，配好官网和管理后台。用一套免费的前后端源码，把开发时间留给自己的产品。",
      start: "体验用户工作台",
      configure: "获取模板与指南",
      features: [
        { label: "01", title: "用户认证", description: "邮箱验证码、Google 与 GitHub 登录，连接账号创建、会话与退出流程。", detail: "邮箱 · Google · GitHub" },
        { label: "02", title: "订阅与计费", description: "Stripe Checkout 配套订阅管理和账单，支持周期套餐、终身访问与积分包。", detail: "月付 / 年付 · 一次性支付" },
        { label: "03", title: "邀请推荐", description: "专属邀请链接串联注册归因与订阅激活，奖励规则可按业务扩展。", detail: "邀请链接 · 转化记录" },
        { label: "04", title: "邮件通知", description: "已整合 Resend，配置发信域名后即可发送验证码，按需启用欢迎邮件。", detail: "验证码 · 欢迎邮件" },
        { label: "05", title: "完整产品官网", description: "首页、套餐、可搜索的文档和博客，配套更新日志、联系页与中英文切换。", detail: "响应式布局 · 双语内容" },
        { label: "06", title: "独立管理后台", description: "查询用户、订阅和支付记录，调整积分、网站信息及服务配置。", detail: "全站记录 · 运营操作" }
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
        { title: "笔记", href: "/notes" },
        { title: "文件", href: "/files" },
        { title: "邀请推荐", href: "/referrals" }
      ],
      flow: [
        { title: "写下并保存笔记", description: "先体验免费的基础操作。", label: "保存笔记" },
        { title: "确认导出所需积分", description: "付费操作前明确展示消耗。", label: "确认费用" },
        { title: "下载 Markdown", description: "完成导出，在积分页查看流水。", label: "查看结果" }
      ]
    },
    en: {
      eyebrow: "Free SaaS starter · Next.js + Go",
      title: "The foundations are ready. Build your SaaS.",
      description: "Sign-in, subscriptions, credits, and referrals, with a website and admin console. Start with free frontend and backend source, and spend your time on your product.",
      start: "Try the customer workspace",
      configure: "Get the template & guides",
      features: [
        { label: "01", title: "Authentication", description: "Email verification, Google, and GitHub sign-in connect account creation, sessions, and sign-out.", detail: "Email · Google · GitHub" },
        { label: "02", title: "Subscriptions & billing", description: "Stripe Checkout connects recurring plans, lifetime access, and credit packages to billing and invoices.", detail: "Monthly / yearly · One-time payments" },
        { label: "03", title: "Referrals", description: "Personal invite links connect sign-up attribution and subscription activation, with room for your reward rules.", detail: "Invite links · Conversion records" },
        { label: "04", title: "Email delivery", description: "Configure your sending domain to deliver verification codes with Resend, and enable welcome emails as needed.", detail: "Verification · Welcome emails" },
        { label: "05", title: "Your public website", description: "A homepage, pricing, searchable docs and blog, release notes, and contact pages with English and Chinese content.", detail: "Responsive layouts · Two languages" },
        { label: "06", title: "Administration", description: "Review users, subscriptions, and payments. Adjust credits and manage site details and service configuration.", detail: "Site-wide records · Operations" }
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
        { title: "Notes", href: "/notes" },
        { title: "Files", href: "/files" },
        { title: "Referrals", href: "/referrals" }
      ],
      flow: [
        { title: "Write and save a note", description: "Start with the free core action.", label: "Save a note" },
        { title: "Review the credit cost", description: "See the cost before the paid action.", label: "Confirm the cost" },
        { title: "Download Markdown", description: "Export your note, then view the transaction.", label: "View the result" }
      ]
    }
  });

  return (
    <SiteShell
      eyebrow={content.eyebrow}
      title={content.title}
      description={content.description}
      showEnvironment={false}
      sideTitle={t({ en: "One product. Three connected spaces.", zh: "一个产品，三个清楚的入口。" })}
      sideBody={
        <div className={styles.entryPoints} id="who-uses-what">
          {[
            { href: "/pricing", label: t({ en: "Public website", zh: "产品官网" }), detail: t({ en: "Discover the product and its plans", zh: "了解产品与套餐" }) },
            { href: "/account", label: t({ en: "Customer workspace", zh: "用户工作台" }), detail: t({ en: "Use the product and manage your account", zh: "使用功能，管理自己的账户" }) },
            { href: "/docs/admin", label: t({ en: "Admin console", zh: "管理员后台" }), detail: t({ en: "Operate your site with separate admin access", zh: "独立管理员权限，运营全站" }) }
          ].map((entry, index) => (
            <Link href={entry.href} key={entry.href}>
              <span className={styles.entryNumber}>0{index + 1}</span>
              <span><strong>{entry.label}</strong><small>{entry.detail}</small></span>
              <span aria-hidden="true">↗</span>
            </Link>
          ))}
          <p className={styles.sourceNote}>{t({ en: "Free source code. Configure services and add your own business.", zh: "源码免费，配置服务后接入你自己的业务。" })}</p>
        </div>
      }
      actions={<><CTAButton href="/account" primary>{content.start}</CTAButton><CTAButton href="#setup">{content.configure}</CTAButton></>}
      toc={[
        { id: "what-is-template", label: t({ en: "Explore the workflow", zh: "看看使用流程" }) },
        { id: "included", label: t({ en: "Included capabilities", zh: "已整合的能力" }) },
        { id: "try-it", label: t({ en: "Try it yourself", zh: "亲自体验" }) },
        { id: "setup", label: t({ en: "Build your product", zh: "开始构建" }) }
      ]}
    >
      <PageSection
        id="what-is-template"
        title={t({ en: "A working flow, from the first action to a paid feature.", zh: "从第一次使用，到完成一次付费操作。" })}
        description={t({ en: "Explore the included notes example. Save for free, review the export cost, and use credits to download Markdown.", zh: "用内置笔记示例，看看免费保存、积分导出和流水查询如何连接。" })}
      >
        <div className={styles.productPreview}>
          <div className={styles.previewBar}>
            <span className={styles.previewLabel}><span aria-hidden="true" />{t({ en: "Customer workspace", zh: "用户工作台" })}</span>
            <span>{t({ en: "Workflow preview", zh: "功能流程预览" })}</span>
          </div>
          <div className={styles.previewBody}>
            <nav className={styles.previewNav} aria-label={t({ en: "Explore workspace features", zh: "浏览工作台功能" })}>
              {content.routes.map((route) => <Link key={route.href} href={route.href} className={route.href === "/notes" ? styles.previewSelected : undefined}><span>{route.title}</span><span aria-hidden="true">↗</span></Link>)}
            </nav>
            <div className={styles.previewContent}>
              <div className={styles.previewHeading}>
                <div><span className={styles.kicker}>{t({ en: "INCLUDED EXAMPLE", zh: "附带业务示例" })}</span><h3>{t({ en: "Notes & credit exports", zh: "笔记与积分导出" })}</h3></div>
                <Link className="button" href="/examples/notes-to-markdown">{t({ en: "See the example", zh: "查看完整示例" })}<span aria-hidden="true">↗</span></Link>
              </div>
              <ol className={styles.previewFlow}>
                {content.flow.map((step, index) => <li key={step.title}><span className={styles.flowNumber}>0{index + 1}</span><strong>{step.title}</strong><p>{step.description}</p><span className={styles.flowLabel}>{step.label}</span></li>)}
              </ol>
              <p className={styles.previewFoot}>{t({ en: "Sign in to try it. Your notes and credit history stay in your own account.", zh: "登录后即可体验，笔记与积分记录归属于你自己的账户。" })}</p>
            </div>
          </div>
        </div>
        <div className={styles.templateNote}><strong>{t({ en: "Free template. Your own product.", zh: "模板免费，产品由你定义。" })}</strong><p>{t({ en: "The pricing page shows example plans for a SaaS, not a price for the source code. Hosting, email, and payment providers charge separately.", zh: "价格页展示 SaaS 套餐示例，获取源码无需购买。域名、服务器、邮件和支付服务的费用由相应服务商收取。" })}</p></div>
      </PageSection>

      <PageSection
        id="included"
        title={t({ en: "Build on the essentials.", zh: "从这些基础能力，开始你的产品。" })}
        description={t({ en: "Reuse the common flows. Add the tools, content, or AI capabilities that make your service useful.", zh: "复用通用流程，接入你的工具、内容或 AI 服务。" })}
      >
        <div className={styles.features}>
          {content.features.map((feature) => <article className={styles.feature} id={feature.label === "06" ? "analytics" : undefined} key={feature.label}><span className={styles.featureNumber}>{feature.label}</span><div><h3><Link href={["/features/authentication", "/features/subscriptions", "/docs/invitations", "/docs/auth-email", "/resources", "/docs/admin"][Number(feature.label) - 1]}>{feature.title}</Link></h3><p>{feature.description}</p><span className={styles.featureDetail}>{feature.detail}</span></div></article>)}
        </div>
        <div className={styles.adminNote}><span className={styles.kicker}>ADMIN</span><p>{t({ en: "Your customers manage their own accounts. The separate admin console manages site-wide records and settings.", zh: "用户管理自己的账户；独立管理员后台处理全站记录与配置。" })}</p><Link href="/docs/admin">{t({ en: "Admin guide", zh: "管理员指南" })} <span aria-hidden="true">↗</span></Link></div>
      </PageSection>

      <PageSection id="try-it" title={t({ en: "Take it for a spin.", zh: "像你的用户一样，亲自体验。" })} description={t({ en: "Start here without deploying your own copy. Available flows depend on the services enabled on this site.", zh: "无需先部署，从这个站点开始体验。可用流程以本站启用的服务为准。" })}>
        {demo && <aside className={styles.demoNotice} aria-label={t({ en: "Demo payment information", zh: "演示支付说明" })}><span className={styles.modeBadge}>{t({ en: "DEMO", zh: "在线体验" })}</span><p>{t({ en: "Accounts and referrals are saved. Stripe test-mode purchases do not move real money; check the Checkout mode before using a test card.", zh: "账号与邀请记录会保存。Stripe 测试模式下的付款不涉及真实资金，使用测试卡前请确认收银台模式。" })}</p><Link href="/docs/payments">{t({ en: "Payment guide", zh: "支付指南" })} <span aria-hidden="true">↗</span></Link></aside>}
        <ol className={styles.journey}>
          {content.steps.map((step, index) => <li className={styles.step} key={step.title}><span className={styles.stepNumber}>0{index + 1}</span><h3>{step.title}</h3><p>{step.description}</p><Link href={step.href}>{step.action}<span aria-hidden="true">↗</span></Link></li>)}
        </ol>
      </PageSection>

      <PageSection id="setup" title={t({ en: "Make it yours.", zh: "准备好，把它变成你的产品。" })} description={t({ en: "Follow the guides to connect your domain, payments, and referrals.", zh: "跟着指南接入域名、支付和邀请流程。" })}>
        <div className={styles.guides}>
          {["domains", "payments", "invitations"].map((slug, index) => {
            const guide = documentation.find((item) => item.id === slug);
            if (!guide) return null;
            return <Link className={styles.guide} href={`/docs/${guide.id}`} id={slug === "payments" ? "test-payment" : slug === "invitations" ? "sharing" : undefined} key={guide.id}><span className={styles.guideNumber}>0{index + 1}</span><div><h3>{t(guide.title)}</h3><p>{t(guide.summary)}</p></div><span className={styles.guideArrow} aria-hidden="true">↗</span></Link>;
          })}
        </div>
        <div className={styles.closing}><div><span className={styles.kicker}>{t({ en: "YOUR NEXT SAAS STARTS HERE", zh: "你的下一个 SaaS，从这里开始" })}</span><h3>{t({ en: "Put your next sprint into your product.", zh: "把下一轮开发，留给核心业务。" })}</h3></div><div className={styles.closingActions}><a className="button primary" href="https://github.com/brizenchi/go-modules" rel="noopener noreferrer">{t({ en: "Get the source", zh: "获取模板源码" })}<span aria-hidden="true">↗</span></a><CTAButton href="/docs">{t({ en: "Browse all guides", zh: "浏览全部指南" })}</CTAButton></div></div>
      </PageSection>
    </SiteShell>
  );
}
