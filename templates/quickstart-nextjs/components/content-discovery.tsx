"use client";

import { LocalizedLink as Link } from "./localized-link";
import { guideDirectories, guideNavigation, productNavigation, siteLinks } from "@/content/navigation";
import { relatedContent, type DiscoveryEntry } from "@/lib/content-discovery";
import { useI18n } from "@/lib/i18n";
import { useDirectoryFilter } from "./navigation-index";
import styles from "./content-discovery.module.css";

export function ProductNavigation({ currentPath }: { currentPath: string }) {
  const { t } = useI18n();
  const shown = useDirectoryFilter();
  return <nav className={styles.productNav} aria-label={t({ en: "Explore the product", zh: "了解产品" })} data-product-navigation>
    {productNavigation.filter(item => shown(item.href)).map(item => <Link href={item.href} key={item.href}
      aria-current={currentPath === item.href ? "page" : currentPath.startsWith(`${item.href}/`) ? "true" : undefined}>{t(item.label)}</Link>)}
  </nav>;
}

export function GuideDirectories() {
  const { t } = useI18n();
  const shown = useDirectoryFilter();
  return <nav className={styles.directories} aria-label={t({ en: "Guide directories", zh: "指南目录" })} data-guide-directories>
    {guideDirectories.filter(item => shown(item.href)).map((item, index) => <Link className={styles.directory} href={item.href} key={item.href}>
      <span className={styles.number} aria-hidden="true">0{index + 1}</span>
      <h2>{t(item.label)}</h2><p>{t(item.description)}</p>
      <span className={styles.destination}>{t({ en: "Browse", zh: "浏览目录" })}<span aria-hidden="true">→</span></span>
    </Link>)}
  </nav>;
}

export function GuideNavigation({ currentPath }: { currentPath: string }) {
  const { t } = useI18n();
  const shown = useDirectoryFilter();
  return <nav className={styles.productNav} aria-label={t({ en: "Guide sections", zh: "指南栏目" })} data-guide-navigation>
    {guideNavigation.filter(item => shown(item.href)).map(item => <Link href={item.href} key={item.href} aria-current={currentPath === item.href ? "page" : undefined}>{t(item.label)}</Link>)}
  </nav>;
}

export function ContentCards({ entries }: { entries: readonly DiscoveryEntry[] }) {
  const { t } = useI18n();
  const labels = { page: { en: "Page", zh: "页面" }, docs: siteLinks.docs.label, blog: siteLinks.blog.label, features: siteLinks.features.label, "use-cases": siteLinks.useCases.label, compare: siteLinks.compare.label };
  return <ul className={styles.cards} data-content-cards>
    {entries.map(entry => <li key={entry.path}><Link className={styles.card} href={entry.path}>
      <span className={styles.category}>{t(labels[entry.kind])}</span>
      <h3>{entry.translation.title}</h3><p>{entry.translation.summary}</p>
      <span className={styles.destination}>{t({ en: "Explore", zh: "查看详情" })}<span aria-hidden="true">→</span></span>
    </Link></li>)}
  </ul>;
}

export function RelatedContent({ entries, currentPath }: { entries: readonly DiscoveryEntry[]; currentPath: string }) {
  const { t } = useI18n();
  const related = relatedContent(entries, currentPath);
  if (!related.length) return null;
  return <section className={styles.related} data-related-content aria-label={t({ en: "Continue exploring", zh: "继续了解" })}>
    <h2>{t({ en: "Continue exploring", zh: "继续了解" })}</h2>
    <ContentCards entries={related} />
  </section>;
}
