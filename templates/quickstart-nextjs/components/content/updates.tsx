"use client";

import Link from "next/link";
import { SiteShell } from "@/components/site-shell";
import { updates } from "@/content/updates";
import { formatContentDate } from "@/content/articles";
import { useI18n } from "@/lib/i18n";
import styles from "./content.module.css";

export function ProductUpdates() {
  const { t, locale } = useI18n();
  return (
    <SiteShell
      eyebrow={t({ en: "Changelog", zh: "更新日志" })}
      title={t({ en: "What’s new in the starter.", zh: "模板的新进展。" })}
      description={t({ en: "Changes you can use today, with links to the pages and guides that explain them.", zh: "记录已经可以使用的功能，并附上相关页面与使用说明。" })}
      sideTitle={t({ en: "Follow the changes", zh: "了解变化" })} showEnvironment={false}
      sideBody={<p>{t({ en: "These notes describe the starter. Replace them with release notes for your own product as you ship.", zh: "这里记录模板本身的更新。发布自己的产品时，可以替换为面向用户的版本记录。" })}</p>}
      breadcrumbs={[{ href: "/", label: t({ en: "Home", zh: "首页" }) }, { label: t({ en: "Updates", zh: "更新日志" }) }]}
      toc={updates.map((update) => ({ id: update.id, label: t(update.label) }))}
    >
      <div className={styles.timeline}>{updates.map((update) => <article className={styles.release} id={update.id} key={update.id}><div className={styles.meta}><span className={styles.category}>{t(update.label)}</span><time dateTime={update.date}>{formatContentDate(update.date, locale)}</time></div><div><h2>{t(update.title)}</h2><ul>{update.items.map((item, index) => <li key={index}>{t(item)}</li>)}</ul><Link className="button" href={update.href}>{t(update.link)} ↗</Link></div></article>)}</div>
    </SiteShell>
  );
}
