import { apiUrl } from "./env";
import type { OAuthProvider } from "./env";

const OAUTH_FLOW_KEY = "go-modules.quickstart-nextjs.oauth-flow";
const OAUTH_FLOW_MAX_AGE_MS = 30 * 60 * 1000;
const MAX_PARALLEL_FLOWS = 8;

const OAUTH_RETURN_PATHS = [
  "/account", "/admin", "/admin/users", "/admin/orders", "/admin/subscriptions",
  "/admin/referrals", "/admin/credits", "/admin/settings", "/admin/audit"
] as const;
export type OAuthReturnTo = (typeof OAUTH_RETURN_PATHS)[number];

function allowedReturnPath(value: unknown): value is OAuthReturnTo {
  return typeof value === "string" && (OAUTH_RETURN_PATHS as readonly string[]).includes(value);
}

// This is a destination hint, never authorization. Admin routes still verify
// their own role. Public sign-in and non-admin identities stay in the workspace.
export function resolveOAuthReturnTo(value: unknown, role?: string): OAuthReturnTo {
  return role === "admin" && allowedReturnPath(value) ? value : "/account";
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
