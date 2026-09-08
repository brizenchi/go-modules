"use client";

import Link from "next/link";
import { SiteShell } from "@/components/site-shell";
import { StructuredData } from "@/components/structured-data";
import { documentation, documentationGroups, type DocumentationSection } from "@/content/docs";
import { articleStructuredData } from "@/lib/structured-data";
import { useI18n } from "@/lib/i18n";
import { ReadingLayout } from "./reading-layout";
import styles from "./reading-layout.module.css";
import guideStyles from "./docs.module.css";

export function GuideArticle({ guide }: { guide: DocumentationSection }) {
  const { t, locale } = useI18n();
  const group = documentationGroups.find((item) => item.id === guide.group)!;
  const position = documentation.findIndex((item) => item.id === guide.id);
  const previous = documentation[position - 1];
  const next = documentation[position + 1];
  const labels = {
    overview: t({ en: "What to know", zh: "阅读说明" }),
    steps: t({ en: "Step by step", zh: "操作步骤" }),
    details: t({ en: "In detail", zh: "详细说明" }),
    configuration: t({ en: "Configuration example", zh: "配置示例" }),
    resources: t({ en: "Resources & next steps", zh: "相关资源与下一步" })
  };
  const toc = [
    { id: "overview", label: labels.overview },
    ...(guide.items?.length ? [{ id: "details", label: guide.ordered ? labels.steps : labels.details }] : []),
    ...(guide.code ? [{ id: "configuration", label: labels.configuration }] : []),
    ...(guide.links?.length ? [{ id: "resources", label: labels.resources }] : [])
  ];
  const List = guide.ordered ? "ol" : "ul";
  return (
    <SiteShell variant="article" eyebrow={t(group.title)} title={t(guide.title)} description={t(guide.summary)} showEnvironment={false}
      sideBody={<div><span>{t({ en: "Setup guide", zh: "配置指南" })}</span><Link href="/docs">{t({ en: "← All guides", zh: "← 全部指南" })}</Link></div>}
      breadcrumbs={[{ href: "/", label: t({ en: "Home", zh: "首页" }) }, { href: "/docs", label: t({ en: "Guide", zh: "使用指南" }) }, { label: t(guide.title) }]}
    >
      <StructuredData data={articleStructuredData({ title: t(guide.title), description: t(guide.summary), path: `/docs/${guide.id}`, language: locale === "zh" ? "zh-CN" : "en" })} />
      <ReadingLayout toc={toc} title={t({ en: "On this page", zh: "本页目录" })} supportLabel={t({ en: "Need help with this guide?", zh: "阅读中遇到问题？" })}>
        <section id="overview"><h2>{labels.overview}</h2>{guide.paragraphs.map((paragraph, index) => <p key={index}>{t(paragraph)}</p>)}</section>
        {guide.items?.length ? <section id="details"><h2>{guide.ordered ? labels.steps : labels.details}</h2>
          <List className={`${guideStyles.items}${guide.ordered ? ` ${guideStyles.steps}` : ""}`}>{guide.items.map((item) => <li key={item.title.en}><strong>{t(item.title)}</strong><p>{t(item.body)}</p></li>)}</List>
        </section> : null}
        {guide.code ? <section id="configuration"><h2>{labels.configuration}</h2><pre className={guideStyles.code}><code>{guide.code}</code></pre></section> : null}
        {guide.links?.length ? <section id="resources" className={styles.related}><h2>{labels.resources}</h2><ul>{guide.links.map((link) => <li key={link.href}><Link href={link.href}>{t(link.label)} <span aria-hidden="true">↗</span></Link></li>)}</ul></section> : null}
        <nav className={styles.pagination} aria-label={t({ en: "More guides", zh: "继续阅读指南" })}>
          {previous ? <Link href={`/docs/${previous.id}`}><span>{t({ en: "← Previous guide", zh: "← 上一篇" })}</span>{t(previous.title)}</Link> : <Link href="/docs"><span>{t({ en: "← Back to the directory", zh: "← 返回目录" })}</span>{t({ en: "Browse all guides", zh: "浏览全部指南" })}</Link>}
          {next ? <Link href={`/docs/${next.id}`}><span>{t({ en: "Next guide →", zh: "下一篇 →" })}</span>{t(next.title)}</Link> : <Link href="/blog"><span>{t({ en: "Keep reading →", zh: "继续阅读 →" })}</span>{t({ en: "The product journal", zh: "产品文章" })}</Link>}
        </nav>
      </ReadingLayout>
    </SiteShell>
  );
}
