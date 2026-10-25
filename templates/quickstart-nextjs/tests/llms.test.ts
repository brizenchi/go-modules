import assert from "node:assert/strict";
import { test } from "node:test";
import { llmsText } from "../lib/llms";
import { canonicalURL } from "../lib/seo";
import type { DiscoveryEntry } from "../lib/content-discovery";

const entry = (path: string, kind: DiscoveryEntry["kind"], title: string, summary: string, position = 0): DiscoveryEntry => ({ id: path, path, kind, position, translation: { title, summary } });

test("llms.txt lists published pages by section with canonical URLs", () => {
  const text = llmsText({
    name: "Acme",
    description: "Acme helps\nteams ship.",
    sourceURL: "https://github.com/acme/acme",
    directories: new Set(["/features", "/docs", "/resources", "/about"]),
    entries: [
      entry("/docs/payments", "docs", "Configure [Stripe]", "Plans and webhooks.", 2),
      entry("/docs/overview", "docs", "Overview", "What is included.", 1),
      entry("/features/credits", "features", "Credits", "A credit ledger."),
      entry("/", "page", "Home", "Not a listed page.")
    ]
  });
  assert.ok(text.startsWith("# Acme\n\n> Acme helps teams ship.\n"));
  assert.ok(text.includes(`## Features\n\n- [Credits](${canonicalURL("/features/credits")}): A credit ledger.`));
  assert.ok(text.indexOf("/docs/overview") < text.indexOf("/docs/payments"));
  assert.ok(text.includes(`- [Configure Stripe](${canonicalURL("/docs/payments")})`));
  assert.ok(!text.includes("## Use cases"), "empty sections are omitted");
  assert.ok(!text.includes("Not a listed page"));
  assert.ok(text.includes("- [Source code](https://github.com/acme/acme)"));
  assert.ok(text.endsWith("\n"));
  assert.ok(text.includes(`[About](${canonicalURL("/about")})`));
  assert.ok(!text.includes(canonicalURL("/updates")), "an unpublished page directory is not listed");
});

test("llms.txt omits empty directories and an empty optional section", () => {
  const text = llmsText({ name: "Acme", description: "Acme", entries: [], directories: new Set() });
  assert.ok(text.includes(`[Pricing](${canonicalURL("/pricing")})`));
  assert.ok(!text.includes(canonicalURL("/features")));
  assert.ok(!text.includes("## Optional"));
});
