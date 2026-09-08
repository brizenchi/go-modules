import Link from "next/link";
import { useI18n } from "@/lib/i18n";
import styles from "./site-footer.module.css";

const groups = [
  { title: { en: "Product", zh: "产品" }, links: [
    { href: "/", label: { en: "Overview", zh: "产品总览" } },
    { href: "/pricing", label: { en: "Pricing", zh: "套餐与价格" } },
    { href: "/updates", label: { en: "What's new", zh: "更新日志" } }
  ] },
  { title: { en: "Learn", zh: "学习与指南" }, links: [
    { href: "/docs", label: { en: "All guides", zh: "全部指南" } },
    { href: "/docs/overview", label: { en: "Getting started", zh: "快速入门" } },
    { href: "/docs/domains", label: { en: "Setup & deployment", zh: "配置与部署" } },
    { href: "/blog", label: { en: "Product journal", zh: "产品文章" } }
  ] },
  { title: { en: "Support & legal", zh: "支持与条款" }, links: [
    { href: "/contact", label: { en: "Contact", zh: "联系支持" } },
    { href: "/privacy", label: { en: "Privacy policy", zh: "隐私政策" } },
    { href: "/terms", label: { en: "Terms of service", zh: "服务条款" } }
  ] }
];

export function SiteFooter({ name, description }: { name: string; description: string }) {
  const { t } = useI18n();
  return (
    <footer className={styles.footer}>
      <div className={styles.inner}>
        <div className={styles.grid}>
          <div className={styles.identity}>
            <Link className={styles.brand} href="/"><span aria-hidden="true">◈</span>{name}</Link>
            <p>{description || t({ en: "Less groundwork. More room for your next idea. A free Next.js + Go SaaS starter.", zh: "少一些重复搭建，多一些专注业务。免费的 Next.js + Go SaaS 启动模板。" })}</p>
            <span className={styles.stack}>Next.js + Go</span>
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
          <span>{t({ en: "Built with go-modules", zh: "基于 go-modules 构建" })}</span>
          <a href="#main-content">{t({ en: "Back to top ↑", zh: "返回顶部 ↑" })}</a>
        </div>
      </div>
    </footer>
  );
}
