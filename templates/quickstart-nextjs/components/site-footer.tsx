import { LocalizedLink as Link } from "@/components/localized-link";
import { useI18n } from "@/lib/i18n";
import { siteConfig } from "@/content/site";
import styles from "./site-footer.module.css";

const groups = [
  { title: { en: "Product", zh: "产品" }, links: [
    { href: "/features", label: { en: "Features", zh: "产品功能" } },
    { href: "/use-cases", label: { en: "Use cases", zh: "使用场景" } },
    { href: "/pricing", label: { en: "Pricing", zh: "套餐与价格" } },
    { href: "/updates", label: { en: "What's new", zh: "更新日志" } }
  ] },
  { title: { en: "Learn", zh: "学习与指南" }, links: [
    { href: "/docs", label: { en: "All guides", zh: "全部指南" } },
    { href: "/resources", label: { en: "Resources", zh: "资源中心" } },
    { href: "/examples", label: { en: "Examples", zh: "使用示例" } },
    { href: "/compare", label: { en: "Comparisons", zh: "方案比较" } },
    { href: "/docs/domains", label: { en: "Setup & deployment", zh: "配置与部署" } },
    { href: "/blog", label: { en: "Product journal", zh: "产品文章" } }
  ] },
  { title: { en: "Support & legal", zh: "支持与条款" }, links: [
    { href: "/about", label: { en: "About", zh: "关于我们" } },
    { href: "/contact", label: { en: "Contact", zh: "联系支持" } },
    { href: "/privacy", label: { en: "Privacy policy", zh: "隐私政策" } },
    { href: "/terms", label: { en: "Terms of service", zh: "服务条款" } }
  ] }
];

export function SiteFooter({ name, description }: { name: string; description?: string }) {
  const { t } = useI18n();
  return (
    <footer className={styles.footer}>
      <div className={styles.inner}>
        <div className={styles.grid}>
          <div className={styles.identity}>
            <Link className={styles.brand} href="/"><span aria-hidden="true">◈</span>{name}</Link>
            <p>{description || t(siteConfig.footer.description)}</p>
            <span className={styles.stack}>{t(siteConfig.brand.subtitle)}</span>
          </div>
          {groups.map((group, index) => (
            <nav key={group.title.en} aria-labelledby={`footer-group-${index}`}>
              <h2 id={`footer-group-${index}`}>{t(group.title)}</h2>
              <ul>{group.links.map((link) => <li key={link.href}><Link href={link.href}>{t(link.label)}</Link></li>)}</ul>
            </nav>
          ))}
        </div>
        <div className={styles.bottom}>
          <span>© {new Date().getFullYear()} {name}</span>
          <span>{t(siteConfig.brand.attribution)}</span>
          <a href="#main-content">{t({ en: "Back to top ↑", zh: "返回顶部 ↑" })}</a>
        </div>
      </div>
    </footer>
  );
}
