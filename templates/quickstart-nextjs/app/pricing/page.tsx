import { SiteShell } from "@/components/site-shell";
import { DetailRows, PageSection } from "@/components/ui";
import { PricingCard } from "@/components/marketing";
import { appEnv } from "@/lib/env";

export default function PricingPage() {
  return (
    <SiteShell
      eyebrow="Pricing"
      title="Commercial pricing template with six explicit billing lanes."
      description="This page keeps subscriptions, buyout, fixed packages, and flexible top-ups clearly separated. Exact money amounts should come from your actual Stripe catalog and business policy, not from fake hardcoded template prices."
      sideTitle="Pricing contract"
      sideBody={
        <DetailRows
          rows={[
            { label: "Default plan", value: <span className="inline-code">{appEnv.defaultPlan}</span> },
            { label: "Default interval", value: <span className="inline-code">{appEnv.defaultInterval}</span> },
            { label: "Credits quantity", value: <span className="inline-code">{String(appEnv.defaultCreditsQuantity)}</span> },
            { label: "Default top-up USD", value: <span className="inline-code">{String(appEnv.defaultTopUpAmountUSD)}</span> },
            { label: "Price source", value: <span className="inline-code">Stripe products + business policy</span> },
            { label: "Checkout UI", value: <span className="inline-code">/billing</span> }
          ]}
        />
      }
      toc={[
        { id: "plans", label: "Plans" },
        { id: "faq", label: "Pricing notes" }
      ]}
    >
      <PageSection
        id="plans"
        title="Plans"
        description="Use this structure as the public-facing commercial model: three recurring tiers, one buyout tier, one fixed package lane, and one flexible top-up lane."
      >
        <div className="pricing-grid">
          <PricingCard
            tier="Starter"
            price="US$4 / month"
            subtitle="Entry subscription for teams validating the product shell before full production rollout."
            features={[
              "Yearly: US$40 / year",
              "Starter yearly includes a 3-day trial",
              "Core auth and account experience",
              "Best fit for early product launch"
            ]}
            href="/billing"
            cta="Choose Starter"
          />
          <PricingCard
            tier="Pro"
            price="US$14 / month"
            subtitle="Primary production subscription for teams turning the shared stack into a revenue product."
            features={[
              "Yearly: US$140 / year",
              "Subscription lifecycle console",
              "Invoices and payment method visibility",
              "Operational baseline for real SaaS launch"
            ]}
            href="/billing"
            cta="Choose Pro"
          />
          <PricingCard
            tier="Premium"
            price="US$59 / month"
            subtitle="Top recurring plan for larger customers who need the highest-value subscription without a buyout."
            features={[
              "Yearly: US$590 / year",
              "Top recurring subscription tier",
              "Clear upgrade path beyond Pro",
              "Can still coexist with credits flows"
            ]}
            href="/billing"
            cta="Choose Premium"
            featured
          />
          <PricingCard
            tier="Lifetime"
            price="US$999"
            subtitle="Permanent access purchase for founder deals, internal tools, launch campaigns, or demo buyout offers."
            features={[
              "One-time buyout tier",
              "No recurring billing cycle",
              "Maps to lifetime entitlement in backend",
              "Stays separate from recurring plan changes"
            ]}
            href="/billing"
            cta="Buy Lifetime"
          />
          <PricingCard
            tier="Package"
            price="US$4.90"
            subtitle="Preset credits bundles backed by a Stripe Price ID and hosted Checkout."
            features={[
              "Fixed credits package",
              "Stripe Checkout with price_id",
              "Quantity multiplier supported",
              "Good for predefined bundle merchandising"
            ]}
            href="/billing"
            cta="Buy Package"
          />
          <PricingCard
            tier="Custom Amount"
            price="Any amount"
            subtitle="自由充值数额: the customer enters the amount, then the backend creates a one-off PaymentIntent."
            features={[
              "No Stripe Price ID required",
              "Stripe Payment Element flow",
              "Useful when recharge amount is not predetermined",
              "Webhook converts paid amount into credits"
            ]}
            href="/billing"
            cta="Top Up Any Amount"
          />
        </div>
      </PageSection>

      <PageSection
        id="faq"
        title="Pricing notes"
        description="These notes explain how to keep the page commercially correct instead of visually polished but operationally false."
      >
        <div className="feature-grid">
          <article className="feature-card">
            <span className="panel-kicker">No fake numbers</span>
            <h3>Do not hardcode prices you cannot guarantee</h3>
            <p>This page now reflects the current commercial amounts you provided, but the operational source of truth still remains your Stripe catalog and backend checkout configuration.</p>
          </article>
          <article className="feature-card">
            <span className="panel-kicker">Six lanes</span>
            <h3>Subscriptions and credits should be named separately</h3>
            <p>Starter, Pro, Premium, Lifetime, Package, and Custom Amount each represent a different commercial path. Do not collapse package credits and flexible recharge into one generic credits card.</p>
          </article>
          <article className="feature-card">
            <span className="panel-kicker">Package vs custom</span>
            <h3>Fixed package and free amount are different Stripe flows</h3>
            <p>Package credits still use hosted Checkout plus a fixed Price ID. Custom amount top-up uses PaymentIntent plus Payment Element, because the amount is created dynamically at runtime.</p>
          </article>
          <article className="feature-card">
            <span className="panel-kicker">Backend truth</span>
            <h3>Marketing copy lives here, payment truth lives there</h3>
            <p>If a referral code was captured before signup, the billing page can still forward it as metadata when checkout is created for subscriptions, lifetime, package credits, or custom amount top-up. Checkout creation and webhook truth still stay backend-owned.</p>
          </article>
        </div>
      </PageSection>
    </SiteShell>
  );
}
