import { apiUrl } from "./env";
import type { OAuthProvider } from "./env";

const OAUTH_FLOW_KEY = "go-modules.quickstart-nextjs.oauth-flow";
const OAUTH_FLOW_MAX_AGE_MS = 30 * 60 * 1000;
const MAX_PARALLEL_FLOWS = 8;

type StoredOAuthFlow = {
  provider: OAuthProvider;
  verifier: string;
  createdAt: number;
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

export async function prepareOAuthBrowserFlow(provider: OAuthProvider): Promise<string> {
  const raw = new Uint8Array(32);
  globalThis.crypto.getRandomValues(raw);
  const verifier = base64URL(raw);
  const challenge = await verifierChallenge(verifier);
  const storage = browserSessionStorage();
  const flows = readStoredFlows(storage);
  flows[challenge] = { provider, verifier, createdAt: Date.now() };
  const newest = Object.entries(flows)
    .sort(([, left], [, right]) => right.createdAt - left.createdAt)
    .slice(0, MAX_PARALLEL_FLOWS);
  storage.setItem(OAUTH_FLOW_KEY, JSON.stringify(Object.fromEntries(newest)));
  return challenge;
}

export async function getOAuthRedirectURL(provider: OAuthProvider): Promise<string> {
  const challenge = await prepareOAuthBrowserFlow(provider);
  return `${apiUrl(`/auth/${provider}/authorize`)}?redirect=1&challenge=${encodeURIComponent(challenge)}`;
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
