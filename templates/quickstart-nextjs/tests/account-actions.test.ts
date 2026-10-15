import assert from "node:assert/strict";
import test from "node:test";
import { ApiError } from "../lib/api";
import { accountLogoutFailure } from "../lib/account-actions";

test("account sign-out failures hide server details and preserve support request IDs", () => {
  for (const locale of ["zh", "en"] as const) {
    const server = accountLogoutFailure(new ApiError("private provider detail", 503, 503, null, "request-example"), locale);
    assert.ok(server.includes("request-example"));
    assert.ok(!server.includes("private provider detail"));
    const forbidden = accountLogoutFailure(new ApiError("internal restriction", 403, 403, null, "request-example"), locale);
    assert.ok(!forbidden.includes("internal restriction"));
    assert.ok(!forbidden.includes("request-example"));
    const network = accountLogoutFailure(new Error("secret network detail"), locale);
    assert.ok(!network.includes("secret network detail"));
    assert.ok(network.length > 0);
  }
});
