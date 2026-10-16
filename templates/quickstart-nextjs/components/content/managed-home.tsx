"use client";

import Image from "next/image";
import type { PagePresentation } from "@/lib/page-presentation";
import { useI18n } from "@/lib/i18n";
import { latestGuides, uniqueContent, type DiscoveryEntry } from "@/lib/content-discovery";
import { siteConfig } from "@/content/site";
import { SiteShell } from "../site-shell";
import { CTAButton, PageSection } from "../ui";
import { ContentCards } from "../content-discovery";
import { LocalizedLink as Link } from "../localized-link";
import { MarkdownContent } from "../markdown-content";
import { ArticleLanguagesProvider } from "../article-language-context";
import styles from "./home.module.css";

// All editorial text comes from published CMS snapshots. Navigation labels and
// visual assets are presentation; they do not duplicate editable product claims.
export function ManagedHome({ page, entries, identityOnly = false }: { page: PagePresentation; entries: DiscoveryEntry[]; identityOnly?: boolean }) {
 const { t, locale } = useI18n();
 const catalog = uniqueContent(entries);
 const features = catalog.filter(entry => entry.kind === "features");
 const cases = catalog.filter(entry => entry.kind === "use-cases").slice(0, 3);
 const guides = latestGuides(catalog);
 return <ArticleLanguagesProvider locales={page.locales}><SiteShell pagePath="/" eyebrow={t(siteConfig.hero.eyebrow)}
  title={page.translation.title} description={page.translation.summary} showEnvironment={false}
  heroVisual={<figure className={styles.heroPreview}><div className={styles.previewChrome}><span aria-hidden="true">● ● ●</span><span>{t({ en: "Account center", zh: "账户中心" })}</span><span>{t({ en: "PREVIEW", zh: "界面预览" })}</span></div><Image src={siteConfig.preview[locale].src} alt={t({ en: "Preview of the supplied customer account interface", zh: "模板用户账户界面预览" })} width={1200} height={640} priority sizes="(max-width: 980px) 100vw, 680px" /></figure>}
  actions={identityOnly ? <><CTAButton href="/pricing" primary>{t({ en: "Explore plans", zh: "查看套餐" })}</CTAButton><CTAButton href="/account">{t({ en: "My account", zh: "账户中心" })}</CTAButton></> : <><CTAButton href="/examples" primary>{t({ en: "Explore the demo", zh: "体验演示" })}</CTAButton><CTAButton href="/docs/overview">{t({ en: "Read the overview", zh: "阅读概览" })}</CTAButton></>}
 >
  <section className={styles.spaces} aria-label={t({ en: "Explore the template", zh: "了解模板" })}>{[
   { href: "/features", label: t({ en: "Features", zh: "功能介绍" }) },
   { href: "/account", label: t({ en: "Account center", zh: "账户中心" }) },
   { href: entries.some(entry => entry.path === "/docs/admin") ? "/docs/admin" : "/docs", label: t({ en: "Documentation", zh: "使用文档" }) }
  ].map((entry, i) => <Link href={entry.href} key={entry.href}><span className={styles.entryNumber}>0{i + 1}</span><strong>{entry.label}</strong><span aria-hidden="true">↗</span></Link>)}</section>
  {features.length ? <PageSection id="included" title={t({ en: "Included capabilities", zh: "已整合的功能" })}><ContentCards entries={features} /><div className={styles.sectionLinks}><Link href="/features">{t({ en: "Explore features", zh: "查看功能介绍" })} →</Link></div></PageSection> : null}
  {page.translation.body ? <section className={styles.editorial} aria-label={t({ en: "About the starter", zh: "了解模板" })}><MarkdownContent source={page.translation.body} locale={locale} idPrefix="home-" /></section> : null}
  {cases.length ? <PageSection id="use-cases" title={t({ en: "Use cases", zh: "使用场景" })}><ContentCards entries={cases} /><div className={styles.sectionLinks}><Link href="/use-cases">{t({ en: "All use cases", zh: "全部使用场景" })} →</Link><Link href="/compare">{t({ en: "Compare starting points", zh: "比较开发方案" })} →</Link></div></PageSection> : null}
  <PageSection id="latest-guides" title={t({ en: "Latest guides", zh: "最近发布的指南" })}>{guides.length ? <ContentCards entries={guides} /> : null}<div className={styles.sectionLinks}><Link href="/resources">{t({ en: "All guides", zh: "全部指南" })} →</Link><Link href="/updates">{t({ en: "Product updates", zh: "更新日志" })} →</Link><Link href="/about">{t({ en: "About the project", zh: "关于项目" })} →</Link></div></PageSection>
 </SiteShell></ArticleLanguagesProvider>;
}
