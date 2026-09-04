import Link from "next/link";
import { SiteShell } from "@/components/site-shell";
import { CTAButton, DetailRows, PageSection, Panel } from "@/components/ui";
import { FeatureCard, MetricCard } from "@/components/marketing";
import { ProductPreview } from "@/components/product-preview";
import { appEnv } from "@/lib/env";

export default function HomePage() {
  return (
    <SiteShell
      eyebrow="Production-ready SaaS starter"
      title="The foundation your next SaaS should not have to rebuild."
      description="Launch with polished marketing, authentication, subscriptions, credits, and referrals already connected to a dependable Go backend—then make the product yours."
      sideTitle="Ready on day one"
      sideBody={
        <DetailRows
          rows={[
            { label: "Frontend", value: <span className="inline-code">{appEnv.appUrl}</span> },
            { label: "API contract", value: <span className="inline-code">{appEnv.apiBaseUrl}</span> },
            { label: "Auth", value: "Email + OAuth" },
            { label: "Monetization", value: "Plans + credits" }
          ]}
        />
      }
      actions={
        <>
          <CTAButton href="/pricing" primary>
            Explore pricing
          </CTAButton>
          <CTAButton href="/docs">Read the docs</CTAButton>
        </>
      }
      toc={[
        { id: "capabilities", label: "What is included" },
        { id: "routes", label: "Product surfaces" },
        { id: "contract", label: "Backend contract" }
      ]}
    >
      <ProductPreview />

      <div className="capability-rail" aria-label="Included platform capabilities">
        <span>Authentication</span>
        <span>Billing</span>
        <span>Credits</span>
        <span>Referrals</span>
        <span>Documentation</span>
      </div>

      <PageSection
        id="capabilities"
        title="The parts every SaaS rebuilds. Already done."
        description="A focused baseline for the work surrounding your actual product—designed to be replaced, extended, and shipped."
      >
        <div className="feature-grid">
          <FeatureCard
            label="01 · Identity"
            title="Authentication that feels native"
            description="Email code and OAuth flows, session refresh, sign-out, and referral-aware signup inside one coherent account experience."
          />
          <FeatureCard
            label="02 · Revenue"
            title="More than one way to monetize"
            description="Recurring plans, lifetime access, fixed credit packages, and custom top-ups share a reliable Stripe-backed contract."
          />
          <FeatureCard
            label="03 · Growth"
            title="Referrals built into the journey"
            description="Capture invite codes before signup, attribute conversions, and give customers a clear place to understand their rewards."
          />
          <FeatureCard
            label="04 · Experience"
            title="Public site and product shell"
            description="Marketing, docs, pricing, and authenticated tools use the same responsive system with multilingual structure included."
          />
        </div>
      </PageSection>

      <PageSection
        id="routes"
        title="A complete customer journey, not a disconnected demo."
        description="Each surface has a clear role, while shared navigation and account state keep the experience continuous."
      >
        <div className="page-grid">
          <Panel className="span-4 route-panel" title="Pricing" subtitle="Discover and compare">
            <p>Present subscriptions, lifetime access, and credit packages with direct paths into checkout.</p>
            <Link className="text-link" href="/pricing">View pricing <span aria-hidden="true">↗</span></Link>
          </Panel>
          <Panel className="span-4 route-panel" title="Documentation" subtitle="Understand and integrate">
            <p>Publish onboarding, technical guides, and product education with breadcrumbs and article navigation.</p>
            <Link className="text-link" href="/docs">Browse docs <span aria-hidden="true">↗</span></Link>
          </Panel>
          <Panel className="span-4 route-panel" title="Authentication" subtitle="Enter the product">
            <p>Give every user a clean email or OAuth entry point with referral attribution carried through signup.</p>
            <Link className="text-link" href="/login">Open sign in <span aria-hidden="true">↗</span></Link>
          </Panel>
          <Panel className="span-4 route-panel" title="Account" subtitle="Control identity and sessions">
            <p>Manage identity, refresh state, sign-out, token visibility, and WebSocket ticket issuance.</p>
            <Link className="text-link" href="/account">View account <span aria-hidden="true">↗</span></Link>
          </Panel>
          <Panel className="span-4 route-panel" title="Billing" subtitle="Manage the lifecycle">
            <p>Start checkout, inspect invoices, change plan state, buy credits, and manage entitlements.</p>
            <Link className="text-link" href="/billing">Open billing <span aria-hidden="true">↗</span></Link>
          </Panel>
          <Panel className="span-4 route-panel" title="Referrals" subtitle="Turn customers into growth">
            <p>Share links, conversion stats, reward history, and attribution in a customer-facing center.</p>
            <Link className="text-link" href="/referrals">View referrals <span aria-hidden="true">↗</span></Link>
          </Panel>
        </div>
      </PageSection>

      <PageSection
        id="contract"
        title="A quiet frontend. A serious backend contract."
        description="The design stays easy to reshape because the operational boundaries remain explicit and predictable."
      >
        <div className="metric-grid">
          <MetricCard
            label="Auth"
            value="Email + Google"
            detail="Provider authorization, token exchange, refresh, and account state are already connected."
          />
          <MetricCard
            label="Billing"
            value="6 billing lanes"
            detail="Plans, buyout, packages, and top-ups all preserve webhook-owned payment truth."
          />
          <MetricCard
            label="Referral"
            value="Closed loop"
            detail="Browser capture, signup attribution, activation state, statistics, and history stay connected."
          />
          <MetricCard
            label="Docs"
            value="Product-ready"
            detail="Breadcrumbs, article navigation, language state, and public routing are part of the shell."
          />
        </div>

        <div className="cta-strip">
          <div>
            <span className="panel-kicker">Start with the hard parts solved</span>
            <strong>Make the next commit about your product.</strong>
          </div>
          <div className="cta-strip-actions">
            <Link className="button primary" href="/billing">Open billing</Link>
            <Link className="button" href="/docs">Read the docs</Link>
          </div>
        </div>
      </PageSection>
    </SiteShell>
  );
}
