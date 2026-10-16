import assert from "node:assert/strict";
import { test } from "node:test";
import { renderToStaticMarkup } from "react-dom/server";
import { MarkdownContent } from "../components/markdown-content";
import { importMarkdown, markdownBlocks, safeMarkdownHref } from "../lib/markdown";
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
