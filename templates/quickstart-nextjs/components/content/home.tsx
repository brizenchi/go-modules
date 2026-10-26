"use client";

import { useSitePresentation } from "@/components/site-presentation";
import Image from "next/image";
import { getHomeContent, siteConfig } from "@/content/site";
import { LocalizedLink as Link } from "@/components/localized-link";
import { SiteShell } from "@/components/site-shell";
import { StructuredData } from "@/components/structured-data";
import { faqStructuredData } from "@/lib/structured-data";
import { CTAButton, PageSection } from "@/components/ui";
import { documentation } from "@/content/docs";
import { appEnv } from "@/lib/env";
import { useI18n } from "@/lib/i18n";
import { latestGuides, uniqueContent, type DiscoveryEntry } from "@/lib/content-discovery";
import { ContentCards } from "@/components/content-discovery";
import { useDirectoryFilter } from "@/components/navigation-index";
import styles from "./home.module.css";

export function HomeContent({ entries }: { entries: DiscoveryEntry[] }) {
  const { t, locale } = useI18n();
  const shown = useDirectoryFilter();
 const presentation=useSitePresentation();
  const demo = appEnv.demoMode;
  const content = t(getHomeContent(demo));
  const cases = uniqueContent(entries).filter(entry => entry.kind === "use-cases").slice(0, 3);
  const latest = latestGuides(entries);

  return (
    <SiteShell
      pagePath="/"
      eyebrow={t(siteConfig.hero.eyebrow)}
      title={(locale === "zh" ? presentation?.headline_zh : presentation?.headline_en) || t(siteConfig.hero.title)}
      description={(locale === "zh" ? presentation?.description_zh : presentation?.description) || t(siteConfig.hero.description)}
      showEnvironment={false}
      heroVisual={
        <figure className={styles.heroPreview}>
          <div className={styles.previewChrome}><span aria-hidden="true">● ● ●</span><span>{t({ en: "Account center", zh: "账户中心" })}</span><span>{t({ en: "PREVIEW", zh: "界面预览" })}</span></div>
          <Image src={siteConfig.preview[locale].src} alt={siteConfig.preview[locale].alt} width={1200} height={640} priority sizes="(max-width: 980px) 100vw, 680px" />
          <figcaption>{t(siteConfig.preview.caption)}</figcaption>
        </figure>
      }
      heroFootnote={t(siteConfig.hero.note)}
      actions={<><CTAButton href={siteConfig.hero.primaryAction.href} primary>{t(siteConfig.hero.primaryAction.label)}</CTAButton><CTAButton href={siteConfig.hero.secondaryAction.href}>{t(siteConfig.hero.secondaryAction.label)}</CTAButton></>}
      toc={[
        { id: "what-is-template", label: t({ en: "Explore the workflow", zh: "看看使用流程" }) },
        { id: "included", label: t({ en: "Included capabilities", zh: "已整合的能力" }) },
        { id: "try-it", label: t({ en: "Try it yourself", zh: "亲自体验" }) },
        { id: "setup", label: t({ en: "Build your product", zh: "开始构建" }) }
      ]}
    >
      <section className={styles.spaces} id="who-uses-what" aria-label={t({ en: "Explore the template", zh: "了解模板的三个空间" })}>
        {[
          { href: "/features", label: t({ en: "Public website", zh: "产品官网" }), detail: t({ en: "Features, pricing and guides", zh: "功能、价格与使用指南" }) },
          { href: "/account", label: t({ en: "Account center", zh: "账户中心" }), detail: t({ en: "Subscriptions, credits and rewards", zh: "订阅、积分与推荐奖励" }) },
          { href: "/docs/admin", label: t({ en: "Admin console", zh: "独立管理后台" }), detail: t({ en: "Site-wide records and settings", zh: "全站记录与运营配置" }) }
        ].map((entry, index) => <Link href={entry.href} key={entry.href}><span className={styles.entryNumber}>0{index + 1}</span><span><strong>{entry.label}</strong><small>{entry.detail}</small></span><span aria-hidden="true">↗</span></Link>)}
      </section>

      <PageSection id="what-is-template" title={t({ en: "Shared foundations. Your own product.", zh: "复用基础能力，定义自己的产品。" })} description={t({ en: "The starter connects common services. You decide what customers create, how they use it and what they pay for.", zh: "模板接好通用服务；用户创造什么、怎样使用，以及为什么付费，由你的产品决定。" })}>
        <div className={styles.productPreview}><div className={styles.previewContent}>
          <ol className={styles.previewFlow}>
            {content.flow.map((step, index) => <li key={step.title}><span className={styles.flowNumber}>0{index + 1}</span><strong>{step.title}</strong><p>{step.description}</p><span className={styles.flowLabel}>{step.label}</span></li>)}
          </ol>
        </div></div>
      </PageSection>

      <PageSection
        id="included"
        title={t({ en: "The essentials, already connected.", zh: "基础能力，已经接好。" })}
        description={t({ en: "Reuse the integrations and build the product-specific experience your customers need.", zh: "复用接入能力，构建你的用户真正需要的产品。" })}
      >
        <div className={styles.features}>
          {content.features.map((feature) => <article className={styles.feature} id={feature.label === "06" ? "analytics" : undefined} key={feature.label}><span className={styles.featureNumber}>{feature.label}</span><div><h3><Link href={feature.href}>{feature.title}</Link></h3><p>{feature.description}</p><span className={styles.featureDetail}>{feature.detail}</span></div></article>)}
        </div>
        <div className={styles.adminNote}><span className={styles.kicker}>ADMIN</span><p>{t({ en: "Your customers manage their own accounts. The separate admin console manages site-wide records and settings.", zh: "用户管理自己的账户；独立管理员后台处理全站记录与配置。" })}</p><Link href="/docs/admin">{t({ en: "Admin guide", zh: "管理员指南" })} <span aria-hidden="true">↗</span></Link></div>
      </PageSection>

      {cases.length > 0 ? <PageSection id="use-cases" title={t({ en: "A starting point for your product.", zh: "找到适合你的产品场景。" })}
        description={t({ en: "See how the shared foundations fit different ways to deliver and charge for your product.", zh: "了解通用基础能力如何用于不同的产品交付与收费方式。" })}>
        <ContentCards entries={cases} />
        <div className={styles.sectionLinks}><Link href="/use-cases">{t({ en: "Explore all use cases", zh: "查看全部使用场景" })}<span aria-hidden="true">→</span></Link>{shown("/compare") ? <Link href="/compare">{t({ en: "Compare starting points", zh: "比较开发方案" })}<span aria-hidden="true">→</span></Link> : null}</div>
      </PageSection> : null}

      <PageSection id="try-it" title={t({ en: "Take it for a spin.", zh: "像你的用户一样，亲自体验。" })} description={t({ en: "Start here without deploying your own copy. Available flows depend on the services enabled on this site.", zh: "无需先部署，从这个站点开始体验。可用流程以本站启用的服务为准。" })}>
        {demo && <aside className={styles.demoNotice} aria-label={t({ en: "Demo payment information", zh: "演示支付说明" })}><span className={styles.modeBadge}>{t({ en: "DEMO", zh: "在线体验" })}</span><p>{t({ en: "Accounts and referrals are saved. Stripe test-mode purchases do not move real money; check the Checkout mode before using a test card.", zh: "账号与邀请记录会保存。Stripe 测试模式下的付款不涉及真实资金，使用测试卡前请确认收银台模式。" })}</p><Link href="/docs/payments">{t({ en: "Payment guide", zh: "支付指南" })} <span aria-hidden="true">↗</span></Link></aside>}
        <ol className={styles.journey}>
          {content.steps.map((step, index) => <li className={styles.step} key={step.title}><span className={styles.stepNumber}>0{index + 1}</span><h3>{step.title}</h3><p>{step.description}</p><Link href={step.href}>{step.action}<span aria-hidden="true">↗</span></Link></li>)}
        </ol>
      </PageSection>

      <PageSection id="questions" title={t({ en: "A few things before you start.", zh: "开始前，你可能想了解。" })}>
        <StructuredData data={faqStructuredData(siteConfig.faq.map(item => ({ question: t(item.question), answer: t(item.answer) })))} />
        <div className={styles.faq}>
          {siteConfig.faq.map(item => <details key={item.question.en}><summary>{t(item.question)}<span aria-hidden="true">+</span></summary><p>{t(item.answer)}</p></details>)}
        </div>
      </PageSection>

      <PageSection id="setup" title={t({ en: "Make it yours.", zh: "准备好，开始构建。" })} description={t({ en: "Follow the guides to connect your domain, payments, and referrals.", zh: "跟着指南接入域名、支付和邀请流程。" })}>
        <div className={styles.guides}>
          {["domains", "payments", "invitations"].map((slug, index) => {
            const guide = documentation.find((item) => item.id === slug);
            if (!guide) return null;
            return <Link className={styles.guide} href={`/docs/${guide.id}`} id={slug === "payments" ? "test-payment" : slug === "invitations" ? "sharing" : undefined} key={guide.id}><span className={styles.guideNumber}>0{index + 1}</span><div><h3>{t(guide.title)}</h3><p>{t(guide.summary)}</p></div><span className={styles.guideArrow} aria-hidden="true">↗</span></Link>;
          })}
        </div>
        <div className={styles.closing}><div><span className={styles.kicker}>{t({ en: "YOUR NEXT SAAS STARTS HERE", zh: "你的下一个 SaaS，从这里开始" })}</span><h3>{t({ en: "Put your next sprint into your product.", zh: "把下一轮开发，留给核心业务。" })}</h3>{shown("/about") ? <Link className={styles.aboutLink} href="/about">{t({ en: "About the project", zh: "了解这个项目" })}<span aria-hidden="true"> →</span></Link> : null}</div><div className={styles.closingActions}>{shown("/resources") ? <CTAButton href="/resources">{t({ en: "Browse all guides", zh: "浏览全部指南" })}</CTAButton> : null}</div></div>
      </PageSection>

      <PageSection id="latest-guides" title={t({ en: "Keep learning. Keep building.", zh: "继续了解，持续构建。" })}
        description={t({ en: "Read the latest published guides and follow product updates.", zh: "查阅最近发布的指南，了解产品更新。" })}>
        {latest.length > 0 ? <ContentCards entries={latest} /> : null}
        <div className={styles.sectionLinks}>{shown("/resources") ? <Link href="/resources">{t({ en: "All guides", zh: "指南总览" })}<span aria-hidden="true">→</span></Link> : null}{shown("/updates") ? <Link href="/updates">{t({ en: "Product updates", zh: "更新日志" })}<span aria-hidden="true">→</span></Link> : null}</div>
      </PageSection>
    </SiteShell>
  );
}
