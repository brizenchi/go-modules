"use client";

import { Suspense, useEffect, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { SiteShell } from "@/components/site-shell";
import { Notice, Panel, DetailRows } from "@/components/ui";
import { ApiError } from "@/lib/api";
import {
  readReferralCode,
  readSession,
  SESSION_EVENT,
  writeReferralCode
} from "@/lib/auth";
import { formatDate, maskToken } from "@/lib/format";
import { exchangeOAuthCodeOnce } from "@/lib/oauth-exchange";
import { consumeOAuthCallbackError } from "@/lib/oauth-flow";
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
  const [exchangeAttempt, setExchangeAttempt] = useState(0);

  useEffect(() => {
    const inboundRef = searchParams.get("ref");
    if (inboundRef && !readSession()) {
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
		const oauthError = searchParams.get("oauth_error");
		const oauthFlowID = searchParams.get("flow") || "";
		if (oauthError) {
			setBusy("");
			setStatus("");
			setError(consumeOAuthCallbackError(oauthError, oauthFlowID));
			return;
		}

		const exchangeCode = searchParams.get("code");
		if (!exchangeCode) {
			return;
    }

    let cancelled = false;
    setBusy("exchange");
    setError("");
    setStatus("Exchanging OAuth code for a session token...");

    exchangeOAuthCodeOnce(exchangeCode, oauthFlowID)
      .then(({ session, committed }) => {
        if (cancelled) {
          return;
        }
        const currentSession = readSession();
        if (!committed || currentSession?.token !== session.token) {
          setSessionToken(maskToken(currentSession?.token));
          if (currentSession) {
            setStatus("A newer browser session is already active. The stale OAuth response was ignored.");
            router.replace("/dashboard");
          } else {
            setError("This OAuth exchange is no longer current. Start OAuth sign-in again.");
            setStatus("");
          }
          return;
        }
        setReferralCode("");
        setSessionToken(maskToken(session.token));
        setStatus("OAuth login succeeded. Local session is now stored in localStorage.");
        router.replace("/dashboard");
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
  }, [exchangeAttempt, router, searchParams]);

  return (
    <SiteShell
      eyebrow="Welcome back"
      title="One sign-in. Your whole workspace."
      description="Choose any available sign-in method. Your plan, account settings, credits, and referral rewards will be waiting in the workspace."
      sideTitle="Secure by default"
      sideBody={
        <DetailRows
          rows={[
            {
              label: "After sign-in",
              value: <span>Your workspace opens automatically</span>
            },
            {
              label: "OAuth",
              value: <span>Browser-bound and single-use</span>
            },
            {
              label: "Available methods",
              value: <span>Always matched to this workspace</span>
            },
            {
              label: "Referral code",
              value: <span>Preserved through first sign-up</span>
            }
          ]}
        />
      }
      toc={[
        { id: "email-login", label: "Sign in" },
        { id: "oauth", label: "OAuth" },
        { id: "session-view", label: "Session view" }
      ]}
    >
      <div className="page-grid">
        <Panel className="span-7" title="Sign in" subtitle="Use email, Google, or GitHub when enabled for this workspace.">
          <div id="email-login" />
          <SignInPanel
            showReferralField
            onSuccess={() => {
              setStatus("Signed in. Opening your workspace...");
              router.push("/dashboard");
            }}
          />
        </Panel>

        <Panel className="span-5" title="Continue securely" subtitle="OAuth opens the provider directly and returns you here when complete.">
          <div id="oauth" />
          <p>
            Your browser completes authorization directly with the selected provider. A short-lived, single-use handoff signs you in without exposing provider credentials to this page.
          </p>
          <p className="footer-note">
            If you cancel authorization, you will return here safely and can choose another sign-in method.
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

        {status ? (
          <div className="span-12">
            {busy === "exchange" ? <Notice>{status}</Notice> : <Notice tone="success">{status}</Notice>}
          </div>
        ) : null}
        {error ? (
          <div className="span-12">
            <Notice tone="error">
              {error}
              {searchParams.get("code") ? (
                <span className="button-row">
                  <button
                    className="button"
                    type="button"
                    disabled={busy === "exchange"}
                    onClick={() => setExchangeAttempt((attempt) => attempt + 1)}
                  >
                    Retry OAuth exchange
                  </button>
                </span>
              ) : null}
            </Notice>
          </div>
        ) : null}
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
