"use client";

import Link from "next/link";
import { Suspense, useEffect, useState } from "react";
import { useSearchParams } from "next/navigation";
import { SiteShell } from "@/components/site-shell";
import { Notice, Panel, DetailRows } from "@/components/ui";
import { readReferralCode, writeReferralCode } from "@/lib/auth";

function InvitePageInner() {
  const searchParams = useSearchParams();
  const [savedCode, setSavedCode] = useState("");

  useEffect(() => {
    const ref = searchParams.get("ref");
    if (ref) {
      setSavedCode(writeReferralCode(ref));
      return;
    }
    setSavedCode(readReferralCode());
  }, [searchParams]);

  return (
    <SiteShell
      eyebrow="Invite Landing"
      title="Capture inbound referral codes before the visitor signs in."
      description="This page is intentionally simple: preserve the invitation code, explain the next step, and move the visitor into your auth flow. In a real product you can redesign the presentation, but the capture behavior should remain stable."
      sideTitle="What happens here"
      sideBody={
        <DetailRows
          rows={[
            { label: "Input query", value: <span className="inline-code">/invite?ref=INV123456</span> },
            { label: "Stored locally", value: <span className="inline-code">{savedCode || "-"}</span> },
            { label: "Next step", value: <span>Use the top-right sign-in dialog</span> }
          ]}
        />
      }
      toc={[
        { id: "invite-flow", label: "Invite flow" }
      ]}
      actions={
        <Link className="button primary" href="/">
          Continue To Home
        </Link>
      }
    >
      <div className="page-grid">
        <Panel className="span-12" title="Referral landing behavior" subtitle="Useful for local testing and for public-domain invite links.">
          <div id="invite-flow" />
          {savedCode ? (
            <Notice tone="success">
              Referral code <span className="inline-code">{savedCode}</span> has been stored in the browser.
            </Notice>
          ) : (
            <Notice>
              No referral code is present in the URL. Open this page with <span className="inline-code">?ref=YOURCODE</span> to test the flow.
            </Notice>
          )}
          <p>
            This page captures the code and preserves it until signup finishes. In the matching quickstart backend, that stored value is forwarded during email-code and Google signup so referral attribution can complete automatically.
          </p>
        </Panel>
      </div>
    </SiteShell>
  );
}

export default function InvitePage() {
  return (
    <Suspense fallback={null}>
      <InvitePageInner />
    </Suspense>
  );
}
