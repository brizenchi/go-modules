import assert from "node:assert/strict";
import { test } from "node:test";
import { activeAdminSection, adminSectionHref, adminSections } from "../lib/admin-navigation";
import { isWorkspacePath, workspaceNavItems } from "../lib/workspace";

test("administrator sections belong to a separate route space from the customer workspace", () => {
  for (const section of adminSections) {
    const path = adminSectionHref(section);
    assert.match(path, /^\/admin(?:\/|$)/);
    assert.equal(activeAdminSection(path), section);
    assert.equal(isWorkspacePath(path), false);
    assert.equal(workspaceNavItems.some((item) => item.href === path), false);
  }
  for (const path of ["/account", "/billing", "/referrals", "/notes", "/credits", "/files"]) {
    assert.equal(activeAdminSection(path), null);
    assert.equal(isWorkspacePath(path), true);
  }
});

test("admin navigation accepts known sections only and handles query strings", () => {
  assert.equal(activeAdminSection("/admin/credits?user_id=customer#form"), "credits");
  assert.equal(activeAdminSection("/admin/"), "overview");
  for (const path of ["/administrator", "/admin/settings/extra", "/admin/unknown", "//admin", "/"]) assert.equal(activeAdminSection(path), null);
});
