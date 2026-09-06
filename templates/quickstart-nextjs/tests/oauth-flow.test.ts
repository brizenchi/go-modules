import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { afterEach, test } from "node:test";

const values = new Map<string, string>();

function installBrowserStorage() {
  values.clear();
  Object.defineProperty(globalThis, "window", {
    configurable: true,
    value: {
      sessionStorage: {
        getItem: (key: string) => values.get(key) ?? null,
        setItem: (key: string, value: string) => values.set(key, value),
        removeItem: (key: string) => values.delete(key)
      }
    }
  });
}

function loadOAuthFlow() {
  process.env.NEXT_PUBLIC_API_BASE_URL = "https://api.example.com/api/v1";
  for (const name of ["../lib/oauth-flow", "../lib/env"]) {
    delete require.cache[require.resolve(name)];
  }
  return require("../lib/oauth-flow") as typeof import("../lib/oauth-flow");
}

afterEach(() => {
  Object.defineProperty(globalThis, "window", { configurable: true, value: undefined });
});

test("OAuth redirect carries only a verifier challenge and retains verifier in sessionStorage", async () => {
  installBrowserStorage();
  const flow = loadOAuthFlow();
  const redirectURL = new URL(await flow.getOAuthRedirectURL("github"));
  const verifier = flow.readOAuthVerifier();
  const expectedChallenge = createHash("sha256").update(verifier).digest("base64url");

  assert.equal(redirectURL.origin, "https://api.example.com");
  assert.equal(redirectURL.pathname, "/api/v1/auth/github/authorize");
  assert.equal(redirectURL.searchParams.get("redirect"), "1");
  assert.equal(redirectURL.searchParams.get("challenge"), expectedChallenge);
  assert.match(verifier, /^[A-Za-z0-9_-]{43}$/);
  assert.doesNotMatch(redirectURL.toString(), new RegExp(verifier));

  flow.clearOAuthVerifier();
  assert.equal(flow.readOAuthVerifier(), "");
});

test("malformed or expired browser flow records are removed", () => {
  installBrowserStorage();
  const flow = loadOAuthFlow();
  const key = "go-modules.quickstart-nextjs.oauth-flow";
  values.set(key, "not-json");
  assert.equal(flow.readOAuthVerifier(), "");
  assert.equal(values.has(key), false);

	values.set(key, JSON.stringify({
		"expired-challenge": {
			provider: "google",
			verifier: "stale",
			createdAt: Date.now() - 31 * 60 * 1000
		}
	}));
  assert.equal(flow.readOAuthVerifier(), "");
  assert.equal(values.has(key), false);
});

test("parallel OAuth flows keep independent verifiers and clear only the completed flow", async () => {
	installBrowserStorage();
	const flow = loadOAuthFlow();
	const firstChallenge = await flow.prepareOAuthBrowserFlow("google");
	const firstVerifier = flow.readOAuthVerifier(firstChallenge);
	const secondChallenge = await flow.prepareOAuthBrowserFlow("github");
	const secondVerifier = flow.readOAuthVerifier(secondChallenge);

	assert.notEqual(firstChallenge, secondChallenge);
	assert.notEqual(firstVerifier, secondVerifier);
	assert.equal(createHash("sha256").update(firstVerifier).digest("base64url"), firstChallenge);
	assert.equal(createHash("sha256").update(secondVerifier).digest("base64url"), secondChallenge);

	flow.clearOAuthVerifier(firstChallenge);
	assert.equal(flow.readOAuthVerifier(firstChallenge), "");
	assert.equal(flow.readOAuthVerifier(secondChallenge), secondVerifier);
});

test("OAuth callback errors clear only their verifier and return safe friendly messages", async () => {
	installBrowserStorage();
	const flow = loadOAuthFlow();
	const deniedChallenge = await flow.prepareOAuthBrowserFlow("google");
	const otherChallenge = await flow.prepareOAuthBrowserFlow("github");
	const otherVerifier = flow.readOAuthVerifier(otherChallenge);

	assert.equal(
		flow.consumeOAuthCallbackError("access_denied", deniedChallenge),
		"OAuth sign-in was canceled. No account changes were made."
	);
	assert.equal(flow.readOAuthVerifier(deniedChallenge), "");
	assert.equal(flow.readOAuthVerifier(otherChallenge), otherVerifier);

	assert.equal(
		flow.consumeOAuthCallbackError("callback_failed", otherChallenge),
		"OAuth sign-in could not be completed. Please try again."
	);
	assert.equal(flow.readOAuthVerifier(otherChallenge), "");
});

test("unrecognized OAuth error query does not discard a verifier", async () => {
	installBrowserStorage();
	const flow = loadOAuthFlow();
	const challenge = await flow.prepareOAuthBrowserFlow("google");
	const verifier = flow.readOAuthVerifier(challenge);

	assert.equal(
		flow.consumeOAuthCallbackError("attacker-controlled", challenge),
		"OAuth sign-in could not be completed. Please try again."
	);
	assert.equal(flow.readOAuthVerifier(challenge), verifier);
});

test("admin destinations belong to the exact browser flow and never leak into ordinary sign-in", async () => {
  installBrowserStorage();
  const flow = loadOAuthFlow();
  const adminURL = new URL(await flow.getOAuthRedirectURL("google", "/admin/credits"));
  const adminID = adminURL.searchParams.get("challenge")!;
  const userID = await flow.prepareOAuthBrowserFlow("github");

  assert.equal(flow.readOAuthReturnTo(adminID), "/admin/credits");
  assert.equal(flow.readOAuthReturnTo(userID), "/account");
  assert.equal(flow.readOAuthReturnTo(""), "/account");
  assert.equal(flow.readOAuthReturnTo("unrelated-flow"), "/account");
  assert.equal(adminURL.searchParams.has("returnTo"), false);
  assert.equal(adminURL.searchParams.has("return_to"), false);
  assert.doesNotMatch(adminURL.toString(), /admin/);

  flow.clearOAuthVerifier(adminID);
  assert.equal(flow.readOAuthReturnTo(adminID), "/account");
  assert.notEqual(flow.readOAuthVerifier(userID), "");
});

test("OAuth accepts only known internal destinations and gates admin return on the signed-in role", async () => {
  installBrowserStorage();
  const flow = loadOAuthFlow();
  for (const path of ["https://evil.example/admin", "//evil.example", "/\\evil.example", "/admin/../account", "/admin/unknown", "/admin/users?next=https://evil.example", "/admin#settings", "/%61dmin", "/admin/", " /admin"]) {
    await assert.rejects(flow.prepareOAuthBrowserFlow("google", path), /Unsupported OAuth return destination/);
    assert.equal(flow.resolveOAuthReturnTo(path, "admin"), "/account");
  }
  assert.equal(values.size, 0);
  for (const path of ["/admin", "/admin/users", "/admin/orders", "/admin/subscriptions", "/admin/referrals", "/admin/credits", "/admin/settings", "/admin/audit"]) {
    assert.equal(flow.resolveOAuthReturnTo(path, "admin"), path);
    assert.equal(flow.resolveOAuthReturnTo(path, "user"), "/account");
    assert.equal(flow.resolveOAuthReturnTo(path), "/account");
  }
  assert.equal(flow.resolveOAuthReturnTo("/account", "admin"), "/account");
});

test("canceled, expired, missing and tampered destinations fall back safely", async () => {
  installBrowserStorage();
  const flow = loadOAuthFlow();
  const denied = await flow.prepareOAuthBrowserFlow("google", "/admin/settings");
  flow.consumeOAuthCallbackError("access_denied", denied);
  assert.equal(flow.readOAuthReturnTo(denied), "/account");

  const challenge = await flow.prepareOAuthBrowserFlow("github", "/admin/users");
  const key = "go-modules.quickstart-nextjs.oauth-flow";
  const records = JSON.parse(values.get(key)!);
  records[challenge].returnTo = "https://evil.example";
  values.set(key, JSON.stringify(records));
  assert.equal(flow.readOAuthReturnTo(challenge), "/account");
  records[challenge].returnTo = "/admin/users";
  records[challenge].createdAt = Date.now() - 31 * 60 * 1000;
  values.set(key, JSON.stringify(records));
  assert.equal(flow.readOAuthReturnTo(challenge), "/account");
  assert.equal(values.has(key), false);

  const legacy = await flow.prepareOAuthBrowserFlow("google");
  const legacyRecords = JSON.parse(values.get(key)!);
  delete legacyRecords[legacy].returnTo;
  values.set(key, JSON.stringify(legacyRecords));
  assert.equal(flow.readOAuthReturnTo(legacy), "/account");
});
