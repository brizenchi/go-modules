import assert from "node:assert/strict";
import { afterEach, test } from "node:test";
import { ApiError, loginAdmin, type CapabilitiesView } from "../lib/api";
import { adminPasswordEnabled, createAdminPasswordLogin, describeAdminLoginFailure } from "../lib/admin-password-login";
import type { AuthSession } from "../lib/auth";

const originalFetch = globalThis.fetch;
afterEach(() => { globalThis.fetch = originalFetch; });

const admin: AuthSession = { token: "admin-token", expires_at: "2099-01-01T00:00:00Z", user: { id: "admin", email: "owner@example.test", role: "admin" } };
function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (error: unknown) => void;
  const promise = new Promise<T>((yes, no) => { resolve = yes; reject = no; });
  return { promise, resolve, reject };
}

test("administrator password is sent only in the login POST body and preserves literal spaces", async () => {
  const controller = new AbortController();
  globalThis.fetch = (async (url, options) => {
    assert.match(String(url), /\/auth\/admin\/login$/);
    assert.equal(new URL(String(url)).search, "");
    assert.equal(options?.method, "POST");
    assert.equal(options?.cache, "no-store");
    assert.equal(options?.signal, controller.signal);
    assert.equal(new Headers(options?.headers).get("Authorization"), null);
    assert.deepEqual(JSON.parse(String(options?.body)), { email: "owner@example.test", password: "  private test password  " });
    return new Response(JSON.stringify({ code: 200, data: admin }), { headers: { "content-type": "application/json" } });
  }) as typeof fetch;
  assert.deepEqual(await loginAdmin(" owner@example.test ", "  private test password  ", controller.signal), admin);
});

test("administrator password login is enabled only by an explicit backend capability", () => {
  const capabilities: CapabilitiesView = { account: { enabled: true }, referral: { enabled: true }, billing: { enabled: true, provider: "stripe", offers: { subscriptions: [], lifetime: false, credits: false } } };
  assert.equal(adminPasswordEnabled(capabilities), false);
  capabilities.auth = { email_enabled: true, oauth_providers: ["google"] };
  assert.equal(adminPasswordEnabled(capabilities), false);
  capabilities.auth.admin_password_enabled = false;
  assert.equal(adminPasswordEnabled(capabilities), false);
  capabilities.auth.admin_password_enabled = true;
  assert.equal(adminPasswordEnabled(capabilities), true);
});

test("successful administrator authentication commits once even when the session event cancels the form", async () => {
  const commits: AuthSession[] = [];
  const login = createAdminPasswordLogin({ authenticate: async () => admin, readSessionToken: () => "", commit: (session) => { commits.push(session); login.cancel(); } });
  assert.equal(await login.login("owner@example.test", "password"), "committed");
  assert.deepEqual(commits, [admin]);
});

test("a late administrator response cannot replace a more recent signed-in account", async () => {
  const result = deferred<AuthSession>();
  let token = "";
  const commits: AuthSession[] = [];
  const login = createAdminPasswordLogin({ authenticate: () => result.promise, readSessionToken: () => token, commit: (session) => commits.push(session) });
  const pending = login.login("owner@example.test", "password");
  token = "new-user-session";
  result.resolve(admin);
  assert.equal(await pending, "stale");
  assert.deepEqual(commits, []);
});

test("session-change cancellation prevents resurrection after another sign-in and logout", async () => {
  const result = deferred<AuthSession>();
  let token = "";
  let signal: AbortSignal | undefined;
  const commits: AuthSession[] = [];
  const login = createAdminPasswordLogin({ authenticate: (_email, _password, requestSignal) => { signal = requestSignal; return result.promise; }, readSessionToken: () => token, commit: (session) => commits.push(session) });
  const pending = login.login("owner@example.test", "password");
  token = "new-session";
  login.cancel();
  token = "";
  login.cancel();
  assert.equal(signal?.aborted, true);
  result.resolve(admin);
  assert.equal(await pending, "stale");
  assert.deepEqual(commits, []);
});

test("unmount cancellation ignores both late success and late failure", async () => {
  for (const succeeds of [true, false]) {
    const result = deferred<AuthSession>();
    const commits: AuthSession[] = [];
    const login = createAdminPasswordLogin({ authenticate: () => result.promise, readSessionToken: () => "", commit: (session) => commits.push(session) });
    const pending = login.login("owner@example.test", "password");
    login.cancel();
    if (succeeds) result.resolve(admin); else result.reject(new Error("late service failure"));
    assert.equal(await pending, "stale");
    assert.deepEqual(commits, []);
  }
});

test("repeated submits share one active attempt and a failed attempt allows a new password", async () => {
  const result = deferred<AuthSession>();
  const passwords: string[] = [];
  const login = createAdminPasswordLogin({ authenticate: (_email, password) => { passwords.push(password); return passwords.length === 1 ? result.promise : Promise.resolve(admin); }, readSessionToken: () => "", commit: () => {} });
  const first = login.login("owner@example.test", "wrong password");
  assert.equal(await login.login("owner@example.test", "second click"), "busy");
  result.reject(new ApiError("invalid admin email or password", 401, 401));
  await assert.rejects(first, ApiError);
  assert.equal(await login.login("owner@example.test", "correct password"), "committed");
  assert.deepEqual(passwords, ["wrong password", "correct password"]);
});

test("cancelling one request permits a fresh attempt without letting the old response commit", async () => {
  const first = deferred<AuthSession>();
  const second = deferred<AuthSession>();
  let calls = 0;
  const commits: AuthSession[] = [];
  const login = createAdminPasswordLogin({ authenticate: () => ++calls === 1 ? first.promise : second.promise, readSessionToken: () => "", commit: (session) => commits.push(session) });
  const abandoned = login.login("owner@example.test", "old password");
  login.cancel();
  const current = login.login("owner@example.test", "new password");
  first.resolve({ ...admin, token: "stale-token" });
  assert.equal(await abandoned, "stale");
  second.resolve(admin);
  assert.equal(await current, "committed");
  assert.deepEqual(commits, [admin]);
});

test("a normal user response is never persisted as an administrator session", async () => {
  const commits: AuthSession[] = [];
  const login = createAdminPasswordLogin({ authenticate: async () => ({ ...admin, user: { ...admin.user, role: "user" } }), readSessionToken: () => "", commit: (session) => commits.push(session) });
  await assert.rejects(login.login("owner@example.test", "password"), (error) => error instanceof ApiError && error.status === 403);
  assert.deepEqual(commits, []);
});

test("login errors have distinct credential, throttle, setup and availability messages", () => {
  assert.equal(describeAdminLoginFailure(new ApiError("invalid admin email or password", 401, 401)), "credentials");
  assert.equal(describeAdminLoginFailure(new ApiError("forbidden", 403, 403)), "credentials");
  assert.equal(describeAdminLoginFailure(new ApiError("too many admin login attempts", 429, 429)), "limited");
  assert.equal(describeAdminLoginFailure(new ApiError("admin password login is not configured", 503, 503)), "disabled");
  assert.equal(describeAdminLoginFailure(new ApiError("database details must not reach the login form", 500, 500)), "unavailable");
  assert.equal(describeAdminLoginFailure(new TypeError("network failed")), "unavailable");
});
