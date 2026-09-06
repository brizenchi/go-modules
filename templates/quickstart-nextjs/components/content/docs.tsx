"use client";

import Link from "next/link";
import { useState } from "react";
import { SiteShell } from "@/components/site-shell";
import { DocArticle } from "@/components/marketing";
import { documentation } from "@/content/docs";
import { useI18n } from "@/lib/i18n";
import styles from "./content.module.css";

export function Documentation() {
  const { t } = useI18n();
  const [query, setQuery] = useState("");
  const needle = query.trim().toLocaleLowerCase();
  const matches = documentation.filter((section) => [t(section.title), ...section.paragraphs.map((paragraph) => t(paragraph))].join(" ").toLocaleLowerCase().includes(needle));
  return (
    <SiteShell eyebrow={t({ en: "Documentation", zh: "配置文档" })} title={t({ en: "Make it yours, step by step.", zh: "一步步，搭好自己的 SaaS。" })} description={t({ en: "Connect your services, test the customer journey, and replace the starter content with your product’s own voice.", zh: "接入服务，验证用户流程，再把模板内容替换成你自己的产品介绍。" })} sideTitle={t({ en: "Setup path", zh: "配置顺序" })} showEnvironment={false} sideBody={<p>{t({ en: "Domains → sign-in → payments → invitations → public content.", zh: "域名 → 登录 → 支付 → 邀请 → 公开内容。" })}</p>} breadcrumbs={[{ href: "/", label: t({ en: "Home", zh: "首页" }) }, { label: t({ en: "Docs", zh: "文档" }) }]} toc={matches.map((section) => ({ id: section.id, label: t(section.title) }))}>
      <div className={styles.toolbar}><label className={styles.search}>{t({ en: "Search documentation", zh: "搜索文档" })}<input type="search" value={query} onChange={(event) => setQuery(event.target.value)} placeholder={t({ en: "Try domains, Resend, or invitations…", zh: "搜索域名、Resend 或邀请…" })} /></label><span className={styles.resultCount} role="status">{t({ en: `${matches.length} sections`, zh: `${matches.length} 个章节` })}</span></div>
      <div className="doc-layout">{matches.map((section) => <DocArticle key={section.id} id={section.id} title={t(section.title)}>{section.paragraphs.map((paragraph, index) => <p key={index}>{t(paragraph)}</p>)}</DocArticle>)}</div>
      {matches.length === 0 ? <div className={styles.empty}><p>{t({ en: "No matching sections. Try another search.", zh: "没有找到相关章节，请尝试其他关键词。" })}</p><button className="button" type="button" onClick={() => setQuery("")}>{t({ en: "Clear search", zh: "清除搜索" })}</button></div> : null}
      <div className={styles.endLinks}><Link className="button" href="/blog">{t({ en: "Read product guides", zh: "阅读产品指南" })}</Link><Link className="button" href="/contact">{t({ en: "Get help", zh: "获取帮助" })}</Link></div>
    </SiteShell>
  );
}
