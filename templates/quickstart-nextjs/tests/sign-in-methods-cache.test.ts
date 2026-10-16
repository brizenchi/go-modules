import assert from "node:assert/strict";
import { test } from "node:test";
import { createSignInMethodsCache } from "../lib/sign-in-methods-cache";
import type { CapabilitiesView } from "../lib/api";

const capabilities: CapabilitiesView = {
  auth: { email_enabled: true, oauth_providers: ["google"] },
  account: { enabled: true },
  billing: {
    enabled: true,
    provider: "stripe",
    offers: { subscriptions: [], lifetime: false, credits: false }
  },
  referral: { enabled: true }
};

test("sign-in discovery is lazy, retains only auth methods and expires after one minute", async () => {
  let requests = 0;
  let now = 100;
  const cache = createSignInMethodsCache(async () => {
    requests++;
    return capabilities;
  }, () => now);

  assert.equal(requests, 0);
  assert.equal(cache.peek(), null);
  const methods = await cache.load();
  assert.deepEqual(methods, { auth: capabilities.auth });
  assert.equal("billing" in methods, false);
  assert.equal(cache.peek(), methods);
  now += 59_999;
  assert.equal(await cache.load(), methods);
  assert.equal(requests, 1);

  now++;
  assert.equal(cache.peek(), null);
  await cache.load();
  assert.equal(requests, 2);
});

test("closing and reopening while discovery is pending shares one request", async () => {
  let requests = 0;
  let finish!: (value: CapabilitiesView) => void;
  const cache = createSignInMethodsCache(() => {
    requests++;
    return new Promise<CapabilitiesView>((resolve) => { finish = resolve; });
  });

  const firstOpen = cache.load();
  const secondOpen = cache.load();
  await Promise.resolve();
  assert.equal(requests, 1);
  assert.equal(firstOpen, secondOpen);
  finish(capabilities);
  assert.equal(await firstOpen, await secondOpen);
});

test("failed discovery is not cached and can be retried", async () => {
  let requests = 0;
  const cache = createSignInMethodsCache(async () => {
    requests++;
    if (requests === 1) throw new Error("fixture offline");
    return capabilities;
  });

  await assert.rejects(cache.load(), /fixture offline/);
  assert.equal(cache.peek(), null);
  assert.deepEqual(await cache.load(), { auth: capabilities.auth });
  assert.equal(requests, 2);
});

test("explicit retry refreshes methods and cannot retain a disabled provider", async () => {
  let requests = 0;
  const cache = createSignInMethodsCache(async () => {
    requests++;
    return requests === 1 ? capabilities : { auth: { email_enabled: false, oauth_providers: [] } };
  });

  await cache.load();
  const refreshed = await cache.load(true);
  assert.equal(requests, 2);
  assert.deepEqual(refreshed, { auth: { email_enabled: false, oauth_providers: [] } });
  assert.equal(cache.peek(), refreshed);
});

test("an unsuccessful refresh clears old methods and legacy APIs remain compatible", async () => {
  let requests = 0;
  const cache = createSignInMethodsCache(async () => {
    requests++;
    if (requests === 2) throw new Error("fixture offline");
    return requests === 1 ? capabilities : {};
  });

  await cache.load();
  await assert.rejects(cache.load(true), /fixture offline/);
  assert.equal(cache.peek(), null);
  assert.deepEqual(await cache.load(), {});
});
