import assert from "node:assert/strict";
import { test } from "node:test";
import { getPublicSiteSettings, normalizePublicSiteSettings, safeSupportEmail, safeSupportURL } from "../lib/site-settings";

test("support links accept contact details and reject executable or credential-bearing URLs", () => {
  assert.equal(safeSupportEmail(" help@example.com "), "help@example.com");
  for (const value of ["a@example.com?bcc=other@example.com", "a@example.com%0Abcc:other", "a@b", "a@b.com\r\nCc: x@y.com"]) assert.equal(safeSupportEmail(value), "");
  assert.equal(safeSupportURL("https://help.example.com/support"), "https://help.example.com/support");
  for (const value of ["javascript:alert(1)", "data:text/html,x", "http://help.example.com", "https://user:pass@example.com", "//help.example.com"]) assert.equal(safeSupportURL(value), "");
});

test("public settings expose only known values and keep unconfigured contact channels empty", () => {
  const parsed = normalizePublicSiteSettings({ brand_name: " Brand ", description: " A product ", support_email: "", support_url: "javascript:alert(1)", export_credit_cost: 4, secret_key: "private" });
  assert.deepEqual(parsed, { brand_name: "Brand", description: "A product", support_email: "", support_url: "", export_credit_cost: 4 });
  for (const value of [0, -1, 0.5, 2.5, 1000001, NaN, Infinity, "4"]) {
    assert.equal(normalizePublicSiteSettings({ export_credit_cost: value }).export_credit_cost, 1);
  }
  assert.equal(normalizePublicSiteSettings({ export_credit_cost: 1000000 }).export_credit_cost, 1000000);
  assert.throws(() => normalizePublicSiteSettings(null));
});

test("public settings fetch reads the API envelope without authentication and rejects application errors", async (context) => {
  const requests: Array<{ input: RequestInfo | URL; init?: RequestInit }> = [];
  context.mock.method(globalThis, "fetch", async (input: RequestInfo | URL, init?: RequestInit) => {
    requests.push({ input, init });
    return new Response(JSON.stringify({ code: 200, data: { brand_name: "Test site", support_email: "help@example.com", support_url: "", export_credit_cost: 1 } }), { status: 200 });
  });
  const controller = new AbortController();
  const settings = await getPublicSiteSettings(controller.signal);
  assert.equal(settings.brand_name, "Test site");
  assert.ok(String(requests[0].input).endsWith("/api/v1/site/settings"));
  assert.equal(requests[0].init?.signal, controller.signal);
  assert.equal(requests[0].init?.headers, undefined);
  context.mock.method(globalThis, "fetch", async () => new Response(JSON.stringify({ code: 500, data: {} }), { status: 200 }));
  await assert.rejects(getPublicSiteSettings(), /Invalid site settings/);
});
