import assert from "node:assert/strict";
import { afterEach, test } from "node:test";
import { getOperatorAuthorization, getOperatorOverview, getOperatorAccess, updateOperatorMembership, deductOperatorCredits, exportNote, getPrivateImage, listCreditTransactions, listNotes, retryReferralReward, saveBusinessSettings, uploadImage } from "../lib/operations-api";
import { ApiError } from "../lib/api";
import { readSession, writeSession } from "../lib/auth";

const originalFetch = globalThis.fetch;
afterEach(() => {
  globalThis.fetch = originalFetch;
  Object.defineProperty(globalThis, "window", { configurable: true, value: undefined });
});
function response(data: unknown) { return new Response(JSON.stringify({ code: 200, data }), { headers: { "content-type": "application/json" } }); }

test("note exports send the agreed cost and preserve the intent key for retries", async () => {
  const calls: Array<{url: string; body: unknown; token: string | null}> = [];
  globalThis.fetch = (async (url, options) => {
    calls.push({ url: String(url), body: JSON.parse(String(options?.body)), token: new Headers(options?.headers).get("Authorization") });
    return response({ filename: "note.md", content: "# Saved note", transaction_id: 4, balance: 9 });
  }) as typeof fetch;
  const result = await exportNote("user-token", 7, "same-export-key", 1);
  await exportNote("user-token", 7, "same-export-key", 1);
  assert.equal(result.content, "# Saved note");
  assert.match(calls[0].url, /\/notes\/7\/export$/);
  assert.deepEqual(calls[0].body, { idempotency_key: "same-export-key", expected_cost: 1 });
  assert.deepEqual(calls[1], calls[0]);
  assert.equal(calls[0].token, "Bearer user-token");
});

test("operator changes carry reason and idempotency header without accepting a client actor", async () => {
  const calls: Array<{url: string; key: string | null; body: Record<string, unknown>}> = [];
  globalThis.fetch = (async (url, options) => {
    calls.push({ url: String(url), key: new Headers(options?.headers).get("Idempotency-Key"), body: JSON.parse(String(options?.body)) });
    return response({});
  }) as typeof fetch;
  await retryReferralReward("admin-token", 5, "reconcile original reward", "reward-key");
  await saveBusinessSettings("admin-token", { brand_name: "SaaS", description: "Starter", support_email: "", support_url: "", export_credit_cost: 2, reason: "update export price" }, "settings-key");
  assert.equal(calls[0].key, "reward-key");
  assert.deepEqual(calls[0].body, { reason: "reconcile original reward" });
  assert.equal(calls[1].key, "settings-key");
  assert.equal(calls[1].body.export_credit_cost, 2);
  assert.equal(calls[1].body.actor_id, undefined);
});

test("different backend list envelopes map to usable frontend records and encode account filters", async () => {
  let url = "";
  globalThis.fetch = (async (input) => { url = String(input); return response({ list: [{ id: 1 }], total: 42, page: 2, limit: 20 }); }) as typeof fetch;
  const credits = await listCreditTransactions("admin", 2, "alice+test&b=1", true);
  assert.deepEqual(credits.items, [{ id: 1 }]);
  assert.equal(credits.total, 42);
  assert.equal(new URL(url).searchParams.get("user_id"), "alice+test&b=1");
  assert.equal(new URL(url).searchParams.has("b"), false);
  assert.deepEqual(await listNotes("user"), [{ id: 1 }]);
});

test("image uploads preserve multipart boundary and private image reads carry bearer auth", async () => {
  const file = new File([new Uint8Array([137, 80, 78, 71])], "example.png", { type: "image/png" });
  globalThis.fetch = (async (_url, options) => {
    assert.ok(options?.body instanceof FormData);
    assert.equal(new Headers(options?.headers).get("Content-Type"), null);
    assert.equal((options.body.get("file") as File).name, "example.png");
    return response({ id: "image-1" });
  }) as typeof fetch;
  await uploadImage("user-token", file);
  globalThis.fetch = (async (url, options) => {
    assert.match(String(url), /\/uploads\/images\/image-1$/);
    assert.equal(new Headers(options?.headers).get("Authorization"), "Bearer user-token");
    return new Response("private bytes", { headers: { "content-type": "image/png" } });
  }) as typeof fetch;
  assert.equal(await (await getPrivateImage("user-token", "image-1")).text(), "private bytes");
});

test("a stale private image request cannot sign out the newly logged-in account", async () => {
  const storage = new Map<string, string>();
  Object.defineProperty(globalThis, "window", { configurable: true, value: {
    localStorage: { getItem: (key: string) => storage.get(key) ?? null, setItem: (key: string, value: string) => storage.set(key, value), removeItem: (key: string) => storage.delete(key) }, dispatchEvent: () => true
  } });
  writeSession({ token: "new-token", expires_at: new Date(Date.now() + 60000).toISOString(), user: { id: "new-user", email: "new@example.test" } });
  globalThis.fetch = (async () => new Response(null, { status: 401 })) as typeof fetch;
  await assert.rejects(getPrivateImage("old-token", "old-file"), ApiError);
  assert.equal(readSession()?.token, "new-token");
});

test("admin reporting carries an explicit timezone, window and payment environment", async () => {
 let captured = "";
 globalThis.fetch = (async url => { captured = String(url); return response({}); }) as typeof fetch;
 await getOperatorOverview("admin", undefined, { days: 7, mode: "test", timezone: "Asia/Shanghai" });
 const query = new URL(captured).searchParams;
 assert.equal(query.get("days"), "7"); assert.equal(query.get("mode"), "test"); assert.equal(query.get("timezone"), "Asia/Shanghai");
});
test("membership and credit edits stay user scoped and preserve retry keys", async () => {
 const calls: Array<{ url: string; method: string | undefined; key: string | null; body: unknown }> = [];
 globalThis.fetch = (async (url, options) => { calls.push({ url: String(url), method: options?.method, key: new Headers(options?.headers).get("Idempotency-Key"), body: options?.body ? JSON.parse(String(options.body)) : null }); return response({}); }) as typeof fetch;
 await getOperatorAccess("admin", "target+user");
 await updateOperatorMembership("admin", "target+user", { plan: "lifetime", reason: "Support grant" }, "member-key");
 await deductOperatorCredits("admin", "target+user", 30, "Support correction", "deduct-key");
 assert.match(calls[0].url, /target%2Buser\/access$/);
 assert.equal(calls[1].method, "PATCH"); assert.equal(calls[1].key, "member-key");
 assert.deepEqual(calls[1].body, { plan: "lifetime", reason: "Support grant" });
 assert.equal(calls[2].key, "deduct-key"); assert.deepEqual(calls[2].body, { amount: 30, reason: "Support correction" });
});

test("admin authorization supports an older API without hiding permission failures", async () => {
 const paths: string[] = [];
 globalThis.fetch = (async url => { const path = new URL(String(url)).pathname; paths.push(path); return path.endsWith("/access") ? new Response(null, { status: 404 }) : response({ users: 1 }); }) as typeof fetch;
 assert.deepEqual(await getOperatorAuthorization("admin"), { authorized: true });
 assert.equal(paths.length, 2); assert.ok(paths[1].endsWith("/overview"));
 paths.length = 0;
 globalThis.fetch = (async url => { paths.push(String(url)); return new Response(null, { status: 403 }); }) as typeof fetch;
 await assert.rejects(getOperatorAuthorization("user"), error => error instanceof ApiError && error.status === 403);
 assert.equal(paths.length, 1);
});
