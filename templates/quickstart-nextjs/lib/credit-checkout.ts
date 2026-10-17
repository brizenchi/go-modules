import type { CreditCheckoutResult } from "./api";

const CREDIT_CONFIRMATION_DELAYS_MS = [0, 1000, 2000, 4000, 8000, 15000, 30000] as const;

export function readCreditCheckoutSession(search: string): string | null {
  const params = new URLSearchParams(search);
  const id = params.get("session_id") || "";
  return params.getAll("checkout").length === 1 && params.get("checkout") === "success"
    && params.getAll("session_id").length === 1 && /^cs_[A-Za-z0-9_]{1,252}$/.test(id) ? id : null;
}

/** Poll verified server records; a return URL or a completed timer cannot prove fulfillment. */
export async function waitForCreditCheckout(deps: {
  confirm?: () => Promise<CreditCheckoutResult>;
  load: () => Promise<CreditCheckoutResult>;
  current: () => boolean;
  wait: (delay: number) => Promise<void>;
}): Promise<CreditCheckoutResult | null> {
  let result: CreditCheckoutResult | null = null;
  let confirmationComplete = !deps.confirm;
  let failure: unknown;
  for (const delay of CREDIT_CONFIRMATION_DELAYS_MS) {
    await deps.wait(delay);
    if (!deps.current()) return null;
    try {
      result = await (!confirmationComplete && deps.confirm ? deps.confirm() : deps.load());
      confirmationComplete = true;
      failure = undefined;
    } catch (error) { if (!deps.current()) return null; failure = error; continue; }
    if (!deps.current()) return null;
    if (result.state === "fulfilled") {
      if (!Number.isSafeInteger(result.credits_granted) || result.credits_granted <= 0) throw new Error("Invalid credit fulfillment result");
      return result;
    }
    if (result.state === "failed" || result.state === "expired") return result;
  }
  if (failure && !result) throw failure;
  return result;
}
