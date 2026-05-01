import { SiteShell } from "@/components/site-shell";
import { DetailRows } from "@/components/ui";
import { DocArticle } from "@/components/marketing";

const toc = [
  { id: "overview", label: "Overview" },
  { id: "routing", label: "Routing model" },
  { id: "account", label: "Account surfaces" },
  { id: "integration", label: "Integration notes" }
];

export default function DocsPage() {
  return (
    <SiteShell
      eyebrow="Documentation"
      title="A docs-ready page template with breadcrumbs and article navigation."
      description="A reusable SaaS frontend template usually needs documentation just as much as it needs login and billing. This page shows how the shell can host product docs, onboarding guides, or integration notes without becoming a separate site."
      sideTitle="Docs shell"
      sideBody={
        <DetailRows
          rows={[
            { label: "Layout mode", value: <span className="inline-code">Article + TOC</span> },
            { label: "Breadcrumbs", value: <span className="inline-code">Enabled by shell</span> },
            { label: "Language switch", value: <span className="inline-code">Top navigation</span> }
          ]}
        />
      }
      toc={toc}
    >
      <div className="doc-layout">
        <DocArticle
          id="overview"
          title="Overview"
          description="This docs page is meant to be copied and replaced with product documentation, not kept as-is."
        >
          <p>
            The old quickstart frontend focused on direct backend flow verification. That remains important, but many real SaaS projects also need a public-facing docs layer for setup, onboarding, and technical explanation. The shell now supports that use case directly.
          </p>
        </DocArticle>

        <DocArticle
          id="routing"
          title="Routing model"
          description="Separate public and authenticated responsibilities without splitting the app shell."
        >
          <p>
            Recommended route split: public routes such as `/`, `/pricing`, `/docs`, and `/login`; authenticated routes such as `/account`, `/billing`, and `/referrals`. The top shell stays consistent so the user does not feel thrown between unrelated interfaces.
          </p>
        </DocArticle>

        <DocArticle
          id="account"
          title="Account surfaces"
          description="The avatar menu is the entry point into settings, billing, and referral management."
        >
          <p>
            Instead of a plain “signed in” pill, the template now treats the authenticated surface as a product area. The avatar menu should be the stable user entry point for settings, sign-out, subscription visibility, and referral visibility. Product teams can extend this with workspace switching, security settings, or profile preferences.
          </p>
        </DocArticle>

        <DocArticle
          id="integration"
          title="Integration notes"
          description="Keep backend contracts stable and frontend routes expressive."
        >
          <p>
            The frontend remains a consumer of the shared backend contract. That means JSON envelope parsing, auth session management, Stripe Checkout session creation, and referral read APIs should remain consistent. The shell and design can evolve, but the integration contract should stay boring and predictable.
          </p>
        </DocArticle>
      </div>
    </SiteShell>
  );
}
