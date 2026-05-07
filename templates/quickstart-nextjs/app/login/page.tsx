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
    setStatus("Exchanging Google OAuth code for a session token...");

    exchangeToken(exchangeCode, readReferralCode())
      .then((session) => {
        if (cancelled) {
          return;
        }
        clearReferralCode();
        setReferralCode("");
        writeSession(session);
        setSessionToken(maskToken(session.token));
        setStatus("Google login succeeded. Local session is now stored in localStorage.");
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

  const googleCallbackExample = useMemo(() => `${appEnv.appUrl}/login`, []);

  return (
    <SiteShell
      eyebrow="Sign In"
      title="Use one auth entry page for passwordless email, Google OAuth, and referral-aware signup."
      description="This page still talks directly to the backend auth flows, but it is framed as a reusable sign-in surface for real products. It supports browser-side referral capture, email-code login, and Google redirect exchange in one place."
      sideTitle="What must align"
      sideBody={
        <DetailRows
          rows={[
            {
              label: "Frontend redirect",
              value: <span className="inline-code">{googleCallbackExample}</span>
            },
            {
              label: "Backend env",
              value: <span className="inline-code">APP_AUTH_FRONTEND_REDIRECT</span>
            },
            {
              label: "Google redirect URI",
              value: <span className="inline-code">APP_AUTH_GOOGLE_REDIRECT_URL</span>
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
        { id: "google-oauth", label: "Google OAuth" },
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

        <Panel className="span-5" title="Google OAuth" subtitle="Matches GET /auth/google/authorize and POST /auth/exchange-token.">
          <div id="google-oauth" />
          <p>
            The browser first asks the backend for a provider authorize URL. After Google redirects back to the backend callback, the backend redirects the browser to{" "}
            <span className="inline-code">{googleCallbackExample}</span>
            {" "}with a short-lived exchange code. This page then exchanges it together with the saved referral code, so Google signup and email-code signup both preserve referral attribution.
          </p>
          <p className="footer-note">
            If this button fails locally, the usual root cause is not the frontend code. It is almost always a mismatch between backend redirect env, Google Console callback URI, and the public backend URL.
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
