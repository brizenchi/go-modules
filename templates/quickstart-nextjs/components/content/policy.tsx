"use client";

import Link from "next/link";
import { SiteShell } from "@/components/site-shell";
import { policies } from "@/content/policies";
import { useI18n } from "@/lib/i18n";
import styles from "./content.module.css";

export function PolicyContent({ kind }: { kind: "privacy" | "terms" }) {
  const { t } = useI18n();
  const policy = policies[kind];
  const name = kind === "privacy" ? t({ en: "Privacy", zh: "隐私说明" }) : t({ en: "Terms", zh: "使用条款" });
  return (
    <SiteShell eyebrow={name} title={t(policy.title)} description={t(policy.summary)} sideTitle={t({ en: "Starter content", zh: "待完善的模板内容" })} showEnvironment={false}
      sideBody={<p>{t({ en: "Adapt this page to the actual service before using it as a published policy.", zh: "请按实际业务完善本页，再将其作为正式政策使用。" })}</p>}
      breadcrumbs={[{ href: "/", label: t({ en: "Home", zh: "首页" }) }, { label: name }]}
      toc={policy.sections.map((section) => ({ id: section.id, label: t(section.title) }))}>
      <article className={styles.body}>
        <div className={styles.note}><p>{t({ en: "This is an editable starting point. Operator identity, dates, retention periods, and product-specific rules still need to be supplied.", zh: "本页是可编辑的起点，运营主体、日期、保存期限和产品具体规则仍需补充。" })}</p></div>
        {policy.sections.map((section) => <section className={styles.section} id={section.id} key={section.id}><h2>{t(section.title)}</h2>{section.paragraphs.map((paragraph, index) => <p key={index}>{t(paragraph)}</p>)}</section>)}
        <div className={styles.endLinks}><Link className="button" href="/contact">{t({ en: "Contact support", zh: "联系支持" })}</Link><Link className="button" href={kind === "privacy" ? "/terms" : "/privacy"}>{kind === "privacy" ? t({ en: "Read terms", zh: "阅读使用条款" }) : t({ en: "Read privacy information", zh: "阅读隐私说明" })}</Link></div>
      </article>
    </SiteShell>
  );
}
