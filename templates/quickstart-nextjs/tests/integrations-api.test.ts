import assert from "node:assert/strict";
import { afterEach, test } from "node:test";
import { getAdminIntegrations, saveAdminIntegration, IntegrationError, type IntegrationView } from "../lib/integrations-api";
import { readSession, writeSession } from "../lib/auth";

const originalFetch = globalThis.fetch;
afterEach(() => {
  globalThis.fetch = originalFetch;
  Object.defineProperty(globalThis, "window", { configurable: true, value: undefined });
});
function view(provider: "resend" | "stripe"): IntegrationView {
  return {
    provider, source: "database", version: 4, enabled: false,
    fields: provider === "resend" ? { sender_email: "hello@example.test", sender_name: "Demo", email_auth_enabled: "false" } : { mode: "test", trial_days: "0", credits_per_package: "100" },
    secrets: provider === "resend" ? { api_key: { configured: true } } : { secret_key: { configured: true }, webhook_secret: { configured: false } },
    active_enabled: true, active_source: "environment", restart_required: true
  };
}
function response(data: unknown) { return new Response(JSON.stringify({ code: 200, data }), { headers: { "content-type": "application/json" } }); }

test("service reads use admin auth and retain saved/current/preview status without returning secret values", async () => {
  const controller = new AbortController();
  const resend = view("resend");
  globalThis.fetch = (async (url, options) => {
    assert.match(String(url), /\/admin\/integrations$/);
    assert.equal(new Headers(options?.headers).get("Authorization"), "Bearer admin-token");
    assert.equal(options?.signal, controller.signal);
    assert.equal(options?.cache, "no-store");
    return response({ storage_ready: true, local_preview: true, providers: [
      { ...resend, private_key: "do-not-read", fields: { ...resend.fields, api_key: "do-not-read" }, secrets: { api_key: { configured: true, value: "do-not-read" } } }, view("stripe")
    ] });
  }) as typeof fetch;
  const result = await getAdminIntegrations("admin-token", controller.signal);
  assert.equal(result.local_preview, true);
  assert.equal(result.providers[0].active_source, "environment");
  assert.equal(result.providers[0].source, "database");
  assert.equal(result.providers[0].restart_required, true);
  assert.deepEqual(result.providers[0].secrets, { api_key: { configured: true } });
  assert.equal(JSON.stringify(result).includes("do-not-read"), false);
});

test("service changes preserve secret bytes and retry identity with optimistic version in an authenticated PATCH", async () => {
  const calls: Array<{ url: string; options?: RequestInit }> = [];
  const patch = { version: 3, source: "database" as const, enabled: true, fields: { sender_email: "hello@example.test" }, secrets: { api_key: "re_exactKey123" }, reason: "configure mail" };
  globalThis.fetch = (async (url, options) => { calls.push({ url: String(url), options }); return response(view("resend")); }) as typeof fetch;
  await saveAdminIntegration("admin-token", "resend", patch, "same-save-key");
  await saveAdminIntegration("admin-token", "resend", patch, "same-save-key");
  for (const call of calls) {
    assert.match(call.url, /\/admin\/integrations\/resend$/);
    assert.equal(call.options?.method, "PATCH");
    assert.equal(new Headers(call.options?.headers).get("Idempotency-Key"), "same-save-key");
    assert.equal(new Headers(call.options?.headers).get("Authorization"), "Bearer admin-token");
    assert.deepEqual(JSON.parse(String(call.options?.body)), patch);
  }
});

test("service API failures retain only safe categories, never raw provider messages or credentials", async () => {
  for (const [status, kind] of [[400, "invalid"], [401, "auth"], [403, "forbidden"], [409, "conflict"], [503, "storage"], [500, "unavailable"]] as const) {
    globalThis.fetch = (async () => new Response(JSON.stringify({ code: status, message: "provider leaked sk_test_never-show" }), { status, headers: { "content-type": "application/json" } })) as typeof fetch;
    await assert.rejects(saveAdminIntegration("admin", "resend", { version: 0, source: "environment", reason: "reset config" }, "key"), (error: unknown) => {
      assert.ok(error instanceof IntegrationError);
      assert.equal(error.kind, kind);
      assert.equal(error.message.includes("sk_test"), false);
      assert.equal(error.cause, undefined);
      return true;
    });
  }
});

test("a stale service 401 cannot clear a newly signed-in account", async () => {
  const storage = new Map<string, string>();
  Object.defineProperty(globalThis, "window", { configurable: true, value: {
    localStorage: { getItem: (key: string) => storage.get(key) ?? null, setItem: (key: string, value: string) => storage.set(key, value), removeItem: (key: string) => storage.delete(key) }, dispatchEvent: () => true
  } });
  writeSession({ token: "new-token", expires_at: new Date(Date.now() + 60000).toISOString(), user: { id: "new-admin", email: "admin@example.test", role: "admin" } });
  globalThis.fetch = (async () => new Response(JSON.stringify({ code: 401 }), { status: 401, headers: { "content-type": "application/json" } })) as typeof fetch;
  await assert.rejects(getAdminIntegrations("old-token"), IntegrationError);
  assert.equal(readSession()?.token, "new-token");
});

test("missing or mismatched provider responses fail closed without supplying invented defaults", async () => {
  for (const providers of [[view("resend")], [view("resend"), view("resend")]]) {
    globalThis.fetch = (async () => response({ storage_ready: true, local_preview: false, providers })) as typeof fetch;
    await assert.rejects(getAdminIntegrations("admin"), IntegrationError);
  }
  globalThis.fetch = (async () => response(view("stripe"))) as typeof fetch;
  await assert.rejects(saveAdminIntegration("admin", "resend", { version: 0, source: "environment", reason: "reset config" }, "key"), IntegrationError);
});
