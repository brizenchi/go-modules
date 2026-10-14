export type BillingIntent =
  | { product: "subscription"; plan: string; interval: "monthly" | "yearly" }
  | { product: "lifetime" }
  | { product: "credits" };

/** Preserve a selection across navigation and sign-in; never starts a purchase. */
export function billingIntentHref(intent: BillingIntent): string {
  const params = new URLSearchParams({ product: intent.product });
  if (intent.product === "subscription") {
    params.set("plan", intent.plan);
    params.set("interval", intent.interval);
  }
  return `/billing?${params.toString()}#${intent.product === "credits" ? "credits" : "plans"}`;
}

/** Only parses the intent. The caller must check it against the live catalog. */
export function readBillingIntent(search: string): BillingIntent | null {
  const params = new URLSearchParams(search);
  if (["product", "plan", "interval"].some((key) => params.getAll(key).length > 1)) return null;

  const product = params.get("product");
  if (product === "credits" || product === "lifetime") {
    return params.has("plan") || params.has("interval") ? null : { product };
  }
  if (product !== "subscription") return null;

  const plan = params.get("plan");
  const interval = params.get("interval");
  if (!plan || plan.length > 64 || !/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(plan)) return null;
  if (interval !== "monthly" && interval !== "yearly") return null;
  return { product, plan, interval };
}
