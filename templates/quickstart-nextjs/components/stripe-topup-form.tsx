"use client";

import { FormEvent, useMemo, useState } from "react";
import { Elements, PaymentElement, useElements, useStripe } from "@stripe/react-stripe-js";
import { loadStripe } from "@stripe/stripe-js";
import { Notice } from "@/components/ui";
import { appEnv, appUrl } from "@/lib/env";

const stripePromise = appEnv.stripePublishableKey ? loadStripe(appEnv.stripePublishableKey) : null;

type StripeTopUpFormProps = {
  clientSecret: string;
  amountUSD: number;
  credits: number;
  onBusyChange?: (busy: boolean) => void;
  onError?: (message: string) => void;
};

export function StripeTopUpForm(props: StripeTopUpFormProps) {
  const options = useMemo(
    () => ({
      clientSecret: props.clientSecret,
      appearance: {
        theme: "stripe" as const,
        variables: {
          colorPrimary: "#10403b",
          colorBackground: "#fffaf3",
          colorText: "#23170d",
          colorDanger: "#b6303d",
          borderRadius: "14px"
        }
      }
    }),
    [props.clientSecret]
  );

  if (!appEnv.stripePublishableKey || !stripePromise) {
    return <Notice tone="error">Missing `NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY`. Stripe Payment Element cannot render.</Notice>;
  }

  return (
    <Elements stripe={stripePromise} options={options}>
      <StripeTopUpInner {...props} />
    </Elements>
  );
}

function StripeTopUpInner({ amountUSD, credits, onBusyChange, onError }: StripeTopUpFormProps) {
  const stripe = useStripe();
  const elements = useElements();
  const [submitting, setSubmitting] = useState(false);
  const [localError, setLocalError] = useState("");

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!stripe || !elements) {
      const message = "Stripe is still loading.";
      setLocalError(message);
      onError?.(message);
      return;
    }

    setSubmitting(true);
    setLocalError("");
    onBusyChange?.(true);

    const result = await stripe.confirmPayment({
      elements,
      confirmParams: {
        return_url: appUrl(appEnv.stripeSuccessPath)
      }
    });

    if (result.error) {
      const message = result.error.message || "Payment confirmation failed.";
      setLocalError(message);
      onError?.(message);
      setSubmitting(false);
      onBusyChange?.(false);
      return;
    }
  }

  return (
    <form className="payment-form" onSubmit={handleSubmit}>
      <Notice>
        Stripe will charge <span className="inline-code">${amountUSD.toFixed(2)}</span> and grant{" "}
        <span className="inline-code">{credits}</span> credits after webhook confirmation.
      </Notice>
      <div className="payment-element-shell">
        <PaymentElement />
      </div>
      <div className="button-row">
        <button className="button primary" disabled={!stripe || !elements || submitting} type="submit">
          {submitting ? "Confirming..." : "Pay And Top Up"}
        </button>
      </div>
      {localError ? <Notice tone="error">{localError}</Notice> : null}
    </form>
  );
}
