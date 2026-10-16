import { apiUrl } from "./env";

export const maxLogoBytes = 5 * 1024 * 1024;
export const logoMimeTypes = ["image/png", "image/jpeg", "image/webp", "image/gif"];
const uploadedLogo = /^\/api\/v1\/site\/logo\/[a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12}$/;

export function safeLogoURL(value: unknown): string {
  if (typeof value !== "string") return "";
  const trimmed = value.trim();
  if (!trimmed || trimmed.length > 1024 || /[\\\u0000-\u0020]/.test(trimmed)) return "";
  if (trimmed.startsWith("/api/")) return uploadedLogo.test(trimmed) ? trimmed : "";
  if (trimmed.startsWith("/") && !trimmed.startsWith("//")) return trimmed;
  try {
    const url = new URL(trimmed);
    return url.protocol === "https:" && !url.username && !url.password ? trimmed : "";
  } catch { return ""; }
}

export function logoImageURL(value: unknown): string {
  const url = safeLogoURL(value);
  return uploadedLogo.test(url) ? apiUrl(url.slice("/api/v1".length)) : url;
}

export function logoFileError(file: Pick<File, "type" | "size">): "type" | "size" | null {
  if (!logoMimeTypes.includes(file.type)) return "type";
  return file.size < 1 || file.size > maxLogoBytes ? "size" : null;
}
