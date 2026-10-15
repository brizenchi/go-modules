export type MarkdownBlock = { kind: "heading"; level: number; text: string; id: string } | { kind: "paragraph" | "quote" | "code"; text: string } | { kind: "list"; ordered: boolean; items: string[] };
export function markdownBlocks(source: string): MarkdownBlock[] {
 const lines = source.replace(/\r\n?/g, "\n").split("\n"); const blocks: MarkdownBlock[] = []; const used = new Set<string>();
 for (let i = 0; i < lines.length;) {
  const line = lines[i]; if (!line.trim()) { i++; continue; }
  if (/^\s*```/.test(line)) { const text: string[] = []; i++; while (i < lines.length && !/^\s*```/.test(lines[i])) text.push(lines[i++]); i++; blocks.push({ kind: "code", text: text.join("\n") }); continue; }
  const heading = /^(#{1,6})\s+(.+)$/.exec(line);
  if (heading) { const base = heading[2].toLowerCase().replace(/[^\p{L}\p{N}]+/gu, "-").replace(/^-|-$/g, "") || "section"; let id=base,n=1;while(used.has(id))id=`${base}-${++n}`;used.add(id);blocks.push({kind:"heading",level:Math.max(2,heading[1].length),text:heading[2],id}); i++; continue; }
  const list = /^\s*(?:([-*])|\d+[.)])\s+(.+)$/.exec(line);
  if (list) { const ordered = !list[1]; const items: string[] = []; while (i < lines.length) { const match = /^\s*(?:([-*])|\d+[.)])\s+(.+)$/.exec(lines[i]); if (!match || !match[1] !== ordered) break; items.push(match[2]); i++; } blocks.push({ kind: "list", ordered, items }); continue; }
  if (/^>\s?/.test(line)) { const text: string[] = []; while (i < lines.length && /^>/.test(lines[i])) text.push(lines[i++].replace(/^>\s?/, "")); blocks.push({ kind: "quote", text: text.join(" ") }); continue; }
  const text = [line]; i++; while (i < lines.length && lines[i].trim() && !/^(#{1,6}\s|\s*```|>\s?|\s*[-*]\s|\s*\d+[.)]\s)/.test(lines[i])) text.push(lines[i++]); blocks.push({ kind: "paragraph", text: text.join(" ") });
 }
 return blocks;
}
export function safeMarkdownHref(raw: string): string | null {
 const value = raw.trim(); if (!value || /[\u0000-\u0020\u007f\\]/.test(value)) return null;
 if (value.startsWith("#")) return value;
 if (value.startsWith("/") && !value.startsWith("//")) return value;
 try { const url = new URL(value); return ["https:", "http:", "mailto:"].includes(url.protocol) && !url.username && !url.password ? value : null; } catch { return null; }
}
export function importMarkdown(source: string) {
 if (new TextEncoder().encode(source).length > 150000) throw new Error("file_too_large");
 const normalized = source.replace(/^\uFEFF/, "").replace(/\r\n?/g, "\n"); const match = /^#\s+([^\n]+)\n?/.exec(normalized);
 return { title: match?.[1]?.trim() || "", body: match ? normalized.slice(match[0].length).trimStart() : normalized };
}
