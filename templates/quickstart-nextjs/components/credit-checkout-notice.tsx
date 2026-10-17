"use client";

import { useEffect, useState } from "react";
import { confirmCreditCheckout, getCreditCheckoutResult, type CreditCheckoutResult } from "@/lib/api";
import { readSession } from "@/lib/auth";
import { checkoutReturnStatus } from "@/lib/billing-state";
import { readCreditCheckoutSession, waitForCreditCheckout } from "@/lib/credit-checkout";
import { useI18n } from "@/lib/i18n";
import { ConsoleError } from "./console-kit";
import { Notice } from "./ui";
import { usePageReturn } from "./use-page-return";

export function CreditCheckoutNotice({ token, onConfirmed }: { token: string; onConfirmed: () => Promise<void> }) {
  const { t, locale } = useI18n();
  const [feedback, setFeedback] = useState<{ owner: string; phase: "checking" | "pending" | "confirmed" | "failed"; credits: number } | null>(null);
  const [error, setError] = useState<unknown>(null);
  const [revision, setRevision] = useState(0);
  usePageReturn(() => setRevision(value => value + 1));

  useEffect(() => {
    if (!token || checkoutReturnStatus(window.location.search) !== "success") return;
    const sessionID = readCreditCheckoutSession(window.location.search);
    let active = true;
    let timer: number | undefined;
    let releaseWait: (() => void) | undefined;
    const current = () => active && readSession()?.token === token;
    setFeedback(previous => previous?.owner === token ? previous : { owner: token, phase: "checking", credits: 0 });
    setError(null);
    async function confirm() {
      try {
        const result: CreditCheckoutResult | null = sessionID ? await waitForCreditCheckout({
          confirm: () => confirmCreditCheckout(token, sessionID),
          load: () => getCreditCheckoutResult(token, sessionID), current,
          wait: delay => new Promise<void>(resolve => { releaseWait = resolve; timer = window.setTimeout(resolve, delay); })
        }) : null;
        if (!current()) return;
        if (result?.state === "fulfilled") {
          await onConfirmed();
          if (!current()) return;
          setFeedback({ owner: token, phase: "confirmed", credits: result.credits_granted });
          const url = new URL(window.location.href);
          url.searchParams.delete("checkout"); url.searchParams.delete("session_id");
          window.history.replaceState(window.history.state, "", `${url.pathname}${url.search}${url.hash}`);
        } else {
          setFeedback({ owner: token, phase: result?.state === "failed" || result?.state === "expired" ? "failed" : "pending", credits: 0 });
        }
      } catch (failure) {
        if (current()) { setError(failure); setFeedback({ owner: token, phase: "pending", credits: 0 }); }
      }
    }
    void confirm();
    return () => { active = false; window.clearTimeout(timer); releaseWait?.(); };
  }, [token, onConfirmed, revision]);

  if (!feedback || feedback.owner !== token) return null;
  const count = new Intl.NumberFormat(locale === "zh" ? "zh-CN" : "en-US").format(feedback.credits);
  return <div role="status"><Notice tone={feedback.phase === "confirmed" ? "success" : "default"}>
    {feedback.phase === "confirmed" ? t({ en: `${count} credits added to your account.`, zh: `${count} 积分已到账。` })
      : feedback.phase === "checking" ? t({ en: "Confirming your credit purchase…", zh: "正在确认积分到账…" })
      : feedback.phase === "failed" ? t({ en: "This checkout was not completed. No purchased credits were added.", zh: "这笔购买尚未完成，未发放购买积分。" })
      : t({ en: "Your credits have not been confirmed yet. If you have paid, please contact support before purchasing again.", zh: "积分尚未确认到账。如果已经付款，请联系支持，先不要重复购买。" })}
    <ConsoleError error={error} />
  </Notice></div>;
}
