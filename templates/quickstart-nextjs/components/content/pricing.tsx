"use client";

import { useState } from "react";
import { LocalizedLink as Link } from "@/components/localized-link";
import { SiteShell } from "@/components/site-shell";
import { DetailRows, PageSection } from "@/components/ui";
import { billingIntentHref } from "@/lib/billing-intent";
import { appEnv } from "@/lib/env";
import { useI18n } from "@/lib/i18n";
import styles from "./pricing.module.css";

const plans = [
  { slug: "starter", name: "Starter", monthly: "4", yearly: "40", label: { en: "Getting started", zh: "轻松开始" }, description: { en: "An entry subscription for your first steps with the product.", zh: "从入门订阅开始，体验产品的日常使用。" } },
  { slug: "pro", name: "Pro", monthly: "14", yearly: "140", label: { en: "Regular use", zh: "持续使用" }, description: { en: "A recurring plan for making the product part of your routine.", zh: "适合经常使用产品，按自己的节奏持续订阅。" } },
  { slug: "premium", name: "Premium", monthly: "59", yearly: "590", label: { en: "Highest tier", zh: "更高等级" }, description: { en: "The highest subscription tier in this example catalog.", zh: "示例商品中的最高订阅等级，可从其他套餐升级。" } }
] as const;

export function ProductPricing() {
  const { t } = useI18n();
  const [interval, setInterval] = useState<"monthly" | "yearly">("monthly");
  const yearly = interval === "yearly";
  const notes = [
    { title: { en: "Is the template free?", zh: "模板本身收费吗？" }, body: { en: "Yes. The starter template is free to use. The prices here demonstrate a product catalog for the SaaS you build with it, and can be adapted to your own product.", zh: "模板免费使用。本页价格用于演示你所构建 SaaS 的商品结构，可以根据自己的产品调整。" } },
    { title: { en: "What happens after I choose a plan?", zh: "选择套餐后会发生什么？" }, body: { en: "Your selection carries into billing. Sign in to review the available plan and interval before starting Checkout. Choosing a plan here does not charge you.", zh: "你的选择会带入计费中心。登录后可以核对已开放的套餐与周期，再进入结账。在本页选择套餐不会产生扣款。" } },
    { title: { en: "Which price will I pay?", zh: "实际支付金额以哪里为准？" }, body: { en: "These are example prices in USD. The configured payment catalog determines the final amount. Review the price, trial terms, and renewal date in Checkout before confirming.", zh: "本页展示美元示例价格，实际金额由已配置的支付商品决定。请在结账时核对价格、试用条件和续费日期，再确认购买。" } },
    { title: { en: "How do lifetime access and credits work?", zh: "终身权益和积分有什么区别？" }, body: { en: "Lifetime is a one-time access purchase. Credits are used inside the product and can be purchased separately. Each follows its own entitlement rules, separate from recurring subscriptions.", zh: "Lifetime 是一次性购买的使用权益；积分用于产品内消费，可以单独购买。两者各有对应的权益规则，与周期订阅分开管理。" } }
  ];
  return (
    <SiteShell variant="compact" eyebrow={t({ en: "Pricing", zh: "价格方案" })} title={t({ en: "A plan that fits your pace.", zh: "找到适合你的使用方案。" })} description={t({ en: "Start with a subscription. Add credits when you need them. Explore an example catalog for your SaaS — the template itself is free.", zh: "选择适合的订阅，按需补充积分。这是为你的 SaaS 准备的示例商品，模板本身免费。" })} sideTitle={t({ en: "Your account, in one place", zh: "在一个地方管理购买" })} showEnvironment={false}
      sideBody={<DetailRows rows={[{ label: t({ en: "Template", zh: "模板" }), value: t({ en: "Free to use", zh: "免费使用" }) }, { label: t({ en: "Example currency", zh: "示例币种" }), value: "USD" }, { label: t({ en: "Already subscribed?", zh: "已有订阅？" }), value: <Link href="/billing">{t({ en: "Manage billing →", zh: "管理订阅 →" })}</Link> }]} />}
      breadcrumbs={[{ href: "/", label: t({ en: "Home", zh: "首页" }) }, { label: t({ en: "Pricing", zh: "价格" }) }]}
      toc={[{ id: "plans", label: t({ en: "Subscriptions", zh: "订阅套餐" }) }, { id: "one-time", label: t({ en: "One-time purchases", zh: "一次性购买" }) }, { id: "faq", label: t({ en: "Common questions", zh: "常见问题" }) }]}>
      {appEnv.demoMode ? <div className="notice info"><strong>{t({ en: "Payment demo", zh: "支付演示" })}</strong><p>{t({ en: "This demo is intended for Stripe test payments. Registration and invitation records are real. Follow the overview’s test walkthrough before trying Checkout.", zh: "此演示用于体验 Stripe 测试支付，注册与邀请记录是真实的。进入 Checkout 前，可以先查看总览中的测试说明。" })} <Link href="/#test-payment">{t({ en: "Open the walkthrough →", zh: "查看体验说明 →" })}</Link></p></div> : null}
      <PageSection id="plans" title={t({ en: "Choose your subscription", zh: "选择你的订阅" })} description={t({ en: "Three tiers. One place to manage your subscription.", zh: "三种订阅等级，都可以在计费中心统一管理。" })}>
        <div className={styles.planToolbar}>
          <p className={styles.catalogNote}>{t({ en: "Example prices · USD", zh: "示例价格 · 美元计价" })}</p>
          <fieldset className={styles.intervalControl}>
            <legend className={styles.visuallyHidden}>{t({ en: "Billing interval", zh: "计费周期" })}</legend>
            <label><input type="radio" name="pricing-interval" value="monthly" checked={!yearly} onChange={() => setInterval("monthly")} /><span>{t({ en: "Monthly", zh: "月付" })}</span></label>
            <label><input type="radio" name="pricing-interval" value="yearly" checked={yearly} onChange={() => setInterval("yearly")} /><span>{t({ en: "Yearly", zh: "年付" })}<small>{t({ en: "Save 2 months", zh: "省 2 个月" })}</small></span></label>
          </fieldset>
        </div>
        <div className={styles.planGrid}>
          {plans.map((plan) => (
            <article key={plan.slug} className={`${styles.planCard}${plan.slug === "pro" ? ` ${styles.emphasized}` : ""}`} aria-labelledby={`plan-${plan.slug}`}>
              <div className={styles.planHeading}><h3 id={`plan-${plan.slug}`}>{plan.name}</h3><span className={styles.planLabel}>{t(plan.label)}</span></div>
              <p className={styles.planDescription}>{t(plan.description)}</p>
              <div className={styles.priceBlock}>
                <p className={styles.price}><span className={styles.currency}>US$</span><strong>{yearly ? plan.yearly : plan.monthly}</strong><span className={styles.period}>{yearly ? t({ en: "/ year", zh: "/ 年" }) : t({ en: "/ month", zh: "/ 月" })}</span></p>
                <p className={styles.priceNote}>{yearly ? t({ en: "One payment, billed every year.", zh: "每年一次付清，按年续费。" }) : t({ en: "Billed monthly. Yearly billing also available.", zh: "按月续费，也可以选择年付。" })}</p>
              </div>
              <Link className={`button${plan.slug === "pro" ? " primary" : ""} ${styles.planButton}`} href={billingIntentHref({ product: "subscription", plan: plan.slug, interval })}>{t({ en: `Choose ${plan.name}`, zh: `选择 ${plan.name}` })}<span aria-hidden="true">→</span></Link>
              <p className={styles.planFootnote}>{plan.slug === "starter" ? t({ en: "The yearly plan includes a 3-day trial.", zh: "年付方案包含 3 天试用。" }) : t({ en: "Review your selection before checkout.", zh: "进入结账前可再次核对选择。" })}</p>
            </article>
          ))}
        </div>
        <div className={styles.sharedTools}>
          <strong>{t({ en: "Your billing workspace", zh: "统一的计费中心" })}</strong>
          <ul>{t({ en: ["Manage your plan", "View invoices", "Manage payment methods"], zh: ["管理订阅套餐", "查看发票记录", "管理付款方式"] }).map((feature) => <li key={feature}><span aria-hidden="true">✓</span>{feature}</li>)}</ul>
        </div>
      </PageSection>
      <PageSection id="one-time" title={t({ en: "Prefer a one-time purchase?", zh: "按需选择，一次性购买" })} description={t({ en: "Lifetime access and credit packages are separate from recurring subscriptions.", zh: "终身权益和积分包独立于周期订阅，按自己的需要选择。" })}>
        <div className={styles.oneTimeGrid}>
          <article className={styles.oneTimeCard}>
            <div className={styles.oneTimeHeading}>
              <div><span className={styles.sectionLabel}>{t({ en: "Lifetime access", zh: "终身权益" })}</span><h3>Lifetime</h3></div>
              <p className={styles.oneTimePrice}><span>US$</span><strong>999</strong><small>{t({ en: "one time", zh: "一次付清" })}</small></p>
            </div>
            <p>{t({ en: "One purchase for the product’s lifetime access offer, with no recurring billing cycle.", zh: "一次购买，获得产品提供的终身使用权益，无周期性扣费。" })}</p>
            <Link className={`button ${styles.oneTimeButton}`} href={billingIntentHref({ product: "lifetime" })}>{t({ en: "Explore Lifetime", zh: "查看终身方案" })}<span aria-hidden="true">→</span></Link>
          </article>
          <article className={styles.oneTimeCard}>
            <div className={styles.oneTimeHeading}>
              <div><span className={styles.sectionLabel}>{t({ en: "On-demand usage", zh: "按需补充" })}</span><h3>{t({ en: "Credit package", zh: "积分包" })}</h3></div>
              <p className={styles.oneTimePrice}><span>US$</span><strong>4.90</strong><small>{t({ en: "per package", zh: "每包" })}</small></p>
            </div>
            <p>{t({ en: "Add credits for use inside the product. Choose the number of packages in billing before checkout.", zh: "补充产品内消费所需的积分，在计费中心选择购买份数，再确认结账。" })}</p>
            <Link className={`button ${styles.oneTimeButton}`} href={billingIntentHref({ product: "credits" })}>{t({ en: "Explore credits", zh: "查看积分包" })}<span aria-hidden="true">→</span></Link>
          </article>
        </div>
        <p className={styles.purchaseNote}>{t({ en: "All amounts shown are examples. Available products, entitlements, and final prices depend on the configured catalog.", zh: "以上金额均为示例。可购买的商品、具体权益和最终价格，以实际配置的商品为准。" })}</p>
      </PageSection>
      <PageSection id="faq" title={t({ en: "A few things to know", zh: "选择前，了解这些" })}>
        <div className={styles.faqGrid}>{notes.map((note) => <article className={styles.faqItem} key={note.title.en}><h3>{t(note.title)}</h3><p>{t(note.body)}</p></article>)}</div>
      </PageSection>
    </SiteShell>
  );
}
