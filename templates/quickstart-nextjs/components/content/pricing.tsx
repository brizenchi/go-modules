"use client";

import Link from "next/link";
import { SiteShell } from "@/components/site-shell";
import { DetailRows, PageSection } from "@/components/ui";
import { PricingCard } from "@/components/marketing";
import { appEnv } from "@/lib/env";
import { useI18n } from "@/lib/i18n";

export function ProductPricing() {
  const { t } = useI18n();
  const plans = [
    { tier: "Starter", price: t({ en: "US$4 / month", zh: "US$4 / 月" }), subtitle: t({ en: "An entry plan for getting started with the product.", zh: "适合开始使用产品的入门订阅。" }), features: t({ en: ["Yearly: US$40 / year", "Starter yearly includes a 3-day trial", "Core auth and account experience", "For an early product launch"], zh: ["年付：US$40 / 年", "Starter 年付包含 3 天试用", "基础登录与账号体验", "适合产品早期使用"] }), cta: t({ en: "Choose Starter", zh: "选择 Starter" }) },
    { tier: "Pro", price: t({ en: "US$14 / month", zh: "US$14 / 月" }), subtitle: t({ en: "A recurring plan for regular product use.", zh: "适合持续使用产品的常规订阅。" }), features: t({ en: ["Yearly: US$140 / year", "Subscription management", "Invoices and payment method visibility", "A plan for ongoing use"], zh: ["年付：US$140 / 年", "订阅管理", "查看发票与支付方式", "适合持续使用"] }), cta: t({ en: "Choose Pro", zh: "选择 Pro" }) },
    { tier: "Premium", price: t({ en: "US$59 / month", zh: "US$59 / 月" }), subtitle: t({ en: "The highest recurring tier in this example catalog.", zh: "示例商品中最高等级的持续订阅。" }), features: t({ en: ["Yearly: US$590 / year", "Top recurring subscription tier", "Upgrade path beyond Pro", "Compatible with credits purchases"], zh: ["年付：US$590 / 年", "最高等级的持续订阅", "可从 Pro 升级", "可搭配积分购买使用"] }), cta: t({ en: "Choose Premium", zh: "选择 Premium" }), featured: true },
    { tier: "Lifetime", price: "US$999", subtitle: t({ en: "One purchase for the product’s lifetime access offer.", zh: "一次购买，获得产品的终身使用权益。" }), features: t({ en: ["One-time buyout tier", "No recurring billing cycle", "Lifetime product entitlement", "Separate from recurring plan changes"], zh: ["一次性买断", "没有周期性扣费", "产品终身使用权益", "与持续订阅的套餐变更分开管理"] }), cta: t({ en: "Buy Lifetime", zh: "购买 Lifetime" }) },
    { tier: t({ en: "Package", zh: "积分包" }), price: "US$4.90", subtitle: t({ en: "A fixed credits package for usage inside the product.", zh: "用于产品内消费的固定积分包。" }), features: t({ en: ["Fixed credits package", "Stripe hosted Checkout", "Multiple packages supported", "Product credits, separate from a subscription"], zh: ["固定积分包", "通过 Stripe Checkout 结账", "支持购买多份", "产品积分，与订阅分开管理"] }), cta: t({ en: "Buy Package", zh: "购买积分包" }) }
  ];
  const notes = [
    { label: { en: "Free template", zh: "模板免费" }, title: { en: "These plans are for your product", zh: "套餐用于你构建的产品" }, body: { en: "The starter is shared for free. This page demonstrates a subscription and credits catalog you can adapt to the SaaS you build with it.", zh: "启动模板免费分享。本页演示了订阅与积分商品结构，你可以根据自己构建的 SaaS 调整它。" } },
    { label: { en: "Subscriptions", zh: "持续订阅" }, title: { en: "Choose a billing interval", zh: "选择适合的计费周期" }, body: { en: "Starter, Pro, and Premium offer monthly and yearly options. Review the selected plan, interval, and total in Checkout before confirming.", zh: "Starter、Pro 和 Premium 提供月付与年付选项。确认前请在 Checkout 中核对套餐、周期和总金额。" } },
    { label: { en: "One-time purchases", zh: "一次性购买" }, title: { en: "Lifetime access and credits", zh: "终身权益与积分" }, body: { en: "Lifetime is a buyout offer. A Package adds product credits. Each follows its own entitlement rules and is separate from recurring subscriptions.", zh: "Lifetime 是买断方案，积分包用于补充产品积分。它们各自遵循对应权益规则，与持续订阅分开管理。" } },
    { label: { en: "Your account", zh: "账户管理" }, title: { en: "Manage everything from billing", zh: "在计费中心继续管理" }, body: { en: "Sign in to review your subscription, view invoice records, and start a purchase. The configured payment catalog determines the final Checkout amount.", zh: "登录后可以查看订阅、发票记录并发起购买。Checkout 的最终金额以已配置的支付商品为准。" } }
  ];
  return (
    <SiteShell eyebrow={t({ en: "Pricing", zh: "价格方案" })} title={t({ en: "A plan for how you use the product.", zh: "找到适合你的使用方案。" })} description={t({ en: "Explore recurring plans, lifetime access, and credits. This is an example catalog for the SaaS you build; the template itself is free.", zh: "选择持续订阅、终身权益或积分包。这是为你构建的 SaaS 准备的示例商品，模板本身免费。" })} sideTitle={t({ en: "At a glance", zh: "方案概览" })} showEnvironment={false}
      sideBody={<DetailRows rows={[{ label: t({ en: "Template", zh: "模板" }), value: t({ en: "Free to use", zh: "免费使用" }) }, { label: t({ en: "Payment provider", zh: "支付服务" }), value: "Stripe" }, { label: t({ en: "Manage purchases", zh: "购买与管理" }), value: <Link href="/billing">{t({ en: "Billing workspace →", zh: "计费中心 →" })}</Link> }]} />}
      breadcrumbs={[{ href: "/", label: t({ en: "Home", zh: "首页" }) }, { label: t({ en: "Pricing", zh: "价格" }) }]}
      toc={[{ id: "plans", label: t({ en: "Plans", zh: "套餐选择" }) }, { id: "faq", label: t({ en: "Before you choose", zh: "选择前了解" }) }]}>
      {appEnv.demoMode ? <div className="notice info"><strong>{t({ en: "Payment demo", zh: "支付演示" })}</strong><p>{t({ en: "This demo is intended for Stripe test payments. Registration and invitation records are real. Follow the overview’s test walkthrough before trying Checkout.", zh: "此演示用于体验 Stripe 测试支付，注册与邀请记录是真实的。进入 Checkout 前，可以先查看总览中的测试说明。" })} <Link href="/#test-payment">{t({ en: "Open the walkthrough →", zh: "查看体验说明 →" })}</Link></p></div> : null}
      <PageSection id="plans" title={t({ en: "Choose your plan", zh: "选择你的方案" })} description={t({ en: "Continue to billing to select the available plan and billing interval.", zh: "进入计费中心，选择已开放的套餐和计费周期。" })}><div className="pricing-grid">{plans.map((plan) => <PricingCard key={plan.tier} {...plan} href="/billing" />)}</div></PageSection>
      <PageSection id="faq" title={t({ en: "Before you choose", zh: "选择前了解" })} description={t({ en: "Know what the template, subscription, and credits each provide.", zh: "了解模板、订阅和积分各自的用途。" })}><div className="feature-grid">{notes.map((note) => <article className="feature-card" key={note.label.en}><span className="panel-kicker">{t(note.label)}</span><h3>{t(note.title)}</h3><p>{t(note.body)}</p></article>)}</div></PageSection>
    </SiteShell>
  );
}
