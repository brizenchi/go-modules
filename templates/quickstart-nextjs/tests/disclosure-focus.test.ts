import assert from "node:assert/strict";
import test from "node:test";
import { disclosureFocusIndex } from "../lib/disclosure-focus";

test("account disclosure keyboard navigation wraps and leaves Tab and activation native", () => {
  assert.equal(disclosureFocusIndex("ArrowDown", -1, 4), 0);
  assert.equal(disclosureFocusIndex("ArrowUp", -1, 4), 3);
  assert.equal(disclosureFocusIndex("ArrowDown", 3, 4), 0);
  assert.equal(disclosureFocusIndex("ArrowUp", 0, 4), 3);
  assert.equal(disclosureFocusIndex("Home", 2, 4), 0);
  assert.equal(disclosureFocusIndex("End", 0, 4), 3);
  for (const key of ["Tab", "Enter", "Escape", " "]) assert.equal(disclosureFocusIndex(key, 0, 4), null);
  assert.equal(disclosureFocusIndex("ArrowDown", 0, 0), null);
});
