import { SiteShell } from "@/components/site-shell";
import { DetailRows, PageSection } from "@/components/ui";
import { PricingCard } from "@/components/marketing";
import { appEnv } from "@/lib/env";

export default function PricingPage() {
  return (
    <SiteShell
      eyebrow="Pricing"
      title="A pricing page template that works for both marketing and authenticated upgrade flows."
      description="This page is intentionally more generic than the old billing demo. It can live in front of the paywall, compare plans, describe credits, and route the user into the shared Stripe checkout flow when they are ready."
      sideTitle="Pricing contract"
      sideBody={
        <DetailRows
          rows={[
            { label: "Default plan", value: <span className="inline-code">{appEnv.defaultPlan}</span> },
            { label: "Default interval", value: <span className="inline-code">{appEnv.defaultInterval}</span> },
            { label: "Credits quantity", value: <span className="inline-code">{String(appEnv.defaultCreditsQuantity)}</span> },
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
        description="Replace plan names and copy, but keep the route and flow structure."
      >
        <div className="pricing-grid">
          <PricingCard
            tier="Starter"
            price="$19"
            subtitle="For teams validating the product shell and basic account flows."
            features={[
              "Core auth and account UI",
              "Shared billing integration",
              "Referral capture and share links",
              "Docs and onboarding shell"
            ]}
            href="/billing"
            cta="Choose Starter"
          />
          <PricingCard
            tier="Pro"
            price="$49"
            subtitle="For production teams that want the full shared SaaS surface."
            features={[
              "Subscription lifecycle console",
              "Invoices and payment method visibility",
              "Referral stats and history",
              "Multilingual-ready shell"
            ]}
            href="/billing"
            cta="Choose Pro"
            featured
          />
          <PricingCard
            tier="Credits"
            price="Usage-based"
            subtitle="For top-ups and product flows that mix subscription and credits."
            features={[
              "Separate credits Checkout path",
              "Credit package quantity support",
              "Suitable for AI, workflow, or seat-like usage",
              "Can coexist with subscription plans"
            ]}
            href="/billing"
            cta="Buy Credits"
          />
        </div>
      </PageSection>

      <PageSection
        id="faq"
        title="Pricing notes"
        description="These notes explain the template boundary, not your final commercial policy."
      >
        <div className="feature-grid">
          <article className="feature-card">
            <span className="panel-kicker">Stripe split</span>
            <h3>Marketing page here, payment truth there</h3>
            <p>The pricing page lives in the frontend, but all checkout creation, webhook truth, and subscription state remain backend-owned.</p>
          </article>
          <article className="feature-card">
            <span className="panel-kicker">Plan copy</span>
            <h3>Replace names, not route structure</h3>
            <p>You should absolutely replace Starter / Pro / Credits copy, but it is useful to keep `/pricing` and `/billing` as stable product routes.</p>
          </article>
          <article className="feature-card">
            <span className="panel-kicker">Referral-aware checkout</span>
            <h3>Growth path already preserved</h3>
            <p>If a referral code was captured before signup, the billing page can still forward it as metadata when checkout is created.</p>
          </article>
        </div>
      </PageSection>
    </SiteShell>
  );
}
