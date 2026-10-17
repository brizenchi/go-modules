import assert from "node:assert/strict";
import test from "node:test";
import { createPageReturnRefresh, PAGE_RETURN_STALE_MS } from "../lib/page-return";

test("brief screen switches and initial focus do not refresh account data", () => {
  let now = 0;
  const refresh = createPageReturnRefresh(() => now);
  assert.equal(refresh.shouldRefresh(), false);
  for (const duration of [1, 120, 5_000, PAGE_RETURN_STALE_MS - 1]) {
    refresh.leave();
    now += duration;
    assert.equal(refresh.shouldRefresh(), false, String(duration));
  }
});

test("returning after a minute refreshes once across focus and visibility events", () => {
  let now = 0;
  const refresh = createPageReturnRefresh(() => now);
  refresh.leave();
  now = 20_000;
  refresh.leave();
  now = PAGE_RETURN_STALE_MS;
  assert.equal(refresh.shouldRefresh(), true);
  assert.equal(refresh.shouldRefresh(), false);
  now += PAGE_RETURN_STALE_MS;
  assert.equal(refresh.shouldRefresh(), false, "remaining on the page does not poll");
});

test("each absence is measured separately without accumulating brief switches", () => {
  let now = 0;
  const refresh = createPageReturnRefresh(() => now);
  for (let index = 0; index < 5; index++) {
    refresh.leave();
    now += 30_000;
    assert.equal(refresh.shouldRefresh(), false);
  }
  refresh.leave();
  now += PAGE_RETURN_STALE_MS;
  assert.equal(refresh.shouldRefresh(), true);
  refresh.leave();
  now += 100;
  assert.equal(refresh.shouldRefresh(), false);
});
