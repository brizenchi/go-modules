"use client";
import { useState } from "react";
import { LocalizedLink as Link } from "./localized-link";
import type { PublishedArticle } from "@/lib/api";
import { useI18n } from "@/lib/i18n";
import { localizeHref } from "@/lib/locale";
import { SiteShell } from "./site-shell";
import { MarkdownContent } from "./markdown-content";
import { ArticleLanguagesProvider } from "./article-language-context";
import { StructuredData } from "./structured-data";
import { articleStructuredData } from "@/lib/structured-data";
import { getHomeContent } from "@/content/site";
import { appEnv } from "@/lib/env";
import { siteLinks } from "@/content/navigation";
import styles from "./content/marketing.module.css";
import listStyles from "./published-content.module.css";
const names = { blog: { en: "Articles", zh: "产品文章" }, docs: { en: "Documentation", zh: "使用文档" }, features: { en: "Features", zh: "功能介绍" }, "use-cases": { en: "Use cases", zh: "使用场景" }, compare: { en: "Comparisons", zh: "方案比较" }, resources: siteLinks.resources.label };
export function PublishedContentIndex({ kind, entries }: { kind: keyof typeof names; entries: PublishedArticle[] }) {
 const { t, locale } = useI18n(); const [query, setQuery] = useState(""); const rows = entries.filter(entry => (kind === "resources" || entry.kind === kind) && `${entry.translation.title} ${entry.translation.summary}`.toLocaleLowerCase().includes(query.trim().toLocaleLowerCase()));
 const title = t(names[kind]);
 return <SiteShell variant="compact" eyebrow={title} title={title} description={t({ en: "Explore product capabilities and practical guides.", zh: "了解产品能力，查阅使用指南与实用文章。" })}>
  {kind === "features" ? <div className={styles.capabilities}>{getHomeContent(appEnv.demoMode)[locale].features.map(feature => <section className={styles.capability} key={feature.label}><span>{feature.label} / {feature.detail}</span><h2>{feature.title}</h2><p>{feature.description}</p>{entries.some(entry => entry.path === feature.href) || ["/resources", "/docs", "/features"].includes(feature.href) ? <Link href={feature.href}>{t({ en: "Learn more", zh: "了解详情" })} →</Link> : null}</section>)}</div> : null}
  <section className={listStyles.library} aria-label={title}>
   <div className={listStyles.toolbar}>
    <label className={listStyles.search} htmlFor="published-search">
     <span>{t({ en: "Find an article", zh: "查找文章" })}</span>
     <input id="published-search" type="search" value={query} onChange={event => setQuery(event.target.value)} placeholder={t({ en: "Search titles and summaries", zh: "搜索标题和摘要" })} />
    </label>
    <span className={listStyles.count} role="status" aria-live="polite">{t({ en: `${rows.length} ${rows.length === 1 ? "article" : "articles"}`, zh: `共 ${rows.length} 篇` })}</span>
   </div>
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
 </SiteShell>;
}
export function PublishedContentArticle({ article }: { article: PublishedArticle }) {
 const { t, locale } = useI18n(); const text = article.translation;
 return <ArticleLanguagesProvider locales={article.locales}><SiteShell variant="article" eyebrow={t(names[article.kind])} title={text.title} description={text.summary} breadcrumbs={[{ href: "/", label: t({ en: "Home", zh: "首页" }) }, { href: `/${article.kind}`, label: t(names[article.kind]) }, { label: text.title }]}>
  <StructuredData data={articleStructuredData({ title: text.title, description: text.summary, path: localizeHref(article.path, locale), language: locale === "zh" ? "zh-CN" : "en", publishedAt: article.published_at })} />
  <div className="published-article" id="details"><MarkdownContent source={text.body} idPrefix="content-" /><footer><Link href={`/${article.kind}`}>← {t(names[article.kind])}</Link><Link href="/contact">{t({ en: "Help & support", zh: "帮助与支持" })} →</Link></footer></div>
 </SiteShell></ArticleLanguagesProvider>;
}
