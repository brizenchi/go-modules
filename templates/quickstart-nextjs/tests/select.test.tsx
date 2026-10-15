import assert from "node:assert/strict";
import { test } from "node:test";
import { renderToStaticMarkup } from "react-dom/server";
import { Select } from "../components/select";

test("shared select retains native labels, values and disabled semantics with a decorative chevron", () => {
 const html = renderToStaticMarkup(<label htmlFor="period">Date range<Select id="period" name="days" value="30" disabled onChange={() => {}}><option value="7">7 days</option><option value="30">30 days</option></Select></label>);
 assert.match(html, /<select[^>]*id="period"/);
 assert.match(html, /name="days"/);
 assert.match(html, /disabled=""/);
 assert.match(html, /<option(?=[^>]*value="30")(?=[^>]*selected="")[^>]*>/);
 assert.match(html, /<svg[^>]*aria-hidden="true"/);
 assert.doesNotMatch(html, /role="combobox"|role="button"/);
});
