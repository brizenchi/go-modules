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
