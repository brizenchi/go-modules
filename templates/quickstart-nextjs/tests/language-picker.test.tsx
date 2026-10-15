import assert from "node:assert/strict";
import { test } from "node:test";
import { renderToStaticMarkup } from "react-dom/server";
import { LanguagePicker } from "../components/language-picker";

test("a single language disclosure retains crawlable alternatives in server HTML", () => {
  const html = renderToStaticMarkup(<LanguagePicker locale="zh" label="选择语言" destinations={{ en: "/features/credits", zh: "/zh/features/credits" }} onSelect={() => {}} />);
  assert.equal((html.match(/<summary\b/g) || []).length, 1);
  assert.match(html, /aria-label="选择语言: 中文"/);
  assert.match(html, /href="\/features\/credits"/);
  assert.match(html, /hrefLang="zh-CN"/);
  assert.match(html, /aria-current="page"/);
  assert.equal((html.match(/<a\b/g) || []).length, 2);
  assert.ok(!html.includes("<details open"));
});

test("private workspace language choices change preference without translated workspace URLs", () => {
  const html = renderToStaticMarkup(<LanguagePicker locale="en" label="Choose language" onSelect={() => {}} />);
  assert.equal((html.match(/<summary\b/g) || []).length, 1);
  assert.equal((html.match(/<button\b/g) || []).length, 2);
  assert.match(html, /aria-label="Choose language: EN"/);
  assert.match(html, /<span lang="en">EN<\/span>/);
  assert.ok(!html.includes("English"));
  assert.match(html, /aria-pressed="true"/);
  assert.ok(!html.includes("href="));
});

test("both languages use a decorative vector chevron without a font-dependent baseline", () => {
  for (const locale of ["en", "zh"] as const) {
    const html = renderToStaticMarkup(<LanguagePicker locale={locale} label="Language" onSelect={() => {}} />);
    const trigger = html.match(/<summary\b[^>]*>([\s\S]*?)<\/summary>/)?.[1] || "";
    assert.match(trigger, /<svg\b[^>]*class="language-chevron"[^>]*aria-hidden="true"/);
    assert.doesNotMatch(trigger, /⌄/u);
  }
});
