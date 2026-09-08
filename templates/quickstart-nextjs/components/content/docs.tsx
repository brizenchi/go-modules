"use client";

import Link from "next/link";
import { useState } from "react";
import { SiteShell } from "@/components/site-shell";
import { documentation, documentationGroups, searchDocumentation } from "@/content/docs";
import { useI18n } from "@/lib/i18n";
import styles from "./content.module.css";
import guideStyles from "./docs.module.css";

export function Documentation() {
  const { t, locale } = useI18n();
  const [query, setQuery] = useState("");
  const matches = searchDocumentation(query, locale);

  return (
    <SiteShell
      eyebrow={t({ en: "The field guide", zh: "模板使用指南" })}
      title={t({ en: "Your next step starts here.", zh: "下一步，从这里开始。" })}
      description={t({ en: "One guide, one task. Explore the template, connect your services, and get your own SaaS ready to launch.", zh: "一篇指南，解决一个问题。从了解模板、连接服务，到上线自己的 SaaS，按需要开始阅读。" })}
      sideTitle={t({ en: "New to the template?", zh: "第一次使用？" })}
      showEnvironment={false}
      sideBody={<div className={guideStyles.startLinks}>
        <Link href="/docs/try-demo">{t({ en: "Try the customer journey →", zh: "先体验完整用户流程 →" })}</Link>
        <Link href="/docs/domains">{t({ en: "Set up your own project →", zh: "开始配置自己的项目 →" })}</Link>
        <p>{t({ en: "Free source. Your own services. A foundation for the product you want to build.", zh: "免费源码，接入自己的服务，为你想做的产品打好基础。" })}</p>
      </div>}
      breadcrumbs={[{ href: "/", label: t({ en: "Home", zh: "首页" }) }, { label: t({ en: "Guide", zh: "使用指南" }) }]}
    >
      <div className={styles.toolbar}>
        <label className={styles.search}>
          {t({ en: "Find your guide", zh: "查找指南" })}
          <input type="search" value={query} onChange={(event) => setQuery(event.target.value)} placeholder={t({ en: "Search domains, Stripe, email…", zh: "搜索域名、Stripe、邮件…" })} />
        </label>
        <span className={styles.resultCount} role="status">{t({ en: `${matches.length} of ${documentation.length} guides`, zh: `共 ${documentation.length} 篇，找到 ${matches.length} 篇` })}</span>
      </div>
      <nav className={guideStyles.categories} aria-label={t({ en: "Guide categories", zh: "指南分类" })}>
        {documentationGroups.filter((group) => matches.some((guide) => guide.group === group.id)).map((group) => <a href={`#group-${group.id}`} key={group.id}>{t(group.title)} <span aria-hidden="true">↓</span></a>)}
      </nav>
      {documentationGroups.map((group) => {
        const guides = matches.filter((guide) => guide.group === group.id);
        if (guides.length === 0) return null;
        return <section className={guideStyles.group} id={`group-${group.id}`} aria-labelledby={`heading-${group.id}`} key={group.id}>
          <div className={guideStyles.groupHeading}><h2 id={`heading-${group.id}`}>{t(group.title)}</h2><p>{t(group.description)}</p></div>
          <div className={guideStyles.cards}>{guides.map((guide) => <Link className={guideStyles.card} href={`/docs/${guide.id}`} id={guide.id} key={guide.id}>
            <h3>{t(guide.title)}</h3><p>{t(guide.summary)}</p><span className={guideStyles.read}>{t({ en: "Read guide", zh: "阅读指南" })} <span aria-hidden="true">↗</span></span>
          </Link>)}</div>
        </section>;
      })}
      {matches.length === 0 ? <div className={styles.empty}>
        <h2>{t({ en: "No matching guides", zh: "没有找到相关指南" })}</h2>
        <p>{t({ en: "Try another topic or browse all guides.", zh: "换一个关键词，或浏览全部指南。" })}</p>
        <button className="button" type="button" onClick={() => setQuery("")}>{t({ en: "Clear search", zh: "清除搜索" })}</button>
      </div> : null}
      <div className={styles.endLinks}>
        <Link className="button" href="/blog">{t({ en: "Explore the product journal", zh: "阅读产品文章" })}</Link>
        <Link className="button" href="/contact">{t({ en: "Need a hand?", zh: "需要帮助？" })}</Link>
      </div>
    </SiteShell>
  );
}
