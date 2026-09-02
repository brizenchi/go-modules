"use client";

import { Suspense, useEffect, useMemo, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { SiteShell } from "@/components/site-shell";
import { Notice, Panel, DetailRows } from "@/components/ui";
import {
  ApiError,
  exchangeToken,
} from "@/lib/api";
import {
  clearReferralCode,
  readReferralCode,
  readSession,
  SESSION_EVENT,
  writeReferralCode,
  writeSession
} from "@/lib/auth";
import { appEnv } from "@/lib/env";
import { formatDate, maskToken } from "@/lib/format";
import { SignInPanel } from "@/components/sign-in-panel";

function messageFromError(error: unknown): string {
  if (error instanceof ApiError) {
    return error.message;
  }
  if (error instanceof Error) {
    return error.message;
  }
  return "unexpected error";
}

function LoginPageInner() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [referralCode, setReferralCode] = useState("");
  const [status, setStatus] = useState<string>("");
  const [error, setError] = useState<string>("");
  const [busy, setBusy] = useState<"" | "exchange">("");
  const [sessionToken, setSessionToken] = useState("-");

  useEffect(() => {
    const inboundRef = searchParams.get("ref");
    if (inboundRef) {
      const saved = writeReferralCode(inboundRef);
      setReferralCode(saved);
      setStatus(`Captured referral code ${saved}. This template will now send it during first signup and OAuth exchange so backend attribution can complete automatically.`);
    } else {
      setReferralCode(readReferralCode());
    }

    const session = readSession();
    setSessionToken(maskToken(session?.token));

    const sync = () => {
      const current = readSession();
      setSessionToken(maskToken(current?.token));
    };
    window.addEventListener(SESSION_EVENT, sync);
    return () => window.removeEventListener(SESSION_EVENT, sync);
  }, [searchParams]);

  useEffect(() => {
    const exchangeCode = searchParams.get("code");
    if (!exchangeCode) {
      return;
    }

    let cancelled = false;
    setBusy("exchange");
    setError("");
    setStatus("Exchanging OAuth code for a session token...");

    exchangeToken(exchangeCode, readReferralCode())
      .then((session) => {
        if (cancelled) {
          return;
        }
        clearReferralCode();
        setReferralCode("");
        writeSession(session);
        setSessionToken(maskToken(session.token));
        setStatus("OAuth login succeeded. Local session is now stored in localStorage.");
        router.replace("/account");
      })
      .catch((err) => {
        if (cancelled) {
          return;
        }
        setError(messageFromError(err));
        setStatus("");
      })
      .finally(() => {
        if (!cancelled) {
          setBusy("");
        }
      });

    return () => {
      cancelled = true;
    };
  }, [router, searchParams]);

  const callbackExample = useMemo(() => `${appEnv.appUrl}/login`, []);
  const oauthProviderNames = appEnv.authOAuthProviders
    .map((provider) => provider === "github" ? "GitHub" : "Google")
    .join(" / ");

  return (
    <SiteShell
      eyebrow="Sign In"
      title="Use one auth entry page for enabled email and OAuth providers."
      description="The visible login methods come from frontend env and should match the backend module configuration. Referral capture works for email, Google, and GitHub signup."
      sideTitle="What must align"
      sideBody={
        <DetailRows
          rows={[
            {
              label: "Frontend redirect",
              value: <span className="inline-code">{callbackExample}</span>
            },
            {
              label: "Backend env",
              value: <span className="inline-code">APP_AUTH_FRONTEND_REDIRECT</span>
            },
            {
              label: "Enabled OAuth",
              value: <span className="inline-code">{oauthProviderNames || "none"}</span>
            },
            {
              label: "Local token store",
              value: <span className="inline-code">localStorage</span>
            }
          ]}
        />
      }
      toc={[
        { id: "email-login", label: "Email login" },
        { id: "oauth", label: "OAuth" },
        { id: "session-view", label: "Session view" }
      ]}
    >
      <div className="page-grid">
        <Panel className="span-7" title="Email-code login" subtitle="Matches POST /auth/send-code and POST /auth/verify-code.">
          <div id="email-login" />
          <SignInPanel
            showReferralField
            onSuccess={() => {
              setStatus("Email-code login succeeded. Session saved in localStorage.");
              router.push("/account");
            }}
          />
        </Panel>

        <Panel className="span-5" title="OAuth" subtitle="Matches GET /auth/:provider/authorize and POST /auth/exchange-token.">
          <div id="oauth" />
          <p>
            The browser first asks the backend for the selected provider&apos;s authorize URL. After the provider callback, the backend redirects the browser to{" "}
            <span className="inline-code">{callbackExample}</span>
            {" "}with a short-lived exchange code. This page exchanges it together with the saved referral code, so every enabled signup method preserves referral attribution.
          </p>
          <p className="footer-note">
            If an OAuth button fails, first compare the backend redirect env, the provider console callback URI, and the public backend URL.
          </p>
        </Panel>

        <Panel className="span-6" title="Current browser session" subtitle="What this page sees in localStorage right now.">
          <div id="session-view" />
          <div className="details-list">
            <div className="details-row">
              <strong>Token</strong>
              <span className="inline-code">{sessionToken}</span>
            </div>
            <div className="details-row">
              <strong>Saved referral code</strong>
              <span className="inline-code">{referralCode || "-"}</span>
            </div>
            <div className="details-row">
              <strong>Next step after login</strong>
              <span>Open the account page to test refresh, logout, and WebSocket ticket issuing.</span>
            </div>
          </div>
        </Panel>

        <Panel className="span-6" title="Referral behavior" subtitle="The quickstart templates now close the loop end-to-end.">
          <p>
            This frontend stores inbound <span className="inline-code">?ref=CODE</span> in local storage and sends it when signup is finalized. The backend quickstart consumes that value on new-user creation, records the referral automatically, and later activates the reward when Stripe subscription activation arrives.
          </p>
        </Panel>
      </div>
    </SiteShell>
  );
}

export default function LoginPage() {
  return (
    <Suspense fallback={null}>
      <LoginPageInner />
    </Suspense>
  );
}
