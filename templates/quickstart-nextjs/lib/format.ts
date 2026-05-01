export function formatDate(value?: string | null): string {
  if (!value) {
    return "-";
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return new Intl.DateTimeFormat("en-US", {
    dateStyle: "medium",
    timeStyle: "short"
  }).format(date);
}

export function formatCurrencyUSD(value?: number | null): string {
  const amount = typeof value === "number" ? value : 0;
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: "USD"
  }).format(amount);
}

export function maskToken(token?: string | null): string {
  if (!token) {
    return "-";
  }
  if (token.length <= 20) {
    return token;
  }
  return `${token.slice(0, 10)}...${token.slice(-8)}`;
}
