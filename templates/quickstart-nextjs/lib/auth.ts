export type SessionUser = {
  id: string;
  email: string;
  username?: string;
  avatar?: string;
  role?: string;
  is_new?: boolean;
};

export type AuthSession = {
  token: string;
  expires_at: string;
  user: SessionUser;
};

const SESSION_KEY = "go-modules.quickstart-nextjs.session";
const REFERRAL_KEY = "go-modules.quickstart-nextjs.referral-code";
export const SESSION_EVENT = "go-modules.quickstart-nextjs.session-change";
export const REFERRAL_EVENT = "go-modules.quickstart-nextjs.referral-change";

function isBrowser(): boolean {
  return typeof window !== "undefined";
}

function safeParse<T>(raw: string | null): T | null {
  if (!raw) {
    return null;
  }

  try {
    return JSON.parse(raw) as T;
  } catch {
    return null;
  }
}

function emit(name: string): void {
  if (!isBrowser()) {
    return;
  }
  window.dispatchEvent(new Event(name));
}

export function readSession(): AuthSession | null {
  if (!isBrowser()) {
    return null;
  }

  const session = safeParse<AuthSession>(window.localStorage.getItem(SESSION_KEY));
  if (!session?.token || !session?.expires_at || !session.user?.id) {
    window.localStorage.removeItem(SESSION_KEY);
    return null;
  }

  const expiresAt = new Date(session.expires_at);
  if (Number.isNaN(expiresAt.getTime()) || expiresAt.getTime() <= Date.now()) {
    window.localStorage.removeItem(SESSION_KEY);
    return null;
  }

  return session;
}

export function writeSession(session: AuthSession | null): void {
  if (!isBrowser()) {
    return;
  }

  if (!session) {
    window.localStorage.removeItem(SESSION_KEY);
    emit(SESSION_EVENT);
    return;
  }

  window.localStorage.setItem(SESSION_KEY, JSON.stringify(session));
  emit(SESSION_EVENT);
}

export function readReferralCode(): string {
  if (!isBrowser()) {
    return "";
  }

  return (window.localStorage.getItem(REFERRAL_KEY) || "").trim();
}

export function writeReferralCode(code: string): string {
  const normalized = code.trim();
  if (!isBrowser()) {
    return normalized;
  }

  if (!normalized) {
    window.localStorage.removeItem(REFERRAL_KEY);
    emit(REFERRAL_EVENT);
    return "";
  }

  window.localStorage.setItem(REFERRAL_KEY, normalized);
  emit(REFERRAL_EVENT);
  return normalized;
}

export function clearReferralCode(): void {
  if (!isBrowser()) {
    return;
  }
  window.localStorage.removeItem(REFERRAL_KEY);
  emit(REFERRAL_EVENT);
}
