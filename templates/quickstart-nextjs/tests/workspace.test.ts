import assert from "node:assert/strict";
import test from "node:test";
import {
  activeWorkspaceHref,
  auxiliaryWorkspaceItems,
  isWorkspacePath,
  workspaceNavItems
} from "../lib/workspace";

test("personal account routes do not capture public pages or administrator routes", () => {
  for (const path of ["/account", "/billing", "/referrals", "/billing/history", "/credits", "/notes", "/files"]) {
    assert.equal(isWorkspacePath(path), true, path);
  }
  for (const path of ["/", "/dashboard", "/login", "/pricing", "/docs", "/invite", "/referral-story", "/admin", "/admin/users", "/admin/settings", "/admin?tab=users", "/admin/credits#grant"]) {
    assert.equal(isWorkspacePath(path), false, path);
  }
});

test("workspace navigation is visible, ordered, and resolves one active destination", () => {
  assert.deepEqual(
    workspaceNavItems.map((item) => item.href),
    ["/account", "/billing", "/referrals"]
  );
  assert.equal(activeWorkspaceHref("/billing"), "/billing");
  assert.equal(activeWorkspaceHref("/billing/invoices"), "/billing");
  assert.equal(activeWorkspaceHref("/pricing"), "");
  assert.equal(activeWorkspaceHref("/admin/subscriptions"), "");
});

test("auxiliary personal features keep route recognition without expanding account navigation", () => {
  for (const item of auxiliaryWorkspaceItems) {
    assert.equal(activeWorkspaceHref(`${item.href}?page=2`), item.href);
    assert.equal(activeWorkspaceHref(`${item.href}/details`), item.href);
    assert.ok(!workspaceNavItems.some((entry) => entry.href === item.href));
  }
});
