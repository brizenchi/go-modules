import { getCapabilities } from "./api";
import type { AuthMethodCapabilities } from "./auth-methods";

const maxAgeMs = 60_000;

function authMethodsOnly(capabilities: AuthMethodCapabilities): AuthMethodCapabilities {
  return capabilities.auth ? {
    auth: { ...capabilities.auth, oauth_providers: [...capabilities.auth.oauth_providers] }
  } : {};
}

/** Cache discovery only: never sessions, authorization URLs or billing facts. */
export function createSignInMethodsCache(
  loader: () => Promise<AuthMethodCapabilities>,
  now: () => number = Date.now
) {
  let snapshot: { data: AuthMethodCapabilities; expiresAt: number } | null = null;
  let pending: Promise<AuthMethodCapabilities> | null = null;

  function peek(): AuthMethodCapabilities | null {
    return snapshot && now() < snapshot.expiresAt ? snapshot.data : null;
  }

  function load(force = false): Promise<AuthMethodCapabilities> {
    if (pending) return pending;
    const cached = force ? null : peek();
    if (cached) return Promise.resolve(cached);

    snapshot = null;
    pending = Promise.resolve().then(loader).then((capabilities) => {
      const data = authMethodsOnly(capabilities);
      snapshot = { data, expiresAt: now() + maxAgeMs };
      return data;
    }).finally(() => { pending = null; });
    return pending;
  }

  return { peek, load };
}

// A tab-local cache. Server rendering must never share discovery across requests.
const browserCache = createSignInMethodsCache(getCapabilities);

export function peekSignInMethods(): AuthMethodCapabilities | null {
  return typeof window === "undefined" ? null : browserCache.peek();
}

export function loadSignInMethods(force = false): Promise<AuthMethodCapabilities> {
  return typeof window === "undefined"
    ? getCapabilities().then(authMethodsOnly)
    : browserCache.load(force);
}
