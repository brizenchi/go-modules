"use client";

import { useEffect, useRef, useState } from "react";
import { ResourceFailure } from "@/components/resource-feedback";
import { Notice } from "@/components/ui";
import {
  ApiError,
  getCapabilities,
  sendCode,
  verifyCode,
  type OAuthProvider
} from "@/lib/api";
import { getOAuthRedirectURL } from "@/lib/oauth-flow";
import { resolveAuthMethods } from "@/lib/auth-methods";
import {
  clearReferralCode,
  readReferralCode,
  readSession,
  REFERRAL_EVENT,
  writeReferralCode,
  writeSession,
  type AuthSession
} from "@/lib/auth";
import { useI18n } from "@/lib/i18n";
import { appEnv } from "@/lib/env";
import {
  beginRequestGeneration,
  invalidateRequestGeneration,
  isCurrentRequestGeneration
} from "@/lib/request-generation";
import {
  idleResource,
  loadingResource,
  settleResource,
  type ResourceState
} from "@/lib/request-state";

function messageFromError(error: unknown): string {
  if (error instanceof ApiError) {
    return error.message;
  }
  if (error instanceof Error) {
    return error.message;
  }
  return "unexpected error";
}

type SignInPanelProps = {
  compact?: boolean;
  showReferralField?: boolean;
  // OAuth-only destination; email login remains controlled by onSuccess.
  returnTo?: string;
  onSuccess?: (session: AuthSession) => void;
};

export function SignInPanel({
  compact = false,
  showReferralField = false,
  returnTo,
  onSuccess
}: SignInPanelProps) {
  const { t } = useI18n();
  const [email, setEmail] = useState("");
  const [code, setCode] = useState("");
  const [referralCode, setReferralCode] = useState("");
  const [sendResult, setSendResult] = useState<{
    email: string;
    expiresAt: string;
    debugCode?: string;
  } | null>(null);
  const [status, setStatus] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState("");
  const [capabilitiesState, setCapabilitiesState] = useState<ResourceState<Awaited<ReturnType<typeof getCapabilities>>>>(idleResource());
  const mountedRef = useRef(false);
  const capabilitiesGenerationRef = useRef(0);
  const emailActionGenerationRef = useRef(0);
  const codeIssued = !!sendResult;
  const authMethods = capabilitiesState.status === "ready"
    ? resolveAuthMethods(capabilitiesState.data, {
        emailConfigured: appEnv.authEmailConfigured,
        emailEnabled: appEnv.authEmailEnabled,
        oauthProvidersConfigured: appEnv.authOAuthProvidersConfigured,
        oauthProviders: appEnv.authOAuthProviders
      })
    : null;
  const emailEnabled = authMethods?.emailEnabled ?? false;
  const oauthProviders = authMethods?.oauthProviders ?? [];

  useEffect(() => {
    mountedRef.current = true;
    const syncReferral = () => setReferralCode(readReferralCode());
    syncReferral();
    void loadCapabilities();
    window.addEventListener(REFERRAL_EVENT, syncReferral);
    window.addEventListener("storage", syncReferral);

    return () => {
      window.removeEventListener(REFERRAL_EVENT, syncReferral);
      window.removeEventListener("storage", syncReferral);
      mountedRef.current = false;
      invalidateRequestGeneration(capabilitiesGenerationRef);
      invalidateRequestGeneration(emailActionGenerationRef);
    };
  }, []);

  async function loadCapabilities() {
    if (!mountedRef.current) return;
    const generation = beginRequestGeneration(capabilitiesGenerationRef);
    setCapabilitiesState((current) => loadingResource(current.data));
    const nextState = await settleResource(getCapabilities(), "Sign-in methods");
    if (
      mountedRef.current
      && isCurrentRequestGeneration(capabilitiesGenerationRef, generation)
    ) {
      setCapabilitiesState(nextState);
    }
  }

  function isLatestMountedEmailAction(generation: number): boolean {
    return mountedRef.current
      && isCurrentRequestGeneration(emailActionGenerationRef, generation);
  }

  function canCommitEmailAction(
    generation: number,
    initialSessionToken: string
  ): boolean {
    return isLatestMountedEmailAction(generation)
      && (readSession()?.token || "") === initialSessionToken;
  }

  async function handleSendCode() {
    const generation = beginRequestGeneration(emailActionGenerationRef);
    const initialSessionToken = readSession()?.token || "";
    setBusy("send");
    setError("");
    setStatus("");

    try {
      const res = await sendCode(email);
      if (!canCommitEmailAction(generation, initialSessionToken)) {
        return;
      }
      setSendResult({
        email: res.email,
        expiresAt: res.expires_at,
        debugCode: res.debug_code
      });
      setStatus(t({
        en: `Verification code sent to ${res.email}.`,
        zh: `验证码已发送到 ${res.email}。`
      }));
    } catch (err) {
      if (canCommitEmailAction(generation, initialSessionToken)) {
        setError(messageFromError(err));
      }
    } finally {
      if (isLatestMountedEmailAction(generation)) {
        setBusy("");
      }
    }
  }

  async function handleVerifyCode() {
    if (!sendResult) {
      return;
    }

    const verificationEmail = sendResult.email;
    const generation = beginRequestGeneration(emailActionGenerationRef);
    const initialSessionToken = readSession()?.token || "";
    setBusy("verify");
    setError("");
    setStatus("");

    try {
      const session = await verifyCode(verificationEmail, code, readReferralCode());
      if (!canCommitEmailAction(generation, initialSessionToken)) {
        return;
      }
      clearReferralCode();
      setReferralCode("");
      writeSession(session);
      setStatus(t({
        en: "You're signed in.",
        zh: "登录成功。"
      }));
      onSuccess?.(session);
    } catch (err) {
      if (canCommitEmailAction(generation, initialSessionToken)) {
        setError(messageFromError(err));
      }
    } finally {
      if (isLatestMountedEmailAction(generation)) {
        setBusy("");
      }
    }
  }

  async function handleOAuthLogin(provider: OAuthProvider) {
    const providerName = provider === "github" ? "GitHub" : "Google";
    setBusy(provider);
    setError("");
    setStatus(t({
      en: `Loading ${providerName} authorization URL...`,
      zh: `正在加载 ${providerName} 授权地址...`
    }));

    try {
      const redirectURL = await getOAuthRedirectURL(provider, returnTo);
      window.location.href = redirectURL;
    } catch (err) {
      setError(messageFromError(err));
      setStatus("");
      setBusy("");
    }
  }

  return (
    <div className={`sign-in-stack${compact ? " compact" : ""}`}>
      {showReferralField ? (
        <div className="field">
          <label htmlFor="sign-in-referral">{t({ en: "Referral code (optional)", zh: "邀请码（选填）" })}</label>
          <input
            id="sign-in-referral"
            value={referralCode}
            maxLength={64}
            disabled={busy !== ""}
            placeholder="INV123456"
            onChange={(event) => setReferralCode(writeReferralCode(event.target.value))}
          />
        </div>
      ) : referralCode ? (
        <div className="sign-in-caption" role="status">
          {t({ en: "Invite code saved", zh: "已记录邀请码" })}: <span className="inline-code">{referralCode}</span>
          <p>{t({ en: "Applied when you create a new account with any sign-in method. Existing accounts cannot accept a new invitation.", zh: "使用下方任一方式首次注册时，将校验并使用此邀请码。已有账号不能重新绑定邀请。" })}</p>
        </div>
      ) : null}
      {capabilitiesState.status === "idle" || capabilitiesState.status === "loading" ? (
        <Notice>{t({ en: "Loading available sign-in methods...", zh: "正在加载可用登录方式..." })}</Notice>
      ) : capabilitiesState.status === "error" ? (
        <ResourceFailure failure={capabilitiesState.failure} onRetry={() => void loadCapabilities()} />
      ) : null}

      {oauthProviders.map((provider) => {
        const providerName = provider === "github" ? "GitHub" : "Google";
        return (
          <button
            className="sign-in-google-button"
            type="button"
            disabled={busy !== ""}
            onClick={() => handleOAuthLogin(provider)}
            key={provider}
          >
            <span className="sign-in-google-icon" aria-hidden="true">
              {provider === "google" ? (
                <svg viewBox="0 0 48 48" className="sign-in-google-svg">
                  <path fill="#FFC107" d="M43.6 20.5H42V20H24v8h11.3c-1.6 4.6-6 8-11.3 8-6.6 0-12-5.4-12-12s5.4-12 12-12c3 0 5.7 1.1 7.8 3l5.7-5.7C33.5 6 28.9 4 24 4 12.9 4 4 12.9 4 24s8.9 20 20 20 20-8.9 20-20c0-1.2-.1-2.3-.4-3.5z" />
                  <path fill="#FF3D00" d="M6.3 14.7l6.6 4.8C14.3 15.2 18.8 12 24 12c3 0 5.7 1.1 7.8 3l5.7-5.7C33.5 6 28.9 4 24 4 16 4 9.3 8.6 6.3 14.7z" />
                  <path fill="#4CAF50" d="M24 44c5.9 0 11.4-2.3 15.4-6.1l-7.1-5.9C30.2 33.3 27.2 34 24 34c-5.2 0-9.6-3.3-11.3-8l-6.5 5c3 6.1 9.7 10 17.8 10z" />
                  <path fill="#1976D2" d="M43.6 20.5H42V20H24v8h11.3c-.8 2.4-2.3 4.5-4.3 5.9.1-.1 7.1 5.9 7.1 5.9 3-2.8 5.2-6.7 6-11.1.5-2.3.5-4.6.5-6.2z" />
                </svg>
              ) : (
                <svg viewBox="0 0 24 24" className="sign-in-google-svg">
                  <path fill="currentColor" d="M12 .7a11.5 11.5 0 0 0-3.6 22.4c.6.1.8-.2.8-.6v-2.2c-3.3.7-4-1.4-4-1.4-.5-1.4-1.3-1.8-1.3-1.8-1.1-.7.1-.7.1-.7 1.2.1 1.8 1.2 1.8 1.2 1.1 1.8 2.8 1.3 3.5 1 .1-.8.4-1.3.8-1.6-2.6-.3-5.4-1.3-5.4-5.7 0-1.3.5-2.3 1.2-3.1-.1-.3-.5-1.5.1-3.1 0 0 1-.3 3.2 1.2a11 11 0 0 1 5.8 0c2.2-1.5 3.2-1.2 3.2-1.2.6 1.6.2 2.8.1 3.1.8.8 1.2 1.8 1.2 3.1 0 4.4-2.7 5.4-5.4 5.7.4.4.8 1.1.8 2.2v3.3c0 .4.2.7.8.6A11.5 11.5 0 0 0 12 .7Z" />
                </svg>
              )}
            </span>
            <span className="sign-in-google-label">
              {busy === provider
                ? t({ en: "Redirecting...", zh: "跳转中..." })
                : t({ en: `Continue With ${providerName}`, zh: `使用 ${providerName} 登录` })}
            </span>
          </button>
        );
      })}

      {emailEnabled && oauthProviders.length > 0 ? (
        <div className="sign-in-divider">
          <span>{t({ en: "or continue with email", zh: "或使用邮箱登录" })}</span>
        </div>
      ) : null}

      {emailEnabled && (!codeIssued ? (
        <div className="sign-in-step">
          <div className="field">
            <label htmlFor="sign-in-email">{t({ en: "Email", zh: "邮箱" })}</label>
            <input
              id="sign-in-email"
              type="email"
              value={email}
              placeholder="user@example.com"
              onChange={(event) => setEmail(event.target.value)}
            />
          </div>

          <div className="button-row">
            <button className="button primary sign-in-send-button" type="button" disabled={busy !== ""} onClick={handleSendCode}>
              {busy === "send"
                ? t({ en: "Sending...", zh: "发送中..." })
                : t({ en: "Send Code", zh: "发送验证码" })}
            </button>
          </div>
        </div>
      ) : (
        <div className="sign-in-step">
          <div className="details-list sign-in-issued-summary">
            <div className="details-row">
              <strong>{t({ en: "Email", zh: "邮箱" })}</strong>
              <span>{sendResult.email}</span>
            </div>
          </div>

          <div className="field">
            <label htmlFor="sign-in-code">{t({ en: "Verification code", zh: "验证码" })}</label>
            <input
              id="sign-in-code"
              value={code}
              placeholder="123456"
              onChange={(event) => setCode(event.target.value)}
            />
          </div>

          <div className="button-row">
            <button className="button primary wide" type="button" disabled={busy !== ""} onClick={handleVerifyCode}>
              {busy === "verify"
                ? t({ en: "Verifying...", zh: "验证中..." })
                : t({ en: "Verify Code", zh: "验证并登录" })}
            </button>
          </div>
        </div>
      ))}

      {capabilitiesState.status === "ready" && !emailEnabled && oauthProviders.length === 0 ? (
        <Notice tone="error">
          {t({ en: "This API has no compatible login method enabled.", zh: "当前 API 没有启用可兼容的登录方式。" })}
        </Notice>
      ) : null}

      {sendResult?.debugCode ? (
        <Notice tone="success">
          {t({ en: "Debug mode only", zh: "仅调试模式" })}:{" "}
          {t({ en: "the backend returned a visible verification code", zh: "后端返回了可见验证码" })}{" "}
          <span className="inline-code">{sendResult.debugCode}</span>.
        </Notice>
      ) : null}

      {status ? <Notice tone="success">{status}</Notice> : null}
      {error ? <Notice tone="error">{error}</Notice> : null}

    </div>
  );
}
