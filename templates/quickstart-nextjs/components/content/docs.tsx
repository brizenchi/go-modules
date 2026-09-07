"use client";

import Link from "next/link";
import { useState } from "react";
import { SiteShell } from "@/components/site-shell";
import { DocArticle } from "@/components/marketing";
import { documentation } from "@/content/docs";
import { useI18n } from "@/lib/i18n";
import styles from "./content.module.css";
import guideStyles from "./docs.module.css";

export function Documentation() {
  const { t } = useI18n();
  const [query, setQuery] = useState("");
  const needle = query.trim().toLocaleLowerCase();
  const matches = documentation.filter((section) => [
    t(section.title),
    ...section.paragraphs.map((paragraph) => t(paragraph)),
    ...(section.items ?? []).flatMap((item) => [t(item.title), t(item.body)]),
    ...(section.links ?? []).map((link) => t(link.label)),
    section.code ?? ""
  ].join(" ").toLocaleLowerCase().includes(needle));

  return (
    <SiteShell
      eyebrow={t({ en: "Template guide", zh: "模板使用指南" })}
      title={t({ en: "From trying the template to launching your SaaS.", zh: "从体验模板，到上线自己的 SaaS。" })}
      description={t({ en: "Understand what is included, try the customer journey, then connect your services and add the feature your customers will pay for.", zh: "先了解模板已经做好的功能，体验一次完整用户流程，再接入自己的服务，加入客户愿意付费的核心业务。" })}
      sideTitle={t({ en: "Two ways to start", zh: "从这里开始" })}
      showEnvironment={false}
      sideBody={<div className={guideStyles.startLinks}>
        <Link href="/docs#try-demo" onClick={() => setQuery("")}>{t({ en: "I want to try the demo →", zh: "我想先体验演示 →" })}</Link>
        <Link href="/docs#domains" onClick={() => setQuery("")}>{t({ en: "I want to build my SaaS →", zh: "我想搭建自己的 SaaS →" })}</Link>
        <p>{t({ en: "The template is free. Add your own business feature and connect your service accounts to make it your product.", zh: "模板免费。加入自己的业务功能，接入自己的服务账号，把它变成你的产品。" })}</p>
      </div>}
      breadcrumbs={[{ href: "/", label: t({ en: "Home", zh: "首页" }) }, { label: t({ en: "Guide", zh: "使用指南" }) }]}
      toc={matches.map((section) => ({ id: section.id, label: t(section.title) }))}
    >
      <div className={styles.toolbar}>
        <label className={styles.search}>
          {t({ en: "Search the guide", zh: "搜索使用指南" })}
          <input type="search" value={query} onChange={(event) => setQuery(event.target.value)} placeholder={t({ en: "Try test cards, Resend, admin, or invitations…", zh: "搜索测试卡、Resend、管理员或邀请…" })} />
        </label>
        <span className={styles.resultCount} role="status">{t({ en: `${matches.length} sections`, zh: `${matches.length} 个章节` })}</span>
      </div>
      <div className={`doc-layout ${guideStyles.layout}`}>
        {matches.map((section) => {
          const List = section.ordered ? "ol" : "ul";
          return <DocArticle key={section.id} id={section.id} title={t(section.title)}>
            {section.paragraphs.map((paragraph, index) => <p key={index}>{t(paragraph)}</p>)}
            {section.items ? <List className={`${guideStyles.items}${section.ordered ? ` ${guideStyles.steps}` : ""}`}>
              {section.items.map((item) => <li key={item.title.en}>
                <strong>{t(item.title)}</strong>
                <p>{t(item.body)}</p>
              </li>)}
            </List> : null}
            {section.code ? <pre className={guideStyles.code}><code>{section.code}</code></pre> : null}
            {section.links ? <div className={guideStyles.sectionLinks}>{section.links.map((link) => <Link href={link.href} key={link.href} onClick={link.href.startsWith("/docs#") ? () => setQuery("") : undefined}>{t(link.label)}<span aria-hidden="true"> ↗</span></Link>)}</div> : null}
          </DocArticle>;
        })}
      </div>
      {matches.length === 0 ? <div className={styles.empty}>
        <p>{t({ en: "No matching sections. Try another search.", zh: "没有找到相关章节，请尝试其他关键词。" })}</p>
        <button className="button" type="button" onClick={() => setQuery("")}>{t({ en: "Clear search", zh: "清除搜索" })}</button>
      </div> : null}
      <div className={styles.endLinks}>
        <Link className="button primary" href="/account">{t({ en: "Try the account center", zh: "体验用户工作台" })}</Link>
        <Link className="button" href="/contact">{t({ en: "Get help", zh: "获取帮助" })}</Link>
      </div>
    </SiteShell>
  );
}
