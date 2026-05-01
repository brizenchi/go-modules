import assert from "node:assert/strict";
import { test } from "node:test";
import {
  DEFAULT_LOCALE,
  humanizeSegment,
  normalizeLocale,
  pickLocaleText
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
