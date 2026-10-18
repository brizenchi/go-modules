"use client";

import { useEffect, useState } from "react";
import { LocalizedLink as Link } from "@/components/localized-link";
import { SiteShell } from "@/components/site-shell";
import { DetailRows, PageSection } from "@/components/ui";
import { PendingLabel } from "@/components/pending-label";
import { pricingIntentMatches, pricingCheckoutFailure, pricingCheckoutResultHref } from "@/lib/pricing-checkout";
import { SignInDialog } from "@/components/sign-in-dialog";
import { SubscriptionChangeDialog } from "@/components/subscription-change-dialog";
import { ConsoleError } from "@/components/console-kit";
import { canSelectSubscriptionPlan, subscriptionPlanIntent } from "@/lib/subscription-change";
import { billingText, lifetimePurchaseAction, pricingPlanPresentation } from "@/lib/billing-presentation";
import { hasOngoingRecurringSubscription } from "@/lib/billing-state";
import { usePricingCheckout } from "@/components/use-pricing-checkout";
import { appEnv } from "@/lib/env";
import { useI18n } from "@/lib/i18n";
import styles from "./pricing.module.css";

const plans = [
  { slug: "starter", name: "Starter", monthly: "4", yearly: "40", label: { en: "Getting started", zh: "轻松开始" }, description: { en: "An entry subscription for your first steps with the product.", zh: "从入门订阅开始，体验产品的日常使用。" } },
  { slug: "pro", name: "Pro", monthly: "14", yearly: "140", label: { en: "Regular use", zh: "持续使用" }, description: { en: "A recurring plan for making the product part of your routine.", zh: "适合经常使用产品，按自己的节奏持续订阅。" } },
  { slug: "premium", name: "Premium", monthly: "59", yearly: "590", label: { en: "Highest tier", zh: "更高等级" }, description: { en: "The highest subscription tier in this example catalog.", zh: "示例商品中的最高订阅等级，可从其他套餐升级。" } }
] as const;

function CheckoutFeedback({ failure }: { failure: ReturnType<typeof pricingCheckoutFailure> | null }) {
  const { t } = useI18n();
  if (!failure) return null;
  return <div className={styles.checkoutFeedback} role="alert">
    <p>{t(failure.messages)}</p>
    {failure.manage ? <Link className="button" href="/billing">{t(billingText.viewBilling)}</Link> : null}
    {failure.requestId ? <small>{t({ en: "Request ID", zh: "请求编号" })}: {failure.requestId}</small> : null}
  </div>;
}

export function ProductPricing() {
  const { t } = useI18n();
  const [interval, setInterval] = useState<"monthly" | "yearly">("monthly");
  const checkout = usePricingCheckout();
  const [quantity, setQuantity] = useState(1);
  useEffect(() => {
    if (checkout.restored?.intent.product === "subscription") setInterval(checkout.restored.intent.interval);
    if (checkout.restored?.intent.product === "credits") setQuantity(checkout.restored.quantity);
  }, [checkout.restored]);
  const yearly = interval === "yearly";
  const subscriptionDisabled = checkout.busy || checkout.accountLoading || Boolean(checkout.accountError);
  const lifetimeAction = lifetimePurchaseAction(checkout.subscription);
  const hasLifetime = lifetimeAction === "owned";
  const ongoingSubscription = hasOngoingRecurringSubscription(checkout.subscription);
  const notes = [
    { title: { en: "Is the template free?", zh: "模板本身收费吗？" }, body: { en: "Yes. The starter template is free to use. The prices here demonstrate a product catalog for the SaaS you build with it, and can be adapted to your own product.", zh: "模板免费使用。本页价格用于演示你所构建 SaaS 的商品结构，可以根据自己的产品调整。" } },
    { title: { en: "What happens after I choose a plan?", zh: "选择套餐后会发生什么？" }, body: { en: "New subscriptions continue to Stripe Checkout after sign-in. Existing subscribers review the amount and effective date before confirming a plan or billing interval change.", zh: "新订阅在登录后进入 Stripe 结账。已有订阅则先核对变更金额与生效时间，再确认升级、降级或切换计费周期。" } },
    { title: { en: "Which price will I pay?", zh: "实际支付金额以哪里为准？" }, body: { en: "These are example prices in USD. The configured payment catalog determines the final amount. Review the price, trial terms, and renewal date in Checkout before confirming.", zh: "本页展示美元示例价格，实际金额由已配置的支付商品决定。请在结账时核对价格、试用条件和续费日期，再确认购买。" } },
    { title: { en: "How do lifetime access and credits work?", zh: "终身权益和积分有什么区别？" }, body: { en: "Lifetime is a one-time access purchase. Credits are used inside the product and can be purchased separately. Each follows its own entitlement rules, separate from recurring subscriptions.", zh: "Lifetime 是一次性购买的使用权益；积分用于产品内消费，可以单独购买。两者各有对应的权益规则，与周期订阅分开管理。" } }
  ];
  return (
    <SiteShell className={styles.page} variant="compact" eyebrow={t({ en: "Pricing", zh: "价格方案" })} title={t({ en: "A plan that fits your pace.", zh: "找到适合你的使用方案。" })} description={t({ en: "Start with a subscription. Add credits when you need them. Explore an example catalog for your SaaS — the template itself is free.", zh: "选择适合的订阅，按需补充积分。这是为你的 SaaS 准备的示例商品，模板本身免费。" })} sideTitle={t({ en: "Your account, in one place", zh: "在一个地方管理购买" })} showEnvironment={false}
      sideBody={<DetailRows rows={[{ label: t({ en: "Template", zh: "模板" }), value: t({ en: "Free to use", zh: "免费使用" }) }, { label: t({ en: "Example currency", zh: "示例币种" }), value: "USD" }, { label: t({ en: "Already subscribed?", zh: "已有订阅？" }), value: <Link href="/billing">{t({ en: "Manage billing →", zh: "管理订阅 →" })}</Link> }]} />}
      breadcrumbs={[{ href: "/", label: t({ en: "Home", zh: "首页" }) }, { label: t({ en: "Pricing", zh: "价格" }) }]}
      toc={[{ id: "plans", label: t({ en: "Subscriptions", zh: "订阅套餐" }) }, { id: "one-time", label: t({ en: "One-time purchases", zh: "一次性购买" }) }, { id: "faq", label: t({ en: "Common questions", zh: "常见问题" }) }]}>
      {appEnv.demoMode ? <div className="notice info"><strong>{t({ en: "Payment demo", zh: "支付演示" })}</strong><p>{t({ en: "This demo is intended for Stripe test payments. Registration and invitation records are real. Follow the overview’s test walkthrough before trying Checkout.", zh: "此演示用于体验 Stripe 测试支付，注册与邀请记录是真实的。进入 Checkout 前，可以先查看总览中的测试说明。" })} <Link href="/#test-payment">{t({ en: "Open the walkthrough →", zh: "查看体验说明 →" })}</Link></p></div> : null}
      {checkout.returnStatus ? <div className={styles.returnNotice} role="status"><p>{checkout.returnStatus === "success" ? t({ en: "View billing to confirm your payment result.", zh: "请前往订阅与账单确认付款结果。" }) : t(billingText.checkoutCanceled)}</p>{checkout.returnStatus === "success" ? <Link className="button" href={pricingCheckoutResultHref(checkout.returnStatus)}>{t({ en: "View purchase result", zh: "查看购买结果" })}</Link> : null}</div> : null}
      <PageSection id="plans" title={t({ en: "Choose your subscription", zh: "选择你的订阅" })} description={t({ en: "Three tiers. One place to manage your subscription.", zh: "三种订阅等级，都可以在计费中心统一管理。" })}>
        <div className={styles.planToolbar}>
          <fieldset className={styles.intervalControl}>
            <legend className={styles.visuallyHidden}>{t({ en: "Billing interval", zh: "计费周期" })}</legend>
            <label><input type="radio" name="pricing-interval" value="monthly" checked={!yearly} disabled={checkout.busy} onChange={() => setInterval("monthly")} /><span>{t({ en: "Monthly", zh: "月付" })}</span></label>
            <label><input type="radio" name="pricing-interval" value="yearly" checked={yearly} disabled={checkout.busy} onChange={() => setInterval("yearly")} /><span>{t({ en: "Yearly", zh: "年付" })}<small>{t({ en: "Save 2 months", zh: "省 2 个月" })}</small></span></label>
          </fieldset>
        </div>
        <div className={styles.planGrid}>
          {plans.map((plan) => {
            const action = subscriptionPlanIntent({ plan: plan.slug, interval }, checkout.subscription);
            const card = pricingPlanPresentation({ plan: plan.slug, interval }, plan.name, checkout.subscription);
            const current = card.current;
            const emphasized = plan.slug === "pro" && !ongoingSubscription && !hasLifetime;
            const selectable = !card.scheduled && canSelectSubscriptionPlan({ plan: plan.slug, interval }, checkout.subscription, {
              busy: checkout.busy, loading: checkout.accountLoading, failed: Boolean(checkout.accountError)
            });
            return <article key={plan.slug} className={`${styles.planCard}${emphasized ? ` ${styles.emphasized}` : ""}${current ? ` ${styles.currentPlan}` : ""}${card.scheduled ? ` ${styles.scheduledPlan}` : ""}`} aria-labelledby={`plan-${plan.slug}`}>
              <div className={styles.planHeading}><h3 id={`plan-${plan.slug}`}>{plan.name}</h3>{card.badge || !ongoingSubscription ? <span className={styles.planLabel}>{t(card.badge || plan.label)}</span> : null}</div>
              <p className={styles.planDescription}>{t(plan.description)}</p>
              <div className={styles.priceBlock}>
                <p className={styles.price}><span className={styles.currency}>US$</span><strong>{yearly ? plan.yearly : plan.monthly}</strong><span className={styles.period}>{yearly ? t({ en: "/ year", zh: "/ 年" }) : t({ en: "/ month", zh: "/ 月" })}</span></p>
                <p className={styles.priceNote}>{card.note ? t(card.note) : yearly ? t({ en: "One payment, billed every year.", zh: "每年一次付清，按年续费。" }) : t({ en: "Billed monthly. Yearly billing also available.", zh: "按月续费，也可以选择年付。" })}</p>
              </div>
              <button className={`button${emphasized ? " primary" : ""} ${styles.planButton}`} disabled={!selectable} aria-busy={checkout.busy && pricingIntentMatches(checkout.lastIntent, { product: "subscription", plan: plan.slug, interval })} onClick={() => checkout.purchase({ product: "subscription", plan: plan.slug, interval })} type="button">{checkout.busy && pricingIntentMatches(checkout.lastIntent, { product: "subscription", plan: plan.slug, interval }) ? <PendingLabel>{t(action === "subscribe" ? { en: "Opening checkout…", zh: "正在打开结账…" } : action === "keep" ? billingText.loading : { en: "Loading preview…", zh: "正在获取预览…" })}</PendingLabel> : checkout.accountLoading ? t(billingText.loading) : t(card.button)}{!card.scheduled && action !== "current" && action !== "lifetime" && !checkout.accountLoading ? <span aria-hidden="true">→</span> : null}</button>
              {pricingIntentMatches(checkout.lastIntent, { product: "subscription", plan: plan.slug, interval }) ? <CheckoutFeedback failure={checkout.failure} /> : null}
            </article>;
          })}
        </div>
        <ConsoleError error={checkout.accountError} retry={() => void checkout.revalidate()} />
        <div className={styles.sharedTools}>
          <strong>{t({ en: "Subscription & billing", zh: "订阅与账单" })}</strong>
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
            <p>{t({ en: "Pay in full once. Lifetime access starts after payment confirmation.", zh: "一次付清，付款成功后终身权益立即生效。" })}</p>
            {ongoingSubscription ? <p className={styles.lifetimeNote}>{t({ en: "Your recurring subscription ends after payment, with no prorated discount on Lifetime.", zh: "付款成功后结束原订阅，Lifetime 按全额购买，不抵扣订阅余款。" })} <Link href="/billing">{t(billingText.viewBilling)}</Link></p> : null}
            <button className={`button ${styles.oneTimeButton}`} type="button" disabled={subscriptionDisabled || hasLifetime} aria-busy={checkout.busy && checkout.lastIntent?.product === "lifetime"} onClick={() => checkout.purchase({ product: "lifetime" })}>{checkout.busy && checkout.lastIntent?.product === "lifetime" ? <PendingLabel>{t({ en: "Opening checkout…", zh: "正在打开结账…" })}</PendingLabel> : checkout.accountLoading ? t(billingText.loading) : hasLifetime ? t(billingText.lifetime) : t({ en: "Buy Lifetime", zh: "购买终身方案" })}{!hasLifetime && !checkout.accountLoading ? <span aria-hidden="true">→</span> : null}</button>
            {checkout.lastIntent?.product === "lifetime" ? <CheckoutFeedback failure={checkout.failure} /> : null}
          </article>
          <article className={styles.oneTimeCard}>
            <div className={styles.oneTimeHeading}>
              <div><span className={styles.sectionLabel}>{t({ en: "On-demand usage", zh: "按需补充" })}</span><h3>{t({ en: "Credit package", zh: "积分包" })}</h3></div>
              <p className={styles.oneTimePrice}><span>US$</span><strong>4.90</strong><small>{t({ en: "per package", zh: "每包" })}</small></p>
            </div>
            <p>{t({ en: "Add credits for use inside the product. Select the number of packages, then review your order in Stripe.", zh: "补充产品内消费所需的积分，选择购买份数后直接在 Stripe 核对订单。" })}</p>
            <label className={styles.quantityField}>{t({ en: "Packages", zh: "购买份数" })}<input type="number" min="1" max="100" step="1" value={quantity} disabled={checkout.busy} onChange={event => setQuantity(Number(event.target.value))} /></label>
            <button className={`button ${styles.oneTimeButton}`} type="button" disabled={checkout.busy || !Number.isInteger(quantity) || quantity < 1 || quantity > 100} aria-busy={checkout.busy && checkout.lastIntent?.product === "credits"} onClick={() => checkout.purchase({ product: "credits" }, quantity)}>{checkout.busy && checkout.lastIntent?.product === "credits" ? <PendingLabel>{t({ en: "Opening checkout…", zh: "正在打开结账…" })}</PendingLabel> : t({ en: "Buy credits", zh: "购买积分包" })}<span aria-hidden="true">→</span></button>
            {checkout.lastIntent?.product === "credits" ? <CheckoutFeedback failure={checkout.failure} /> : null}
          </article>
        </div>
        <p className={styles.purchaseNote}>{t({ en: "All amounts shown are examples. Available products, entitlements, and final prices depend on the configured catalog.", zh: "以上金额均为示例。可购买的商品、具体权益和最终价格，以实际配置的商品为准。" })}</p>
      </PageSection>
      <section className={styles.paymentTerms} aria-labelledby="payment-notes">
        <div className={styles.paymentTermsHeading}><svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 3 4 6v6c0 4 8 9 8 9s8-5 8-9V6z" /><path d="m8 12 3 3 5-6" /></svg><h2 id="payment-notes">{t({ en: "Before you pay", zh: "付款前请了解" })}</h2></div>
        <ul>
          <li>{t({ en: "Payment takes place on Stripe. Review the final amount, currency and any applicable taxes before confirming.", zh: "付款在 Stripe 安全页面完成。确认前请核对最终金额、币种及适用税费。" })}</li>
          <li>{t({ en: "Subscriptions renew automatically each month or year. Yearly subscriptions are charged once for the full year.", zh: "订阅按所选月付或年付周期自动续费；年付套餐一次收取全年费用。" })}</li>
          <li>{t({ en: "After subscribing, manage renewal, cancellation, payment methods and invoices in your billing page. Review the effective date before cancelling.", zh: "订阅后可在个人账单页管理续费、取消、付款方式及发票。取消前请核对页面显示的生效时间。" })}</li>
          <li>{t({ en: "Lifetime access and credit packages are separate one-time purchases. Trial arrangements, when offered, are stated at Checkout.", zh: "终身权益与积分包属于单独的一次性购买；若商品提供试用，以结账页展示的安排为准。" })}</li>
        </ul>
        <p><Link href="/terms">{t({ en: "Terms of service", zh: "服务条款" })}</Link><span aria-hidden="true"> · </span><Link href="/contact">{t({ en: "Payment questions? Contact us", zh: "付款有疑问？联系支持" })}</Link></p>
      </section>
      <SignInDialog open={Boolean(checkout.pending)} onClose={checkout.close} onSuccess={checkout.signedIn} returnTo={checkout.returnTo} />
      {checkout.changeRequest && checkout.token ? <SubscriptionChangeDialog token={checkout.token} subscription={checkout.changeRequest.subscription} offers={checkout.changeRequest.offers} selection={checkout.changeRequest.selection}
        onClose={() => checkout.setChangeRequest(null)} onChanged={checkout.revalidate} /> : null}
      <PageSection id="faq" title={t({ en: "A few things to know", zh: "选择前，了解这些" })}>
        <div className={styles.faqGrid}>{notes.map((note) => <article className={styles.faqItem} key={note.title.en}><h3>{t(note.title)}</h3><p>{t(note.body)}</p></article>)}</div>
      </PageSection>
    </SiteShell>
  );
}
