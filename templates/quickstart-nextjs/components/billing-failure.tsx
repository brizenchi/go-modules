"use client";

import { billingFailureText, type BillingRequestFailure } from "@/lib/billing-presentation";
import { useI18n } from "@/lib/i18n";
import { LocalizedLink as Link } from "./localized-link";

export function BillingFailure({ failure, onRetry }: { failure: BillingRequestFailure; onRetry?: () => void }) {
  const { t } = useI18n();
  const copy = billingFailureText(failure.kind, failure.reason);
  return <div className="resource-feedback" role="alert"><span className="resource-feedback-mark" aria-hidden="true">!</span><div className="resource-feedback-copy">
    <strong>{t(copy.title)}</strong><p>{t(copy.message)}</p>
    {failure.requestId ? <p className="request-id">{t({ en: "Request ID", zh: "请求编号" })}: <code>{failure.requestId}</code></p> : null}
    {failure.kind === "auth" ? <Link className="button" href="/login">{t({ en: "Sign in", zh: "登录" })}</Link>
      : failure.retryable && onRetry ? <button className="button" onClick={onRetry}>{t(failure.reason === "subscription_change_pending" ? { en: "Check billing status", zh: "查看最新状态" } : { en: "Try again", zh: "重试" })}</button> : null}
  </div></div>;
}
