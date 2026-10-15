import { isPublicPath } from "./locale";

export function analyticsMeasurementId(value: string): string {
  const id = value.trim();
  return /^G-[A-Z0-9]{4,32}$/.test(id) ? id : "";
}
export function crispChatURL(value: string): string {
  const id = value.trim();
  if (!/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(id)) return "";
  return `https://go.crisp.chat/chat/embed/?website_id=${id}`;
}
/** Only published public paths are eligible; never forward purchase/auth queries. */
export function analyticsPageURL(origin: string, pathname: string): string | null {
  try {
    const url = new URL(pathname, origin);
    if (url.origin !== new URL(origin).origin || !["http:","https:"].includes(url.protocol) || !isPublicPath(url.pathname)) return null;
    return `${url.origin}${url.pathname}`;
  } catch { return null; }
}
export function analyticsConsent(value: string | null): "allowed" | "declined" | null {
  return value === "allowed" || value === "declined" ? value : null;
}
