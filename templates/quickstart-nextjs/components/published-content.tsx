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
import styles from "./content/marketing.module.css";
const names = { blog: { en: "Articles", zh: "产品文章" }, docs: { en: "Documentation", zh: "使用文档" }, features: { en: "Features", zh: "功能介绍" }, "use-cases": { en: "Use cases", zh: "使用场景" }, compare: { en: "Comparisons", zh: "方案比较" }, resources: { en: "Resources", zh: "资源" } };
export function PublishedContentIndex({ kind, entries }: { kind: keyof typeof names; entries: PublishedArticle[] }) {
 const { t, locale } = useI18n(); const [query, setQuery] = useState(""); const rows = entries.filter(entry => (kind === "resources" || entry.kind === kind) && `${entry.translation.title} ${entry.translation.summary}`.toLocaleLowerCase().includes(query.trim().toLocaleLowerCase()));
 const title = t(names[kind]);
 return <SiteShell variant="compact" eyebrow={title} title={title} description={t({ en: "Explore product capabilities and practical guides.", zh: "了解产品能力，查阅使用指南与实用文章。" })}>
  {kind === "features" ? <div className={styles.capabilities}>{getHomeContent(appEnv.demoMode)[locale].features.map(feature => <section className={styles.capability} key={feature.label}><span>{feature.label} / {feature.detail}</span><h2>{feature.title}</h2><p>{feature.description}</p>{entries.some(entry => entry.path === feature.href) || ["/resources", "/docs", "/features"].includes(feature.href) ? <Link href={feature.href}>{t({ en: "Learn more", zh: "了解详情" })} →</Link> : null}</section>)}</div> : null}
  <div className="published-search"><label htmlFor="published-search">{t({ en: "Find an article", zh: "查找文章" })}</label><input id="published-search" type="search" value={query} onChange={event => setQuery(event.target.value)} placeholder={t({ en: "Search titles and summaries", zh: "搜索标题和摘要" })} /><span role="status">{rows.length} {t({ en: "articles", zh: "篇" })}</span></div>
  <div className={styles.indexList}>{rows.map(entry => <article className={styles.indexItem} key={entry.id}><div className={styles.itemCopy}><span className={styles.sectionLabel}>{t(names[entry.kind])}</span><h2><Link href={entry.path}>{entry.translation.title}</Link></h2><p>{entry.translation.summary}</p><Link className={styles.textLink} href={entry.path}>{t({ en: "Read article", zh: "阅读全文" })} →</Link></div></article>)}</div>
  {!rows.length ? <p role="status">{t({ en: "No published articles match this search.", zh: "暂无匹配的已发布文章。" })}</p> : null}
 </SiteShell>;
}
export function PublishedContentArticle({ article }: { article: PublishedArticle }) {
 const { t, locale } = useI18n(); const text = article.translation;
 return <ArticleLanguagesProvider locales={article.locales}><SiteShell variant="article" eyebrow={t(names[article.kind])} title={text.title} description={text.summary} breadcrumbs={[{ href: "/", label: t({ en: "Home", zh: "首页" }) }, { href: `/${article.kind}`, label: t(names[article.kind]) }, { label: text.title }]}>
  <StructuredData data={articleStructuredData({ title: text.title, description: text.summary, path: localizeHref(article.path, locale), language: locale === "zh" ? "zh-CN" : "en", publishedAt: article.published_at })} />
  <div className="published-article" id="details"><MarkdownContent source={text.body} idPrefix="content-" /><footer><Link href={`/${article.kind}`}>← {t(names[article.kind])}</Link><Link href="/contact">{t({ en: "Help & support", zh: "帮助与支持" })} →</Link></footer></div>
 </SiteShell></ArticleLanguagesProvider>;
}
