import assert from "node:assert/strict";
import { test } from "node:test";
import { ApiError } from "../lib/api";
import { describeRequestFailure, settleResource } from "../lib/request-state";

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
