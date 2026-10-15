import assert from "node:assert/strict";
import { test } from "node:test";
import { browserLocaleStorage, LOCALE_STORAGE_KEY, readLocalePreference, saveLocalePreference } from "../lib/locale-preference";
import { localeFromPath, resolvePageLocale } from "../lib/locale";

function fixture() {
  const values = new Map<string, string>();
  return { values, storage: { getItem: (key: string) => values.get(key) ?? null, setItem: (key: string, value: string) => { values.set(key, value); } } };
}

test("no saved language defaults to English and server rendering needs no browser storage", () => {
  assert.equal(readLocalePreference(fixture().storage), "en");
  assert.equal(readLocalePreference(null), "en");
  assert.equal(browserLocaleStorage(), null);
});

test("explicit language selection survives reloads and private page transitions", () => {
  const { storage } = fixture();
  for (const choice of ["zh", "en"] as const) {
    saveLocalePreference(choice, storage);
    for (const page of ["/account", "/billing", "/referrals", "/admin/users"]) {
      assert.equal(resolvePageLocale(page, readLocalePreference(storage)), choice);
    }
  }
});

test("the active public URL can replace stale private preferences without changing indexed content", () => {
  const { storage, values } = fixture();
  values.set(LOCALE_STORAGE_KEY, "zh");
  assert.equal(resolvePageLocale("/pricing", readLocalePreference(storage)), "en");
  saveLocalePreference(localeFromPath("/pricing"), storage);
  assert.equal(resolvePageLocale("/account", readLocalePreference(storage)), "en");
  saveLocalePreference(localeFromPath("/zh/docs"), storage);
  assert.equal(resolvePageLocale("/account", readLocalePreference(storage)), "zh");
});

test("blocked persistence preserves the current in-memory choice and malformed data defaults to English", () => {
  const blocked = { getItem: () => { throw new Error("blocked"); }, setItem: () => { throw new Error("blocked"); } };
  assert.equal(readLocalePreference(blocked), "en");
  assert.equal(readLocalePreference(blocked, "zh"), "zh");
  assert.doesNotThrow(() => saveLocalePreference("zh", blocked));
  const { storage, values } = fixture();
  values.set(LOCALE_STORAGE_KEY, "invalid-value");
  assert.equal(readLocalePreference(storage), "en");
});
