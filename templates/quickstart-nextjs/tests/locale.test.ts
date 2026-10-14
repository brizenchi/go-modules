import assert from "node:assert/strict";
import { test } from "node:test";
import {
  DEFAULT_LOCALE,
  humanizeSegment,
  isPublicPath,
  localeFromPath,
  localizeHref,
  normalizeLocale,
  pickLocaleText,
  resolvePageLocale,
  stripLocalePrefix
} from "../lib/locale";

test("normalizeLocale maps zh variants and falls back to default", () => {
  assert.equal(normalizeLocale("zh-CN"), "zh");
  assert.equal(normalizeLocale("zh-TW"), "zh");
  assert.equal(normalizeLocale("en-US"), "en");
  assert.equal(normalizeLocale(""), DEFAULT_LOCALE);
  assert.equal(normalizeLocale(undefined), DEFAULT_LOCALE);
});

test("pickLocaleText selects the matching localized value", () => {
  assert.equal(
    pickLocaleText("en", { en: "Pricing", zh: "价格" }),
    "Pricing"
  );
  assert.equal(
    pickLocaleText("zh", { en: "Pricing", zh: "价格" }),
    "价格"
  );
});

test("humanizeSegment turns route fragments into labels", () => {
  assert.equal(humanizeSegment("quickstart-nextjs"), "Quickstart Nextjs");
  assert.equal(humanizeSegment("settings"), "Settings");
});

test("public language routes keep a single prefix and preserve query strings and anchors", () => {
  assert.equal(localizeHref("/pricing?source=guide#plans", "zh"), "/zh/pricing?source=guide#plans");
  assert.equal(localizeHref("/zh/pricing?source=guide#plans", "zh"), "/zh/pricing?source=guide#plans");
  assert.equal(localizeHref("/zh/pricing?source=guide#plans", "en"), "/pricing?source=guide#plans");
  assert.equal(localizeHref("/", "zh"), "/zh");
  assert.equal(stripLocalePrefix("/zh"), "/");
  assert.equal(stripLocalePrefix("/zh/docs/domains#configuration"), "/docs/domains#configuration");
  assert.equal(localeFromPath("/zh/docs"), "zh");
  assert.equal(localeFromPath("/zhish"), "en");
});

test("localization leaves private, external, asset and in-page links alone", () => {
  for (const href of ["/billing?product=credits#credits", "/login", "/admin/users", "/api/auth", "/opengraph-image", "https://example.test/guide", "//example.test/guide", "mailto:help@example.test", "#details"]) {
    assert.equal(localizeHref(href, "zh"), href);
  }
  assert.equal(isPublicPath("/zh/features/authentication"), true);
  assert.equal(isPublicPath("/zh/notes"), false);
});

test("public URLs determine the initial language while private pages retain a saved preference", () => {
  assert.equal(resolvePageLocale("/pricing", "zh"), "en");
  assert.equal(resolvePageLocale("/zh/pricing", "en"), "zh");
  assert.equal(resolvePageLocale("/billing", "zh"), "zh");
  assert.equal(resolvePageLocale("/account", "en"), "en");
});
