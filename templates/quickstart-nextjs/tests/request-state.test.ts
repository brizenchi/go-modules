import assert from "node:assert/strict";
import { test } from "node:test";
import { ApiError } from "../lib/api";
import { describeRequestFailure, settleResource, readyResource, startResourceRefresh, reconcileResourceRefresh } from "../lib/request-state";
import { retainEqualData } from "../lib/resource-data";

test("background refresh leaves an existing screen ready until changed data arrives", () => {
  const previous = readyResource({ registered: 3, paid: 1, rows: [{ id: 1 }] });
  assert.strictEqual(startResourceRefresh(previous, true), previous);
  assert.strictEqual(reconcileResourceRefresh(previous, readyResource({ paid: 1, rows: [{ id: 1 }], registered: 3 }), true), previous);
  const changed = reconcileResourceRefresh(previous, readyResource({ registered: 4, paid: 2, rows: [{ id: 1 }] }), true);
  assert.equal(changed.status, "ready");
  assert.equal(changed.data?.registered, 4);
  assert.equal(previous.data?.registered, 3);
});

test("background outages preserve displayed data while access errors still surface", async () => {
  const previous = readyResource({ count: 3 });
  for (const error of [new TypeError("offline"), new ApiError("failure", 500, 500)]) {
    const next = await settleResource(Promise.reject(error), "Stats");
    assert.strictEqual(reconcileResourceRefresh(previous, next, true), previous);
    assert.equal(reconcileResourceRefresh(previous, next).status, "error");
  }
  for (const status of [401, 403]) {
    const next = await settleResource(Promise.reject(new ApiError("denied", status, status)), "Stats");
    assert.equal(reconcileResourceRefresh(previous, next, true).status, "error");
    assert.equal(reconcileResourceRefresh(previous, next, true).data, null);
  }
});

test("first loads retain their loading and error feedback", () => {
  assert.equal(startResourceRefresh({ status: "idle", data: null, failure: null }, true).status, "loading");
  assert.equal(startResourceRefresh(readyResource({ count: 0 })).status, "loading");
});

test("equal JSON data retains its reference but changed and reordered lists update", () => {
  const previous = { items: [{ id: 1, value: null }, { id: 2, value: false }], total: 2 };
  assert.strictEqual(retainEqualData(previous, { total: 2, items: [{ value: null, id: 1 }, { id: 2, value: false }] }), previous);
  assert.notStrictEqual(retainEqualData(previous, { items: [...previous.items].reverse(), total: 2 }), previous);
  assert.notStrictEqual(retainEqualData(previous, { items: previous.items, total: 3 }), previous);
  assert.notStrictEqual(retainEqualData(previous, { items: [{ id: 1, value: false }, previous.items[1]], total: 2 }), previous);
  assert.deepEqual(retainEqualData(null, { total: 0, items: [] }), { total: 0, items: [] });
});

test("request failures distinguish auth, disabled, configuration, outage, and network errors", () => {
  assert.equal(describeRequestFailure(new ApiError("unauthorized", 401, 401), "Profile").kind, "auth");
  assert.equal(describeRequestFailure(new ApiError("not found", 404, 404), "Billing").kind, "disabled");
  assert.equal(describeRequestFailure(new ApiError("billing is not configured", 503, 503), "Billing").kind, "configuration");
  assert.equal(describeRequestFailure(new ApiError("service unavailable", 503, 503), "Billing").kind, "unavailable");
  assert.equal(describeRequestFailure(new TypeError("fetch failed"), "Profile").kind, "network");
});

test("settleResource preserves successful resources when a sibling request fails", async () => {
  const [ready, failed] = await Promise.all([
    settleResource(Promise.resolve({ count: 3 }), "Stats"),
    settleResource(Promise.reject(new ApiError("not found", 404, 404)), "Billing")
  ]);

  assert.deepEqual(ready, { status: "ready", data: { count: 3 }, failure: null });
  assert.equal(failed.status, "error");
  assert.equal(failed.failure?.kind, "disabled");
});

test("server failures carry the request id for support", () => {
  const failure = describeRequestFailure(new ApiError("internal", 500, 500, null, "rid-500"), "Billing");
  assert.equal(failure.kind, "unknown");
  assert.equal(failure.requestId, "rid-500");
  assert.equal(describeRequestFailure(new ApiError("unauthorized", 401, 401, null, "rid-401"), "Profile").requestId, undefined);
});
