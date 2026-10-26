import assert from "node:assert/strict";
import { test } from "node:test";
import { renderToStaticMarkup } from "react-dom/server";
import { MarkdownContent } from "../components/markdown-content";
import { importMarkdown, markdownBlocks, markdownImage, safeMarkdownHref } from "../lib/markdown";
import { linkablePath } from "../lib/content-discovery";
test("Markdown renders headings, lists, code and escapes HTML and script links", () => {
 const html = renderToStaticMarkup(<MarkdownContent source={'## Guide\n\n- One\n- Two\n\n```html\n<script>alert(1)</script>\n```\n\n<img src=x onerror=alert(1)>\n\n[Bad](javascript:alert) [Good](https://example.test)'} />);
 assert.match(html, /id="guide"/); assert.match(html, /<ul>/); assert.match(html, /&lt;script&gt;/); assert.match(html, /&lt;img/); assert.doesNotMatch(html, /href="javascript/); assert.match(html, /href="https:\/\/example.test"/);
});
test("Markdown rejects dangerous URLs and imports a title into only the selected draft", () => {
 for (const url of ["javascript:alert(1)","data:text/html,hi","//evil.test","/\\evil.test","https://user:secret@example.test"]) assert.equal(safeMarkdownHref(url), null);
 assert.equal(safeMarkdownHref("/docs/guide#start"), "/docs/guide#start");
 assert.deepEqual(importMarkdown("# A title\r\n\r\nBody"), { title: "A title", body: "Body" });
 assert.throws(() => importMarkdown("字".repeat(50001)));
 assert.deepEqual(markdownBlocks("## Same\n\n## Same").map(block => "id" in block ? block.id : ""), ["same","same-2"]);
});
test("translated Markdown contains complete readable HTML and language-correct internal links without client requests", () => {
 const html = renderToStaticMarkup(<MarkdownContent locale="zh" idPrefix="content-" source={'## 发布流程\n\n在后台保存 **草稿**，再发布当前语言。\n\n[指南](/docs/content#publish) [账户](/account) [外部](https://example.test)'} />);
 assert.match(html, /<h2 id="content-发布流程">发布流程<\/h2>/);
 assert.match(html, /在后台保存 <strong>草稿<\/strong>，再发布当前语言。/);
 assert.match(html, /href="\/zh\/docs\/content#publish"/);
 assert.match(html, /href="\/account"/);
 assert.match(html, /href="https:\/\/example.test"/);
});

test("internal links to pages that are not published read as plain text", () => {
  const linkable = linkablePath(new Set(["/docs/setup"]), new Set(["/docs"]));
  const html = renderToStaticMarkup(<MarkdownContent locale="zh" linkable={linkable} source={"[Setup](/docs/setup) [Removed](/docs/removed) [Blog](/blog) [Docs](/docs#top) [Pricing](/pricing) [Site](https://example.com)"} />);
  assert.match(html, /<a href="\/zh\/docs\/setup">Setup<\/a>/);
  assert.match(html, /<a href="\/zh\/docs#top">Docs<\/a>/);
  assert.match(html, /<a href="\/zh\/pricing">Pricing<\/a>/);
  assert.match(html, /<a href="https:\/\/example.com" rel="noopener noreferrer">Site<\/a>/);
  assert.ok(!html.includes("/docs/removed") && html.includes("Removed"), "unpublished page keeps its text without a link");
  assert.ok(!html.includes('href="/zh/blog"') && html.includes("Blog"), "empty directory keeps its text without a link");
  // Without the rule (editor preview), every safe link renders.
  assert.match(renderToStaticMarkup(<MarkdownContent source="[Removed](/docs/removed)" />), /href="\/docs\/removed"/);
});

test("article images render as lazy figures with known size; unsafe sources render nothing", () => {
  const id = "00000000-0000-4000-8000-000000000000";
  const blocks = markdownBlocks(`Intro\n![Member list](/api/v1/site/images/${id}?w=1200&h=630)\nAfter`);
  assert.deepEqual(blocks.map(block => block.kind), ["paragraph", "image", "paragraph"]);
  assert.deepEqual(markdownImage(`/api/v1/site/images/${id}?w=1200&h=630`, "https://api.example.test"), { src: `https://api.example.test/api/v1/site/images/${id}?w=1200&h=630`, width: 1200, height: 630 });
  assert.deepEqual(markdownImage("https://cdn.example.test/a.png", "https://api.example.test"), { src: "https://cdn.example.test/a.png" });
  for (const unsafe of ["javascript:alert(1)", "http://cdn.example.test/a.png", "/uploads/images/x", `/api/v1/site/images/${id}?w=1&h=1&x=2`, "data:image/png;base64,AAAA", "https://user:pass@cdn.example.test/a.png"]) {
    assert.equal(markdownImage(unsafe, "https://api.example.test"), null, unsafe);
  }
  const html = renderToStaticMarkup(<MarkdownContent source={`![Member list](/api/v1/site/images/${id}?w=1200&h=630)\n\n![Bad](javascript:alert(1))`} />);
  assert.match(html, /<figure class="markdown-figure"><img src="[^"]+\/api\/v1\/site\/images\/[^"]+" alt="Member list" width="1200" height="630" loading="lazy" decoding="async"\/><\/figure>/);
  assert.equal((html.match(/<img/g) ?? []).length, 1);
});
