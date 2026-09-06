import assert from "node:assert/strict";
import test from "node:test";
import {
  activeWorkspaceHref,
  isWorkspacePath,
  workspaceNavItems
} from "../lib/workspace";

test("workspace routes are explicit and do not capture public pages", () => {
  for (const path of ["/dashboard", "/account", "/billing", "/referrals", "/billing/history"]) {
    assert.equal(isWorkspacePath(path), true, path);
  }
  for (const path of ["/", "/login", "/pricing", "/docs", "/invite", "/referral-story"]) {
    assert.equal(isWorkspacePath(path), false, path);
  }
});

test("workspace navigation is visible, ordered, and resolves one active destination", () => {
  assert.deepEqual(
    workspaceNavItems.map((item) => item.href),
    ["/dashboard", "/billing", "/referrals", "/account"]
  );
  assert.equal(activeWorkspaceHref("/billing"), "/billing");
  assert.equal(activeWorkspaceHref("/billing/invoices"), "/billing");
  assert.equal(activeWorkspaceHref("/pricing"), "");
});
