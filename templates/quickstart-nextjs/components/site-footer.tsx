import { LocalizedLink as Link } from "@/components/localized-link";
import { useI18n } from "@/lib/i18n";
import { siteConfig } from "@/content/site";
import { siteFooterGroups, siteLegalLinks } from "@/content/navigation";
import { BrandMark } from "./brand-mark";
import styles from "./site-footer.module.css";


export function SiteFooter({ name, description }: { name: string; description?: string }) {
  const { t } = useI18n();
  return (
    <footer className={styles.footer} data-site-footer>
      <div className={styles.inner}>
        <div className={styles.grid}>
          <div className={styles.identity}>
            <Link className={styles.brand} href="/"><BrandMark className={styles.brandMark} />{name}</Link>
            <p>{description || t(siteConfig.footer.description)}</p>
          </div>
          {siteFooterGroups.map((group, index) => (
            <nav key={group.title.en} aria-labelledby={`footer-group-${index}`}>
              <h2 id={`footer-group-${index}`}>{t(group.title)}</h2>
              <ul>{group.links.map((link) => <li key={link.href}><Link href={link.href}>{t(link.label)}</Link></li>)}</ul>
            </nav>
          ))}
        </div>
        <div className={styles.bottom}>
          <span className={styles.copyright}>© {new Date().getFullYear()} {name}</span>
          <nav className={styles.legal} aria-label={t({ en: "Legal", zh: "法律条款" })}>{siteLegalLinks.map(link => <Link key={link.href} href={link.href}>{t(link.label)}</Link>)}</nav>
          <a className={styles.backToTop} href="#main-content">{t({ en: "Back to top ↑", zh: "返回顶部 ↑" })}</a>
        </div>
      </div>
    </footer>
  );
}
