import assert from "node:assert/strict";
import test from "node:test";
import { dialogFocusTarget } from "../lib/dialog-focus";

test("dialog focus wraps at either boundary and recovers an escaped focus", () => {
  assert.equal(dialogFocusTarget(2, 3, false), 0);
  assert.equal(dialogFocusTarget(0, 3, true), 2);
  assert.equal(dialogFocusTarget(1, 3, false), null);
  assert.equal(dialogFocusTarget(1, 3, true), null);
  assert.equal(dialogFocusTarget(-1, 3, false), 0);
  assert.equal(dialogFocusTarget(-1, 3, true), 2);
  assert.equal(dialogFocusTarget(0, 1, true), 0);
  assert.equal(dialogFocusTarget(-1, 0, false), null);
});
