"use client";

import Link from "next/link";
import { useState } from "react";
import { SiteShell } from "@/components/site-shell";
import { articles, formatContentDate, searchArticles, type Article } from "@/content/articles";
import { useI18n } from "@/lib/i18n";
import styles from "./content.module.css";

export function BlogIndex() {
  const { t, locale } = useI18n();
  const [query, setQuery] = useState("");
  const matches = searchArticles(query, locale);

  return (
    <SiteShell
      eyebrow={t({ en: "The product journal", zh: "产品手记" })}
      title={t({ en: "Build it. Launch it. Keep improving.", zh: "从开始构建，到持续经营。" })}
      description={t({ en: "Practical guides to make the starter your own, understand invitations, and support the people using your product.", zh: "从模板配置到邀请增长，再到内容与支持，帮助你把产品真正用起来。" })}
      sideTitle={t({ en: "A useful place to start", zh: "从这里开始" })}
      showEnvironment={false}
      sideBody={<p>{t({ en: "Short guides, written around a task. Switch languages at any time from the navigation.", zh: "围绕具体操作编写的简短指南。可以随时从导航栏切换语言。" })}</p>}
      breadcrumbs={[{ href: "/", label: t({ en: "Home", zh: "首页" }) }, { label: t({ en: "Blog", zh: "博客" }) }]}
      toc={[{ id: "articles", label: t({ en: "All guides", zh: "全部指南" }) }]}
      actions={<Link className="button" href="/updates">{t({ en: "See product updates", zh: "查看产品更新" })}</Link>}
    >
      <section id="articles" aria-label={t({ en: "Articles", zh: "文章" })}>
        <div className={styles.toolbar}>
          <label className={styles.search}>
            {t({ en: "Find a guide", zh: "搜索指南" })}
            <input type="search" value={query} onChange={(event) => setQuery(event.target.value)} placeholder={t({ en: "Search a topic or a question…", zh: "搜索主题或问题…" })} />
          </label>
          <div className={styles.resultCount} role="status" aria-live="polite">{t({ en: `${matches.length} of ${articles.length} guides`, zh: `共 ${articles.length} 篇，找到 ${matches.length} 篇` })}</div>
        </div>
        <div className={styles.list}>
          {matches.map((article) => (
            <Link className={styles.articleLink} href={`/blog/${article.slug}`} key={article.slug}>
              <div className={styles.meta}><span className={styles.category}>{t(article.category)}</span><time dateTime={article.publishedAt}>{formatContentDate(article.publishedAt, locale)}</time></div>
              <div><h2>{t(article.title)}</h2><p>{t(article.summary)}</p></div>
              <span className={styles.arrow} aria-hidden="true">↗</span>
            </Link>
          ))}
        </div>
        {matches.length === 0 ? <div className={styles.empty}><h2>{t({ en: "No guides found", zh: "没有找到相关指南" })}</h2><p>{t({ en: "Try a shorter search or browse all guides.", zh: "试试更简短的关键词，或者浏览全部指南。" })}</p><button className="button" type="button" onClick={() => setQuery("")}>{t({ en: "Clear search", zh: "清除搜索" })}</button></div> : null}
      </section>
    </SiteShell>
  );
}

export function BlogArticle({ article }: { article: Article }) {
  const { t, locale } = useI18n();
  return (
    <SiteShell
      eyebrow={t(article.category)} title={t(article.title)} description={t(article.summary)}
      sideTitle={t({ en: "Guide details", zh: "指南信息" })} showEnvironment={false}
      sideBody={<div className={styles.meta}><span>{t({ en: "Published", zh: "发布日期" })}</span><time dateTime={article.publishedAt}>{formatContentDate(article.publishedAt, locale)}</time><Link href="/blog">{t({ en: "← All guides", zh: "← 全部指南" })}</Link></div>}
      breadcrumbs={[{ href: "/", label: t({ en: "Home", zh: "首页" }) }, { href: "/blog", label: t({ en: "Blog", zh: "博客" }) }, { label: t(article.title) }]}
      toc={article.sections.map((section) => ({ id: section.id, label: t(section.title) }))}
    >
      <article className={styles.body}>
        {article.sections.map((section) => <section className={styles.section} id={section.id} key={section.id}><h2>{t(section.title)}</h2>{section.paragraphs.map((paragraph, index) => <p key={index}>{t(paragraph)}</p>)}</section>)}
        <div className={styles.endLinks}><Link className="button primary" href="/docs">{t({ en: "Open the setup guide", zh: "查看配置文档" })}</Link><Link className="button" href="/contact">{t({ en: "Get help", zh: "获取帮助" })}</Link></div>
      </article>
    </SiteShell>
  );
}
