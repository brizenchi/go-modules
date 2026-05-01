import Link from "next/link";
import { SiteShell } from "@/components/site-shell";
import { CTAButton, DetailRows, PageSection, Panel } from "@/components/ui";
import { FeatureCard, MetricCard } from "@/components/marketing";
import { appEnv } from "@/lib/env";

export default function HomePage() {
  return (
    <SiteShell
      eyebrow="Universal SaaS Shell"
      title="A reusable frontend template for product marketing, docs, pricing, auth, billing, and referral flows."
      description="This template no longer behaves like a single-purpose test harness. It is a starter shell for real SaaS products: a public-facing home, navigation system, docs experience, pricing page, multilingual-ready layout, and authenticated account surfaces all wired to the shared go-modules backend contract."
      sideTitle="Contract map"
      sideBody={
        <DetailRows
          rows={[
            { label: "Frontend origin", value: <span className="inline-code">{appEnv.appUrl}</span> },
            { label: "Backend API", value: <span className="inline-code">{appEnv.apiBaseUrl}</span> },
            { label: "Login landing", value: <span className="inline-code">{`${appEnv.appUrl}/login`}</span> },
            { label: "Referral invite", value: <span className="inline-code">{`${appEnv.appUrl}/invite?ref=`}</span> }
          ]}
        />
      }
      actions={
        <>
          <CTAButton href="/pricing" primary>
            Open Pricing
          </CTAButton>
          <CTAButton href="/docs">Read Docs</CTAButton>
        </>
      }
      toc={[
        { id: "shell", label: "Shell capabilities" },
        { id: "routes", label: "Route map" },
        { id: "ops", label: "Operational contract" }
      ]}
    >
      <PageSection
        id="shell"
        title="What this template is now"
        description="It gives you a public site shell and an authenticated product shell in the same starter."
      >
        <div className="feature-grid">
          <FeatureCard
            label="Navigation"
            title="Public and product entry points"
            description="A top navigation system that can carry marketing pages, documentation, pricing, and authenticated account flows without rewriting the layout every time."
          />
          <FeatureCard
            label="Account UX"
            title="Avatar menu and management surfaces"
            description="Signed-in users get an account menu with settings, sign-out, subscription access, and referral access instead of a single debug session pill."
          />
          <FeatureCard
            label="Documentation"
            title="Breadcrumbs and article navigation"
            description="Docs-style pages can expose article sections with table-of-contents navigation so the template works for launch docs and product education pages."
          />
          <FeatureCard
            label="Localization"
            title="Language-ready structure"
            description="The shell carries an EN / 中文 switch and a basic i18n layer, making multilingual product sites possible without redesigning the whole frame."
          />
        </div>
      </PageSection>

      <PageSection
        id="routes"
        title="Route map"
        description="These routes are intended as a reusable baseline, not one-off demos."
      >
        <div className="page-grid">
          <Panel className="span-4" title="/pricing" subtitle="Plan comparison and checkout entry">
            <p>Use for subscription entry, upgrade framing, credits explanation, and plan differentiation before redirecting into Checkout.</p>
          </Panel>
          <Panel className="span-4" title="/docs" subtitle="Onboarding and product documentation">
            <p>Use for integration docs, getting-started flows, technical setup guides, and feature explanation pages with table-of-contents support.</p>
          </Panel>
          <Panel className="span-4" title="/login" subtitle="Auth entry surface">
            <p>Still supports email code, Google OAuth exchange, and referral-aware signup, but now sits inside a broader product shell.</p>
          </Panel>
          <Panel className="span-4" title="/account" subtitle="Settings and session management">
            <p>Identity, refresh, logout, current token/session view, and ticket issue flow. This is the base for a settings or workspace page.</p>
          </Panel>
          <Panel className="span-4" title="/billing" subtitle="Subscription and invoice management">
            <p>Checkout creation, subscription lifecycle, invoice list, cancel, and reactivate. This doubles as the management page behind the avatar menu.</p>
          </Panel>
          <Panel className="span-4" title="/referrals" subtitle="Referral center">
            <p>Share link, stats, history, and the consumer-facing part of the referral system, suitable for a growth or rewards area.</p>
          </Panel>
        </div>
      </PageSection>

      <PageSection
        id="ops"
        title="Operational contract"
        description="Keep the shell generic, keep the backend contract stable."
      >
        <div className="metric-grid">
          <MetricCard
            label="Auth"
            value="Email + Google"
            detail="Uses /auth/send-code, /auth/verify-code, /auth/google/authorize, /auth/exchange-token, /auth/refresh, and /auth/logout."
          />
          <MetricCard
            label="Billing"
            value="Checkout + invoices"
            detail="Creates Stripe sessions in the browser but keeps webhook truth and subscription state on the backend."
          />
          <MetricCard
            label="Referral"
            value="Closed loop"
            detail="Captures ?ref= in the browser, forwards referral_code during signup, then reads stats and history from backend APIs."
          />
          <MetricCard
            label="Docs"
            value="Product-ready"
            detail="Supports docs pages, breadcrumbs, and in-page article navigation so the starter can front a real product site."
          />
        </div>

        <div className="cta-strip">
          <Link className="button primary" href="/billing">
            Open Billing Console
          </Link>
          <Link className="button" href="/referrals">
            Open Referral Center
          </Link>
        </div>
      </PageSection>
    </SiteShell>
  );
}
