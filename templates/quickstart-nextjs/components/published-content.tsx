"use client";
import { useState } from "react";
import { LocalizedLink as Link } from "./localized-link";
import type { PublishedArticle } from "@/lib/api";
import { useI18n } from "@/lib/i18n";
import { localizeHref } from "@/lib/locale";
import { SiteShell } from "./site-shell";
import { MarkdownContent } from "./markdown-content";
import { markdownBlocks } from "@/lib/markdown";
import { ArticleLanguagesProvider } from "./article-language-context";
import { StructuredData } from "./structured-data";
import { articleStructuredData } from "@/lib/structured-data";
import { siteLinks } from "@/content/navigation";
import { filterGuideIndex, type GuideCategory, type GuideIndexEntry } from "@/lib/guide-index";
import type { DiscoveryEntry } from "@/lib/content-discovery";
import { GuideDirectories, ProductNavigation, RelatedContent } from "./content-discovery";
import listStyles from "./published-content.module.css";
const names = { page: { en: "Page", zh: "页面" }, blog: { en: "Articles", zh: "产品文章" }, docs: { en: "Documentation", zh: "使用文档" }, features: { en: "Features", zh: "功能介绍" }, "use-cases": { en: "Use cases", zh: "使用场景" }, compare: { en: "Comparisons", zh: "方案比较" }, resources: siteLinks.resources.label };
const descriptions = {
 blog: { en: "Practical answers to the questions around your product and account.", zh: "围绕产品与账户问题，查阅实用文章与操作建议。" },
 docs: { en: "Set up your site, connect your services and prepare to launch.", zh: "从站点配置到服务接入，完成上线前的准备。" },
 features: { en: "Explore the authentication, billing and account foundations already connected.", zh: "了解已经接通的登录、计费与账户基础能力。" },
 "use-cases": { en: "Find a starting workflow for the way your product delivers value and charges customers.", zh: "根据产品的交付与收费方式，选择合适的起步流程。" },
 compare: { en: "Compare development approaches and choose the starting point that fits your product.", zh: "比较不同开发方式，选择适合自己产品的起点。" },
 resources: { en: "Practical guides to get started, use your account and make the most of the product.", zh: "从入门配置到日常使用，找到你需要的教程与实践方法。" }
};
export function PublishedContentIndex({ kind, entries, page }: { kind: Exclude<keyof typeof names, "page">; entries: GuideIndexEntry[]; page?: PublishedArticle }) {
 const { t, locale } = useI18n();
 const [query, setQuery] = useState("");
 const [category, setCategory] = useState<GuideCategory>("all");
 const rows = filterGuideIndex(entries, kind, category, query);
 const title = page?.translation.title || t(names[kind]);
 return <ArticleLanguagesProvider locales={page?.locales || ["en", "zh"]}><SiteShell variant="compact" eyebrow={title} title={title} description={page?.translation.summary || t(descriptions[kind])}>
  {kind === "resources" ? <GuideDirectories /> : ["features", "use-cases", "compare"].includes(kind) ? <ProductNavigation currentPath={`/${kind}`} /> : null}
  {page ? <div className={listStyles.introduction}><MarkdownContent source={page.translation.body} locale={locale} /></div> : null}
  <section className={listStyles.library} aria-label={title}>
   <div className={listStyles.toolbar}>
    {kind === "resources" ? <div className={listStyles.filters} role="group" aria-label={t({ en: "Guide category", zh: "指南分类" })}>
     {(["all", "docs", "blog"] as const).map(value => <button key={value} type="button" aria-pressed={category === value} onClick={() => setCategory(value)}>{t(value === "all" ? { en: "All guides", zh: "全部指南" } : names[value])}</button>)}
    </div> : <span className={listStyles.indexLabel}>{title}</span>}
    <label className={listStyles.search} htmlFor="published-search">
     <span className="sr-only">{t({ en: "Find an article", zh: "查找文章" })}</span>
     <svg viewBox="0 0 20 20" aria-hidden="true"><circle cx="8.5" cy="8.5" r="5.5" /><path d="m13 13 4 4" /></svg>
     <input id="published-search" type="search" value={query} onChange={event => setQuery(event.target.value)} placeholder={t({ en: "Search titles and summaries", zh: "搜索标题和摘要" })} />
    </label>
   </div>
   <p className={listStyles.count} role="status" aria-live="polite">{t({ en: `${rows.length} ${rows.length === 1 ? "article" : "articles"}`, zh: `共 ${rows.length} 篇` })}</p>
   <ul className={listStyles.list}>
    {rows.map(entry => <li key={entry.id}>
     <article className={listStyles.entry}>
      <div className={listStyles.copy}>
       <span className={listStyles.category}>{t(names[entry.kind])}</span>
       <h2><Link href={entry.path}>{entry.translation.title}</Link></h2>
       <p>{entry.translation.summary}</p>
      </div>
      <Link className={listStyles.readLink} href={entry.path} aria-label={t({ en: `Read article: ${entry.translation.title}`, zh: `阅读全文：${entry.translation.title}` })}>
       {t({ en: "Read article", zh: "阅读全文" })}<span aria-hidden="true">→</span>
      </Link>
     </article>
    </li>)}
   </ul>
   {!rows.length ? <p className={listStyles.empty}>{t({ en: "No published articles match this search.", zh: "暂无匹配的已发布文章。" })}</p> : null}
  </section>
 </SiteShell></ArticleLanguagesProvider>;
}
export function PublishedContentArticle({ article, entries }: { article: PublishedArticle; entries: DiscoveryEntry[] }) {
 const { t, locale } = useI18n(); const text = article.translation;
 const toc = markdownBlocks(text.body).flatMap(block => block.kind === "heading" && block.level === 2 ? [{ id: `content-${block.id}`, label: block.text }] : []);
 return <ArticleLanguagesProvider locales={article.locales}><SiteShell variant="article" eyebrow={t(names[article.kind])} title={text.title} description={text.summary} breadcrumbs={[{ href: "/", label: t({ en: "Home", zh: "首页" }) }, { href: `/${article.kind}`, label: t(names[article.kind]) }, { label: text.title }]}>
  {["features", "use-cases", "compare"].includes(article.kind) ? <ProductNavigation currentPath={article.path} /> : null}
  <StructuredData data={articleStructuredData({ title: text.title, description: text.summary, path: localizeHref(article.path, locale), language: locale === "zh" ? "zh-CN" : "en", publishedAt: article.published_at })} />
  <div className="published-article" id="details">{toc.length > 2 ? <details className={listStyles.outline}><summary>{t({ en: "On this page", zh: "本页目录" })}</summary><nav aria-label={t({ en: "On this page", zh: "本页目录" })}><ol>{toc.map(item => <li key={item.id}><a href={`#${item.id}`}>{item.label}</a></li>)}</ol></nav></details> : null}<MarkdownContent locale={locale} source={text.body} idPrefix="content-" /><footer><Link href={`/${article.kind}`}>← {t(names[article.kind])}</Link><Link href="/contact">{t({ en: "Help & support", zh: "帮助与支持" })} →</Link></footer></div>
  <RelatedContent entries={entries} currentPath={article.path} />
 </SiteShell></ArticleLanguagesProvider>;
}
