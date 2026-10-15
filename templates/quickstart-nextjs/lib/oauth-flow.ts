import { apiUrl } from "./env";
import type { OAuthProvider } from "./env";

const OAUTH_FLOW_KEY = "go-modules.quickstart-nextjs.oauth-flow";
const OAUTH_FLOW_MAX_AGE_MS = 30 * 60 * 1000;
const MAX_PARALLEL_FLOWS = 8;

const ADMIN_RETURN_PATHS = [
  "/admin", "/admin/users", "/admin/orders", "/admin/subscriptions",
  "/admin/referrals", "/admin/credits", "/admin/settings", "/admin/integrations", "/admin/audit"
] as const;
const CUSTOMER_RETURN_PATHS = ["/account", "/billing", "/referrals", "/credits", "/pricing", "/zh/pricing"] as const;
type CustomerReturnPath = (typeof CUSTOMER_RETURN_PATHS)[number];
type CustomerReturnTo = CustomerReturnPath | `${CustomerReturnPath}?${string}` | `${CustomerReturnPath}#${string}`;
export type OAuthReturnTo = (typeof ADMIN_RETURN_PATHS)[number] | CustomerReturnTo;

function allowedCustomerReturnPath(value: unknown): value is CustomerReturnTo {
  if (typeof value !== "string" || value.length > 2048 || value.includes("\\")) return false;
  if (Array.from(value).some(character => character.charCodeAt(0) <= 32 || character.charCodeAt(0) === 127)) return false;
  // Match the literal path before any query/fragment. Encoded or relative path
  // normalization must never turn a customer destination into an admin route.
  const pathname = value.split(/[?#]/, 1)[0];
  return (CUSTOMER_RETURN_PATHS as readonly string[]).includes(pathname);
}

function allowedReturnPath(value: unknown): value is OAuthReturnTo {
  return allowedCustomerReturnPath(value)
    || (typeof value === "string" && (ADMIN_RETURN_PATHS as readonly string[]).includes(value));
}

// This is a destination hint, never authorization. Admin routes still verify
// their own role. Non-admin identities may only return to customer pages.
export function resolveOAuthReturnTo(value: unknown, role?: string): OAuthReturnTo {
  return allowedCustomerReturnPath(value) || (role === "admin" && allowedReturnPath(value)) ? value : "/account";
}

/** Explicit admin sign-in wins; ordinary dialogs retain only customer routes. */
export function resolveSignInDestination(currentPath: string, explicitReturnTo?: string): OAuthReturnTo {
  if (explicitReturnTo !== undefined) return allowedReturnPath(explicitReturnTo) ? explicitReturnTo : "/account";
  return allowedCustomerReturnPath(currentPath) ? currentPath : "/account";
}

type StoredOAuthFlow = {
  provider: OAuthProvider;
  verifier: string;
  createdAt: number;
  returnTo?: OAuthReturnTo;
};

type StoredOAuthFlows = Record<string, StoredOAuthFlow>;

function browserSessionStorage(): Storage {
  if (typeof window === "undefined" || !window.sessionStorage) {
    throw new Error("OAuth sign-in requires browser session storage");
  }
  return window.sessionStorage;
}

function base64URL(bytes: Uint8Array): string {
  let binary = "";
  for (const value of bytes) {
    binary += String.fromCharCode(value);
  }
  return btoa(binary).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/g, "");
}

async function verifierChallenge(verifier: string): Promise<string> {
  const digest = await globalThis.crypto.subtle.digest(
    "SHA-256",
    new TextEncoder().encode(verifier)
  );
  return base64URL(new Uint8Array(digest));
}

export async function prepareOAuthBrowserFlow(provider: OAuthProvider, returnTo: string = "/account"): Promise<string> {
  if (!allowedReturnPath(returnTo)) {
    throw new Error("Unsupported OAuth return destination");
  }
  const raw = new Uint8Array(32);
  globalThis.crypto.getRandomValues(raw);
  const verifier = base64URL(raw);
  const challenge = await verifierChallenge(verifier);
  const storage = browserSessionStorage();
  const flows = readStoredFlows(storage);
  flows[challenge] = { provider, verifier, createdAt: Date.now(), returnTo };
  const newest = Object.entries(flows)
    .sort(([, left], [, right]) => right.createdAt - left.createdAt)
    .slice(0, MAX_PARALLEL_FLOWS);
  storage.setItem(OAUTH_FLOW_KEY, JSON.stringify(Object.fromEntries(newest)));
  return challenge;
}

export async function getOAuthRedirectURL(provider: OAuthProvider, returnTo?: string): Promise<string> {
  const challenge = await prepareOAuthBrowserFlow(provider, returnTo);
  return `${apiUrl(`/auth/${provider}/authorize`)}?redirect=1&challenge=${encodeURIComponent(challenge)}`;
}

export function readOAuthReturnTo(flowID: string): OAuthReturnTo {
  // A destination is accepted only from this callback's exact, unexpired flow.
  // Never inherit the newest flow: it might be an abandoned admin sign-in.
  if (!flowID) return "/account";
  const storage = browserSessionStorage();
  const flows = readStoredFlows(storage);
  const returnTo = flows[flowID]?.returnTo;
  persistFlows(storage, flows);
  return allowedReturnPath(returnTo) ? returnTo : "/account";
}

export function readOAuthVerifier(flowID = ""): string {
  const storage = browserSessionStorage();
  const flows = readStoredFlows(storage);
  const selected = flowID ? flows[flowID] : Object.values(flows)
    .sort((left, right) => right.createdAt - left.createdAt)[0];
  persistFlows(storage, flows);
  return selected?.verifier || "";
}

export function clearOAuthVerifier(flowID = ""): void {
  const storage = browserSessionStorage();
  if (!flowID) {
    storage.removeItem(OAUTH_FLOW_KEY);
    return;
  }
  const flows = readStoredFlows(storage);
  delete flows[flowID];
	persistFlows(storage, flows);
}

export function consumeOAuthCallbackError(errorCode: string, flowID = ""): string {
	const normalized = errorCode.trim().toLowerCase();
	if ((normalized === "access_denied" || normalized === "callback_failed") && flowID) {
		// The backend returns the verifier challenge as flow. Clearing that exact
		// key preserves any other Google/GitHub login running in the same tab.
		clearOAuthVerifier(flowID);
	}
	if (normalized === "access_denied") {
		return "OAuth sign-in was canceled. No account changes were made.";
	}
	return "OAuth sign-in could not be completed. Please try again.";
}

function readStoredFlows(storage: Storage): StoredOAuthFlows {
  const raw = storage.getItem(OAUTH_FLOW_KEY);
  if (!raw) {
    return {};
  }
  try {
    const parsed = JSON.parse(raw) as Record<string, Partial<StoredOAuthFlow>>;
    const now = Date.now();
    const flows: StoredOAuthFlows = {};
    for (const [challenge, flow] of Object.entries(parsed)) {
      if (
        flow !== null
        && typeof flow === "object"
        && typeof flow.provider === "string"
        && typeof flow.verifier === "string"
        && typeof flow.createdAt === "number"
        && Number.isFinite(flow.createdAt)
        && flow.createdAt <= now
        && now - flow.createdAt <= OAUTH_FLOW_MAX_AGE_MS
      ) {
        flows[challenge] = flow as StoredOAuthFlow;
      }
    }
    return flows;
  } catch {
    return {};
  }
}

function persistFlows(storage: Storage, flows: StoredOAuthFlows): void {
  if (Object.keys(flows).length === 0) {
    storage.removeItem(OAUTH_FLOW_KEY);
    return;
  }
  storage.setItem(OAUTH_FLOW_KEY, JSON.stringify(flows));
}
