import { canonicalURL } from "./seo";
import { isContentDirectory, uniqueContent, type DiscoveryEntry } from "./content-discovery";

type Section = { title: string; kind?: DiscoveryEntry["kind"]; links?: Array<{ path: string; title: string; summary: string }> };

const sections: Section[] = [
  { title: "Product", links: [
    { path: "/features", title: "Features", summary: "Capabilities included in the product." },
    { path: "/pricing", title: "Pricing", summary: "Plans, one-time purchases and credit packages." },
    { path: "/examples", title: "Live demo", summary: "Try the customer account center." }
  ] },
  { title: "Features", kind: "features" },
  { title: "Use cases", kind: "use-cases" },
  { title: "Comparisons", kind: "compare" },
  { title: "Docs", kind: "docs" },
  { title: "Articles", kind: "blog" }
];

const oneLine = (value: string) => value.replace(/\s+/g, " ").trim();
const label = (value: string) => oneLine(value).replace(/[[\]]/g, "");

/** An llms.txt (llmstxt.org) index of published pages, so AI assistants can cite current URLs. */
export function llmsText(site: { name: string; description: string; entries: readonly DiscoveryEntry[]; directories: ReadonlySet<string>; sourceURL?: string }): string {
  const catalog = uniqueContent(site.entries);
  const shown = (path: string) => !isContentDirectory(path) || site.directories.has(path);
  const lines = [`# ${label(site.name)}`, "", `> ${oneLine(site.description)}`, "", `Pages are available in English and Chinese (${canonicalURL("/zh")}).`];
  for (const section of sections) {
    const links = section.kind
      ? catalog.filter(entry => entry.kind === section.kind).sort((a, b) => (a.position ?? 0) - (b.position ?? 0) || a.path.localeCompare(b.path))
        .map(entry => ({ path: entry.path, title: entry.translation.title, summary: entry.translation.summary }))
      : (section.links || []).filter(link => shown(link.path));
    if (!links.length) continue;
    lines.push("", `## ${section.title}`, "", ...links.map(link => `- [${label(link.title)}](${canonicalURL(link.path)})${link.summary ? `: ${oneLine(link.summary)}` : ""}`));
  }
  const optional = [
    ...[{ path: "/updates", title: "Product updates" }, { path: "/about", title: "About" }].filter(link => shown(link.path)).map(link => `- [${link.title}](${canonicalURL(link.path)})`),
    ...(site.sourceURL ? [`- [Source code](${site.sourceURL})`] : [])
  ];
  if (optional.length) lines.push("", "## Optional", "", ...optional);
  return `${lines.join("\n")}\n`;
}
