"use client";

import { useEffect, useState } from "react";
import { Notice } from "@/components/ui";
import {
  ApiError,
  getGoogleAuthorizeURL,
  sendCode,
  verifyCode
} from "@/lib/api";
import {
  clearReferralCode,
  readReferralCode,
  writeReferralCode,
  writeSession,
  type AuthSession
} from "@/lib/auth";
import { useI18n } from "@/lib/i18n";

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
  onSuccess?: (session: AuthSession) => void;
};

export function SignInPanel({
  compact = false,
  showReferralField = false,
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
  const [busy, setBusy] = useState<"" | "send" | "verify" | "google">("");
  const codeIssued = !!sendResult;

  useEffect(() => {
    setReferralCode(readReferralCode());
  }, []);

  async function handleSendCode() {
    setBusy("send");
    setError("");
    setStatus("");

    try {
      const res = await sendCode(email);
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
      setError(messageFromError(err));
    } finally {
      setBusy("");
    }
  }

  async function handleVerifyCode() {
    setBusy("verify");
    setError("");
    setStatus("");

    try {
      const session = await verifyCode(email, code, referralCode);
      clearReferralCode();
      setReferralCode("");
      writeSession(session);
      setStatus(t({
        en: "Email login succeeded. Session saved in localStorage.",
        zh: "邮箱登录成功，会话已保存到 localStorage。"
      }));
      onSuccess?.(session);
    } catch (err) {
      setError(messageFromError(err));
    } finally {
      setBusy("");
    }
  }

  async function handleGoogleLogin() {
    setBusy("google");
    setError("");
    setStatus(t({
      en: "Loading Google authorization URL...",
      zh: "正在加载 Google 授权地址..."
    }));

    try {
      const redirectURL = await getGoogleAuthorizeURL();
      window.location.href = redirectURL;
    } catch (err) {
      setError(messageFromError(err));
      setStatus("");
      setBusy("");
    }
  }

  return (
    <div className={`sign-in-stack${compact ? " compact" : ""}`}>
      <button className="sign-in-google-button" type="button" disabled={busy !== ""} onClick={handleGoogleLogin}>
        <span className="sign-in-google-icon" aria-hidden="true">
          <svg viewBox="0 0 48 48" className="sign-in-google-svg">
            <path fill="#FFC107" d="M43.6 20.5H42V20H24v8h11.3c-1.6 4.6-6 8-11.3 8-6.6 0-12-5.4-12-12s5.4-12 12-12c3 0 5.7 1.1 7.8 3l5.7-5.7C33.5 6 28.9 4 24 4 12.9 4 4 12.9 4 24s8.9 20 20 20 20-8.9 20-20c0-1.2-.1-2.3-.4-3.5z" />
            <path fill="#FF3D00" d="M6.3 14.7l6.6 4.8C14.3 15.2 18.8 12 24 12c3 0 5.7 1.1 7.8 3l5.7-5.7C33.5 6 28.9 4 24 4 16 4 9.3 8.6 6.3 14.7z" />
            <path fill="#4CAF50" d="M24 44c5.9 0 11.4-2.3 15.4-6.1l-7.1-5.9C30.2 33.3 27.2 34 24 34c-5.2 0-9.6-3.3-11.3-8l-6.5 5c3 6.1 9.7 10 17.8 10z" />
            <path fill="#1976D2" d="M43.6 20.5H42V20H24v8h11.3c-.8 2.4-2.3 4.5-4.3 5.9.1-.1 7.1 5.9 7.1 5.9 3-2.8 5.2-6.7 6-11.1.5-2.3.5-4.6.5-6.2z" />
          </svg>
        </span>
        <span className="sign-in-google-label">
          {busy === "google"
            ? t({ en: "Redirecting...", zh: "跳转中..." })
            : t({ en: "Continue With Google", zh: "使用 Google 登录" })}
        </span>
      </button>

      <div className="sign-in-divider">
        <span>{t({ en: "or continue with email", zh: "或使用邮箱登录" })}</span>
      </div>

      {!codeIssued ? (
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

          {showReferralField ? (
            <div className="field">
              <label htmlFor="sign-in-referral">{t({ en: "Referral code", zh: "邀请码" })}</label>
              <input
                id="sign-in-referral"
                value={referralCode}
                placeholder="INV123456"
                onChange={(event) => setReferralCode(writeReferralCode(event.target.value))}
              />
            </div>
          ) : referralCode ? (
            <div className="sign-in-caption">
              {t({ en: "Captured referral code", zh: "已捕获邀请码" })}:{" "}
              <span className="inline-code">{referralCode}</span>
            </div>
          ) : null}

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
      )}

      {sendResult?.debugCode ? (
        <Notice tone="success">
          {t({ en: "Debug mode only", zh: "仅调试模式" })}:{" "}
          {t({ en: "the backend returned a visible verification code", zh: "后端返回了可见验证码" })}{" "}
          <span className="inline-code">{sendResult.debugCode}</span>.
        </Notice>
      ) : null}

      {status ? <Notice tone="success">{status}</Notice> : null}
      {error ? <Notice tone="error">{error}</Notice> : null}

      <p className="footer-note">
        {t({
          en: "Google sign-in still uses backend OAuth redirect flow. Email sign-in stays in this UI.",
          zh: "Google 登录仍然走后端 OAuth 重定向流程，邮箱登录则直接在当前界面完成。"
        })}
      </p>
    </div>
  );
}
