import { Fragment, type ReactNode } from "react";
import { markdownBlocks, safeMarkdownHref } from "../lib/markdown";
function inline(text: string): ReactNode[] {
 const parts = text.split(/(\[[^\]\n]+\]\([^\s)]+\)|\*\*[^*\n]+\*\*|`[^`\n]+`)/g);
 return parts.map((part, index) => {
  const link = /^\[([^\]]+)\]\(([^)]+)\)$/.exec(part);
  if (link) { const href = safeMarkdownHref(link[2]); return href ? <a key={index} href={href} {...(/^https?:/.test(href) ? { rel: "noopener noreferrer" } : {})}>{link[1]}</a> : <Fragment key={index}>{link[1]}</Fragment>; }
  if (part.startsWith("**") && part.endsWith("**")) return <strong key={index}>{part.slice(2, -2)}</strong>;
  if (part.startsWith("`") && part.endsWith("`")) return <code key={index}>{part.slice(1, -1)}</code>;
  return <Fragment key={index}>{part}</Fragment>;
 });
}
export function MarkdownContent({ source, idPrefix="" }: { source: string; idPrefix?:string }) {
 return <div className="markdown-content">{markdownBlocks(source).map((block, index) => {
  if (block.kind === "heading") { const Tag = `h${block.level}` as "h2" | "h3" | "h4" | "h5" | "h6"; return <Tag id={`${idPrefix}${block.id}`} key={index}>{inline(block.text)}</Tag>; }
  if (block.kind === "list") { const Tag = block.ordered ? "ol" : "ul"; return <Tag key={index}>{block.items.map((item, n) => <li key={n}>{inline(item)}</li>)}</Tag>; }
  if (block.kind === "code") return <pre key={index}><code>{block.text}</code></pre>;
  if (block.kind === "quote") return <blockquote key={index}>{inline(block.text)}</blockquote>;
  return <p key={index}>{inline(block.text)}</p>;
 })}</div>;
}
