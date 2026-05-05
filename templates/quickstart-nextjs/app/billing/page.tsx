"use client";

import { useEffect, useState } from "react";
import { SiteShell } from "@/components/site-shell";
import { EmptyState, Notice, Panel, DetailRows } from "@/components/ui";
import {
  ApiError,
  cancelSubscription,
  changeSubscription,
  createBillingPortalSession,
  createCheckoutSession,
  getSubscription,
  listInvoices,
  previewSubscriptionChange,
  reactivateSubscription,
  type InvoiceItem,
  type SubscriptionChangeMode,
  type SubscriptionPreview,
  type SubscriptionView
} from "@/lib/api";
import {
  readReferralCode,
  readSession,
  REFERRAL_EVENT,
  SESSION_EVENT
} from "@/lib/auth";
import { appEnv, appUrl } from "@/lib/env";
import { formatCurrencyUSD, formatDate } from "@/lib/format";

function messageFromError(error: unknown): string {
  if (error instanceof ApiError) {
    return error.message;
  }
  if (error instanceof Error) {
    return error.message;
  }
  return "unexpected error";
}

const defaultSubscriptionPlan = ["starter", "pro", "premium"].includes(appEnv.defaultPlan)
  ? appEnv.defaultPlan
  : "pro";

const defaultInterval = appEnv.defaultInterval === "yearly" ? "yearly" : "monthly";

export default function BillingPage() {
  const [session, setSession] = useState<ReturnType<typeof readSession>>(null);
  const [busy, setBusy] = useState<"" | "load" | "subscription" | "change" | "portal" | "credits" | "cancel" | "reactivate">("");
  const [status, setStatus] = useState("");
  const [error, setError] = useState("");
  const [subscription, setSubscription] = useState<SubscriptionView | null>(null);
  const [preview, setPreview] = useState<SubscriptionPreview | null>(null);
  const [invoices, setInvoices] = useState<InvoiceItem[]>([]);
  const [plan, setPlan] = useState(defaultSubscriptionPlan);
  const [interval, setInterval] = useState(defaultInterval);
  const [creditsPriceID, setCreditsPriceID] = useState(appEnv.creditsPriceId);
  const [creditsQuantity, setCreditsQuantity] = useState(String(appEnv.defaultCreditsQuantity));
  const [referralCode, setReferralCode] = useState("");
  const currentPlan = subscription?.plan || "";
  const hasLifetime = currentPlan === "lifetime";

  useEffect(() => {
    const syncSession = () => setSession(readSession());
    const syncReferral = () => setReferralCode(readReferralCode());
    syncSession();
    syncReferral();
    window.addEventListener("storage", syncSession);
    window.addEventListener(SESSION_EVENT, syncSession);
    window.addEventListener(REFERRAL_EVENT, syncReferral);
    return () => {
      window.removeEventListener("storage", syncSession);
      window.removeEventListener(SESSION_EVENT, syncSession);
      window.removeEventListener(REFERRAL_EVENT, syncReferral);
    };
  }, []);

  useEffect(() => {
    if (!session) {
      return;
    }
    void loadBillingState(session.token);
  }, [session]);

  useEffect(() => {
    if (!session || !currentPlan || currentPlan === "free") {
      setPreview(null);
      return;
    }

    let cancelled = false;
    void previewSubscriptionChange(session.token, { plan, interval })
      .then((data) => {
        if (!cancelled) {
          setPreview(data);
        }
      })
      .catch(() => {
        if (!cancelled) {
          setPreview(null);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [session, currentPlan, plan, interval]);

  async function loadBillingState(token: string) {
    setBusy("load");
    setError("");

    try {
      const [subscriptionData, invoicesData] = await Promise.all([
        getSubscription(token),
        listInvoices(token)
      ]);
      setSubscription(subscriptionData);
      setInvoices(invoicesData.items);
    } catch (err) {
      setError(messageFromError(err));
    } finally {
      setBusy("");
    }
  }

  async function handleSubscriptionCheckout() {
    if (!session) {
      setError("sign in first");
      return;
    }
    setBusy("subscription");
    setError("");
    setStatus("");

    try {
      const res = await createCheckoutSession(session.token, {
        product_type: "subscription",
        plan,
        interval,
        success_url: appUrl(appEnv.stripeSuccessPath),
        cancel_url: appUrl(appEnv.stripeCancelPath),
        metadata: referralCode ? { referral_code: referralCode } : undefined
      });
      setStatus(`Checkout session created. Redirecting to Stripe: ${res.session_id}`);
      window.location.href = res.checkout_url;
    } catch (err) {
      setError(messageFromError(err));
      setBusy("");
    }
  }

  async function handleChangeSubscription() {
    if (!session) {
      setError("sign in first");
      return;
    }
    setBusy("change");
    setError("");
    setStatus("");

    try {
      const res = await changeSubscription(session.token, {
        plan,
        interval,
        change_mode: preview?.change_mode
      });
      setStatus(res.message);
      await loadBillingState(session.token);
    } catch (err) {
      setError(messageFromError(err));
    } finally {
      setBusy("");
    }
  }

  function changeModeLabel(mode?: SubscriptionChangeMode): string {
    switch (mode) {
      case "immediate_reset_cycle":
        return "Immediate switch, restart billing cycle";
      case "period_end":
        return "Takes effect next billing cycle";
      case "immediate_prorated":
        return "Immediate switch with proration";
      default:
        return "-";
    }
  }

  async function handleOpenPortal() {
    if (!session) {
      setError("sign in first");
      return;
    }
    setBusy("portal");
    setError("");
    setStatus("");

    try {
      const res = await createBillingPortalSession(session.token, `${appEnv.appUrl}/billing`);
      window.location.href = res.url;
    } catch (err) {
      setError(messageFromError(err));
      setBusy("");
    }
  }

  async function handleCreditsCheckout() {
    if (!session) {
      setError("sign in first");
      return;
    }
    setBusy("credits");
    setError("");
    setStatus("");

    try {
      const quantity = Number.parseInt(creditsQuantity, 10);
      const res = await createCheckoutSession(session.token, {
        product_type: "credits",
        price_id: creditsPriceID || undefined,
        quantity: Number.isFinite(quantity) && quantity > 0 ? quantity : 1,
        success_url: appUrl(appEnv.stripeSuccessPath),
        cancel_url: appUrl(appEnv.stripeCancelPath),
        metadata: referralCode ? { referral_code: referralCode } : undefined
      });
      setStatus(`Credits checkout created. Redirecting to Stripe: ${res.session_id}`);
      window.location.href = res.checkout_url;
    } catch (err) {
      setError(messageFromError(err));
      setBusy("");
    }
  }

  async function handleLifetimeCheckout() {
    if (!session) {
      setError("sign in first");
      return;
    }
    setBusy("subscription");
    setError("");
    setStatus("");

    try {
      const res = await createCheckoutSession(session.token, {
        product_type: "lifetime",
        success_url: appUrl(appEnv.stripeSuccessPath),
        cancel_url: appUrl(appEnv.stripeCancelPath),
        metadata: referralCode ? { referral_code: referralCode } : undefined
      });
      setStatus(`Lifetime checkout created. Redirecting to Stripe: ${res.session_id}`);
      window.location.href = res.checkout_url;
    } catch (err) {
      setError(messageFromError(err));
      setBusy("");
    }
  }

  async function handleCancel(cancelType: "end_of_period" | "3days") {
    if (!session) {
      setError("sign in first");
      return;
    }
    setBusy("cancel");
    setError("");
    setStatus("");

    try {
      const res = await cancelSubscription(session.token, cancelType);
      setStatus(res.message);
      await loadBillingState(session.token);
    } catch (err) {
      setError(messageFromError(err));
    } finally {
      setBusy("");
    }
  }

  async function handleReactivate() {
    if (!session) {
      setError("sign in first");
      return;
    }
    setBusy("reactivate");
    setError("");
    setStatus("");

    try {
      const res = await reactivateSubscription(session.token);
      setStatus(res.message);
      await loadBillingState(session.token);
    } catch (err) {
      setError(messageFromError(err));
    } finally {
      setBusy("");
    }
  }

  return (
    <SiteShell
      eyebrow="Subscription Console"
      title="Use one page for upgrades, billing management, invoices, and usage top-ups."
      description="This page is the operational billing console behind the pricing page and avatar menu. It creates Checkout sessions from the browser, then treats the backend and webhook flow as the only source of billing truth."
      accountMenuData={{ subscription }}
      sideTitle="Stripe callback split"
      sideBody={
        <DetailRows
          rows={[
            {
              label: "Frontend success URL",
              value: <span className="inline-code">{appUrl(appEnv.stripeSuccessPath)}</span>
            },
            {
              label: "Frontend cancel URL",
              value: <span className="inline-code">{appUrl(appEnv.stripeCancelPath)}</span>
            },
            {
              label: "Stripe webhook",
              value: <span className="inline-code">https://api.example.com/api/v1/stripe/webhook</span>
            },
            {
              label: "Client never verifies Stripe",
              value: <span>All billing truth comes from backend reads and webhook processing.</span>
            }
          ]}
        />
      }
      toc={[
        { id: "subscription-checkout", label: "Subscription management" },
        { id: "credits-checkout", label: "Credits checkout" },
        { id: "subscription-state", label: "Subscription state" },
        { id: "invoices", label: "Invoices" }
      ]}
    >
      <div className="page-grid">
        <Panel className="span-7" title="Subscription management" subtitle="First purchase uses Checkout. Existing subscriptions should change plan in-place.">
          <div id="subscription-checkout" />
          <div className="field-grid">
            <div className="field">
              <label htmlFor="plan">Plan</label>
              <select id="plan" value={plan} onChange={(event) => setPlan(event.target.value)}>
                <option value="starter">starter</option>
                <option value="pro">pro</option>
                <option value="premium">premium</option>
              </select>
            </div>
            <div className="field">
              <label htmlFor="interval">Interval</label>
              <select id="interval" value={interval} onChange={(event) => setInterval(event.target.value)}>
                <option value="monthly">monthly</option>
                <option value="yearly">yearly</option>
              </select>
            </div>
          </div>
          <Notice>
            Success URL: <span className="inline-code">{appUrl(appEnv.stripeSuccessPath)}</span>
            <br />
            Cancel URL: <span className="inline-code">{appUrl(appEnv.stripeCancelPath)}</span>
            <br />
            Optional referral metadata carried from browser: <span className="inline-code">{referralCode || "-"}</span>
          </Notice>
          <div className="button-row">
            {subscription && subscription.plan !== "free" && !hasLifetime ? (
              <>
                <button className="button primary" disabled={busy !== ""} onClick={handleChangeSubscription}>
                  {busy === "change" ? "Updating..." : "Change Plan"}
                </button>
                <button className="button" disabled={busy !== ""} onClick={handleOpenPortal}>
                  {busy === "portal" ? "Opening..." : "Open Billing Portal"}
                </button>
              </>
            ) : (
              <div className="button-row">
                {!hasLifetime ? (
                  <button className="button primary" disabled={busy !== ""} onClick={handleSubscriptionCheckout}>
                    {busy === "subscription" ? "Creating..." : "Start Subscription Checkout"}
                  </button>
                ) : null}
                {!hasLifetime ? (
                  <button className="button" disabled={busy !== ""} onClick={handleLifetimeCheckout}>
                    {busy === "subscription" ? "Creating..." : "Buy Lifetime"}
                  </button>
                ) : (
                  <Notice tone="success">This account already has lifetime access.</Notice>
                )}
              </div>
            )}
          </div>
          {preview ? (
            <Notice>
              Mode: <span className="inline-code">{changeModeLabel(preview.change_mode)}</span>
              <br />
              Amount due now: <span className="inline-code">{formatCurrencyUSD(preview.amount_due_now)}</span>
              <br />
              Current period end: <span className="inline-code">{formatDate(preview.current_period_end)}</span>
              <br />
              Next billing: <span className="inline-code">{formatDate(preview.next_billing_at)}</span>
              <br />
              {preview.message}
            </Notice>
          ) : null}
          <p className="footer-note">
            Professional default: existing subscriptions change in place with proration; card updates and invoice self-service go through Stripe Billing Portal. Lifetime is a separate one-time buyout path.
          </p>
        </Panel>

        <Panel className="span-5" title="Create credits checkout" subtitle="Matches POST /stripe/checkout/session for product_type=credits.">
          <div id="credits-checkout" />
          <div className="input-row">
            <div className="field">
              <label htmlFor="credits-price">Credits price ID</label>
              <input
                id="credits-price"
                value={creditsPriceID}
                placeholder="price_credits_xxx"
                onChange={(event) => setCreditsPriceID(event.target.value)}
              />
            </div>
            <div className="field">
              <label htmlFor="credits-qty">Quantity</label>
              <input
                id="credits-qty"
                value={creditsQuantity}
                onChange={(event) => setCreditsQuantity(event.target.value)}
              />
            </div>
          </div>
          <div className="button-row">
            <button className="button" disabled={busy !== ""} onClick={handleCreditsCheckout}>
              {busy === "credits" ? "Creating..." : "Buy Credits"}
            </button>
          </div>
          <p className="footer-note">
            Leave the price ID blank only if the backend Stripe config already has exactly one default credits price configured.
          </p>
        </Panel>

        <Panel className="span-6" title="Current subscription" subtitle="Loaded from GET /stripe/subscription.">
          <div id="subscription-state" />
          {subscription ? (
            <div className="details-list">
              <div className="details-row">
                <strong>Plan</strong>
                <span>{subscription.plan}</span>
              </div>
              <div className="details-row">
                <strong>Status</strong>
                <span>{subscription.status}</span>
              </div>
              <div className="details-row">
                <strong>Billing cycle</strong>
                <span>{subscription.billing_cycle || "-"}</span>
              </div>
              <div className="details-row">
                <strong>Current period end</strong>
                <span>{formatDate(subscription.current_period_end)}</span>
              </div>
              <div className="details-row">
                <strong>Cancel at period end</strong>
                <span>{subscription.cancel_at_period_end ? "true" : "false"}</span>
              </div>
              <div className="details-row">
                <strong>Payment method</strong>
                <span>
                  {subscription.payment_method
                    ? `${subscription.payment_method.brand} •••• ${subscription.payment_method.last4}`
                    : "-"}
                </span>
              </div>
            </div>
          ) : (
            <EmptyState>{session ? "No subscription payload loaded yet." : "Sign in to load billing data."}</EmptyState>
          )}
          {!hasLifetime ? (
            <div className="button-row">
              <button className="button danger" disabled={busy !== ""} onClick={() => void handleCancel("end_of_period")}>
                Cancel End Of Period
              </button>
              <button className="button danger" disabled={busy !== ""} onClick={() => void handleCancel("3days")}>
                Cancel In 3 Days
              </button>
              <button className="button" disabled={busy !== ""} onClick={handleReactivate}>
                Reactivate
              </button>
            </div>
          ) : (
            <p className="footer-note">Lifetime access has no recurring cancellation or reactivation flow.</p>
          )}
        </Panel>

        <Panel className="span-6" title="Invoices" subtitle="Loaded from GET /stripe/invoices.">
          <div id="invoices" />
          {invoices.length > 0 ? (
            <table className="table">
              <thead>
                <tr>
                  <th>Period</th>
                  <th>Status</th>
                  <th>Amount</th>
                  <th>Issued</th>
                  <th>PDF</th>
                </tr>
              </thead>
              <tbody>
                {invoices.map((invoice) => {
                  const pdf = invoice.pdf_url || invoice.pdfurl;
                  return (
                    <tr key={invoice.id}>
                      <td>{invoice.period}</td>
                      <td>{invoice.status}</td>
                      <td>{formatCurrencyUSD(invoice.amount_usd)}</td>
                      <td>{formatDate(invoice.created_at)}</td>
                      <td>
                        {pdf ? (
                          <a href={pdf} target="_blank" rel="noreferrer">
                            Open
                          </a>
                        ) : (
                          "-"
                        )}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          ) : (
            <EmptyState>{busy === "load" ? "Loading invoices..." : "No invoices yet."}</EmptyState>
          )}
        </Panel>

        {status ? <div className="span-12"><Notice tone="success">{status}</Notice></div> : null}
        {error ? <div className="span-12"><Notice tone="error">{error}</Notice></div> : null}
      </div>
    </SiteShell>
  );
}
