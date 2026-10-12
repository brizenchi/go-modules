// Stripe amounts are integers in the currency's smallest unit.
export function formatBillingAmount(amount: number | null, currency: string, locale = "en-US"): string {
  if (amount === null || !Number.isSafeInteger(amount) || !/^[a-zA-Z]{3}$/.test(currency)) return "—";
  try {
    const formatter = new Intl.NumberFormat(locale, { style: "currency", currency: currency.toUpperCase() });
    // Stripe retains a two-decimal wire representation for ISK and UGX.
    // https://docs.stripe.com/currencies#special-cases
    const digits = ["ISK", "UGX"].includes(currency.toUpperCase())
      ? 2
      : formatter.resolvedOptions().maximumFractionDigits ?? 2;
    return formatter.format(amount / 10 ** digits);
  } catch {
    return "—";
  }
}
