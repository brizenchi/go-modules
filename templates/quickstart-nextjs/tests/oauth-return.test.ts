import assert from "node:assert/strict";
import { afterEach, test } from "node:test";
import type { AuthSession } from "../lib/auth";

const originalFetch = globalThis.fetch;

function installOAuthBrowser() {
  const storage = () => {
    const values = new Map<string, string>();
    return {
      getItem: (key: string) => values.get(key) ?? null,
      setItem: (key: string, value: string) => values.set(key, value),
      removeItem: (key: string) => values.delete(key)
    };
  };
  Object.defineProperty(globalThis, "window", {
    configurable: true,
    value: { sessionStorage: storage(), localStorage: storage(), dispatchEvent: () => true }
  });
  process.env.NEXT_PUBLIC_API_BASE_URL = "https://api.example.com/api/v1";
  for (const path of ["../lib/oauth-exchange", "../lib/api", "../lib/auth", "../lib/oauth-flow", "../lib/env"]) {
    delete require.cache[require.resolve(path)];
  }
  return {
    flow: require("../lib/oauth-flow") as typeof import("../lib/oauth-flow"),
    exchange: require("../lib/oauth-exchange") as typeof import("../lib/oauth-exchange"),
    auth: require("../lib/auth") as typeof import("../lib/auth")
  };
}

function session(token: string, role: string = "admin"): AuthSession {
  return { token, expires_at: new Date(Date.now() + 600_000).toISOString(), user: { id: token, email: `${token}@example.test`, role } };
}

function response(data: AuthSession) {
  return new Response(JSON.stringify({ code: 200, data }), { status: 200, headers: { "content-type": "application/json" } });
}

afterEach(() => {
  globalThis.fetch = originalFetch;
  Object.defineProperty(globalThis, "window", { configurable: true, value: undefined });
});

test("OAuth exchange consumes an admin flow once and retains its destination for effect replays", async () => {
  const { flow, exchange, auth } = installOAuthBrowser();
  const adminID = await flow.prepareOAuthBrowserFlow("google", "/admin/referrals");
  const ordinaryID = await flow.prepareOAuthBrowserFlow("github");
  let calls = 0;
  globalThis.fetch = (async () => { calls++; return response(session("admin-token")); }) as typeof fetch;

  const first = await exchange.exchangeOAuthCodeOnce("admin-code", adminID);
  assert.equal(first.committed, true);
  assert.equal(first.returnTo, "/admin/referrals");
  assert.equal(flow.resolveOAuthReturnTo(first.returnTo, first.session.user.role), "/admin/referrals");
  assert.equal(flow.readOAuthVerifier(adminID), "");
  assert.equal(flow.readOAuthReturnTo(adminID), "/account");
  assert.notEqual(flow.readOAuthVerifier(ordinaryID), "");
  assert.equal(auth.readSession()?.token, "admin-token");

  const replay = await exchange.exchangeOAuthCodeOnce("admin-code", adminID);
  assert.deepEqual(replay, first);
  assert.equal(calls, 1);
});

test("ordinary OAuth flow never inherits an outstanding admin destination", async () => {
  const { flow, exchange } = installOAuthBrowser();
  const adminID = await flow.prepareOAuthBrowserFlow("google", "/admin/settings");
  const ordinaryID = await flow.prepareOAuthBrowserFlow("github");
  globalThis.fetch = (async () => response(session("ordinary-login-admin-account"))) as typeof fetch;
  const result = await exchange.exchangeOAuthCodeOnce("ordinary-code", ordinaryID);
  assert.equal(result.returnTo, "/account");
  assert.equal(flow.resolveOAuthReturnTo(result.returnTo, result.session.user.role), "/account");
  assert.equal(flow.readOAuthReturnTo(adminID), "/admin/settings");
});

test("failed exchange keeps its flow destination for retry without allowing non-admin return", async () => {
  const { flow, exchange } = installOAuthBrowser();
  const challenge = await flow.prepareOAuthBrowserFlow("google", "/admin/users");
  let calls = 0;
  globalThis.fetch = (async () => {
    if (++calls === 1) throw new Error("temporary network failure");
    return response(session("ordinary-user", "user"));
  }) as typeof fetch;
  await assert.rejects(exchange.exchangeOAuthCodeOnce("retry-code", challenge), /temporary network failure/);
  assert.equal(flow.readOAuthReturnTo(challenge), "/admin/users");
  const result = await exchange.exchangeOAuthCodeOnce("retry-code", challenge);
  assert.equal(result.committed, true);
  assert.equal(flow.resolveOAuthReturnTo(result.returnTo, result.session.user.role), "/account");
  assert.equal(flow.readOAuthVerifier(challenge), "");
});

test("stale admin OAuth result cannot overwrite a newer workspace session", async () => {
  const { flow, exchange, auth } = installOAuthBrowser();
  const challenge = await flow.prepareOAuthBrowserFlow("google", "/admin");
  let complete!: (value: Response) => void;
  globalThis.fetch = (() => new Promise<Response>((resolve) => { complete = resolve; })) as typeof fetch;
  const pending = exchange.exchangeOAuthCodeOnce("old-admin-code", challenge);
  auth.writeSession(session("new-workspace-session", "user"));
  complete(response(session("old-admin-result")));
  const result = await pending;
  assert.equal(result.committed, false);
  assert.equal(auth.readSession()?.token, "new-workspace-session");
  assert.equal(flow.readOAuthReturnTo(challenge), "/account");
});
