import assert from "node:assert/strict";
import { test } from "node:test";
import { renderToStaticMarkup } from "react-dom/server";
import { NavigationFeedback } from "../components/navigation-feedback";

test("pending navigation announces its destination without removing the link label", () => {
  const html = renderToStaticMarkup(<NavigationFeedback pending label="Subscription" pendingLabel="Opening Subscription…" />);
  assert.match(html, /Subscription/);
  assert.match(html, /role="status"/);
  assert.match(html, /Opening Subscription…/);
  assert.match(html, /data-navigation-pending="true"/);
});

test("idle or canceled navigation clears its busy announcement", () => {
  const html = renderToStaticMarkup(<NavigationFeedback pending={false} label="订阅与账单" pendingLabel="正在打开订阅与账单…" />);
  assert.match(html, /订阅与账单/);
  assert.doesNotMatch(html, /role="status"|正在打开|data-navigation-pending="true"/);
});
