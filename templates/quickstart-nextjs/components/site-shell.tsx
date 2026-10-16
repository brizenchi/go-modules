"use client";

import { LocalizedLink as Link } from "@/components/localized-link";
import {
  useEffect,
  useMemo,
  useState
} from "react";
import { usePathname, useRouter } from "next/navigation";
import { appEnv } from "@/lib/env";
import { siteLinkForPath } from "@/content/navigation";
import { PublicNavigation } from "./public-navigation";
import { useSitePresentation } from "./site-presentation";
import { type AccountSummary } from "@/lib/account-summary";
import { readSession, SESSION_EVENT, type AuthSession } from "@/lib/auth";
import { useI18n } from "@/lib/i18n";
import { humanizeSegment, stripLocalePrefix, localizeHref } from "@/lib/locale";
import { CustomerPage } from "@/components/customer-shell";
import { AccountMenu } from "@/components/account-menu";
import { LocaleSwitch } from "@/components/locale-switch";
import { Breadcrumbs } from "@/components/breadcrumbs";
import { BrandMark } from "./brand-mark";
import { SiteFooter } from "@/components/site-footer";
import { publicPaths } from "@/lib/seo";
import {
  customerSignInDestination,
  isWorkspacePath,
} from "@/lib/workspace";

type TOCItem = {
  id: string;
  label: string;
};

type SiteShellProps = {
  pagePath?: string;
  variant?: "article" | "compact";
  className?: string;
  eyebrow: string;
  title: string;
  description: string;
  sideTitle?: string;
  sideBody?: React.ReactNode;
  heroVisual?: React.ReactNode;
  heroFootnote?: React.ReactNode;
  showEnvironment?: boolean;
  children: React.ReactNode;
  actions?: React.ReactNode;
  breadcrumbs?: Array<{ href?: string; label: string }>;
  toc?: TOCItem[];
  accountMenuData?: Partial<AccountSummary>;
};

function TableOfContents({
  items,
  title
}: {
  items?: TOCItem[];
  title: string;
}) {
  if (!items || items.length === 0) {
    return null;
  }

  return (
    <aside className="toc-card">
      <div className="toc-title">{title}</div>
      <div className="toc-list">
        {items.map((item) => (
          <a className="toc-link" href={`#${item.id}`} key={item.id}>
            {item.label}
          </a>
        ))}
      </div>
    </aside>
  );
}

function buildBreadcrumbs(pathname: string): Array<{ href?: string; label: string }> {
  if (pathname === "/") {
    return [{ label: "Home" }];
  }

  const parts = pathname.split("/").filter(Boolean);
  const breadcrumbs: Array<{ href?: string; label: string }> = [{ href: "/", label: "Home" }];

  let current = "";
  for (const part of parts) {
    current += `/${part}`;
    breadcrumbs.push({
      href: current,
      label: humanizeSegment(part)
    });
  }

  return breadcrumbs;
}

export function SiteShell(props: SiteShellProps) {
  const pathname = stripLocalePrefix(usePathname());
  if (isWorkspacePath(pathname)) {
    return <CustomerPage title={props.title} description={props.description} actions={props.actions}
      context={props.sideBody} contextTitle={props.sideTitle}>{props.children}</CustomerPage>;
  }
  return <PublicSiteShell {...props} />;
}

function PublicSiteShell(props: SiteShellProps) {
  const rawPathname = usePathname();
  const pathname = props.pagePath ?? stripLocalePrefix(rawPathname);
  const router = useRouter();
  const isHome = pathname === "/";
 const presentation=useSitePresentation();
 const brand=presentation?.brand_name || appEnv.appName;
  const [session, setSession] = useState<AuthSession | null>(null);
  const { t, locale } = useI18n();

  useEffect(() => {
    const sync = () => setSession(readSession());
    sync();
    window.addEventListener("storage", sync);
    window.addEventListener(SESSION_EVENT, sync);
    return () => {
      window.removeEventListener("storage", sync);
      window.removeEventListener(SESSION_EVENT, sync);
    };
  }, []);

  const breadcrumbs = useMemo(
    () => (props.breadcrumbs && props.breadcrumbs.length > 0 ? props.breadcrumbs : buildBreadcrumbs(pathname)).map((item, index, items) => {
      const known = siteLinkForPath(item.href || (index === items.length - 1 ? pathname : ""));
      return { ...item, label: known ? t(known.label) : item.label, href: item.href ? localizeHref(item.href, locale) : undefined };
    }),
    [pathname, props.breadcrumbs, locale, t]
  );

  const accountMenu = (
    <AccountMenu
      session={session}
      onSignInSuccess={() => {
        const currentPath = `${window.location.pathname}${window.location.search}${window.location.hash}`;
        const destination = customerSignInDestination(currentPath);
        if (destination !== currentPath) router.push(destination);
      }}
    />
  );

  const Content = props.variant === "article" ? "article" : "div";
  const hasPublicSchema = publicPaths.includes(pathname) || pathname.startsWith("/docs/") || pathname.startsWith("/blog/") || /^\/(features|use-cases|compare)\//.test(pathname);

  return (
    <div className={`app-shell ${props.className || ""}${props.variant === "article" ? " article-shell" : props.variant === "compact" ? " compact-shell" : ""}`}>
      <a className="skip-content" href="#main-content">{t({ en: "Skip to content", zh: "跳到正文" })}</a>
      <header className="topbar">
        <div className="topbar-inner">
          <Link className="brand-lockup" href="/" aria-label={`${brand} · ${t({ en: "Home", zh: "首页" })}`}>
            <BrandMark className="brand-mark" />
            <span className="brand-copy">
              <span className="brand-title">{brand}</span>
            </span>
          </Link>

          <PublicNavigation pathname={pathname} />

          <div className="topbar-tools">
            <LocaleSwitch />
            {accountMenu}
          </div>
        </div>
      </header>

      <main id="main-content" tabIndex={-1} className={`page-shell${isHome ? " home-page-shell" : ""}`}>
        {!isHome ? <Breadcrumbs items={breadcrumbs} path={rawPathname} label={t({ en: "Breadcrumb", zh: "面包屑导航" })} structured={hasPublicSchema} /> : null}
        <Content aria-label={props.variant === "article" ? props.title : undefined}>
          <section className={`hero-grid${isHome ? " home-hero" : ""}${props.heroVisual ? " hero-with-visual" : ""}`}>
            <div className="hero-main-card">
              <span className="eyebrow">{props.eyebrow}</span>
              <h1>{props.title}</h1>
              {props.description.trim() ? <p>{props.description}</p> : null}
              {props.variant === "article" ? <div className="article-byline">{props.sideBody}</div> : null}
              {props.actions ? <div className="hero-actions">{props.actions}</div> : null}
              {props.heroFootnote ? <div className="hero-footnote">{props.heroFootnote}</div> : null}
            </div>

            {props.heroVisual ? <div className="hero-visual">{props.heroVisual}</div> : !props.variant ? <div className="hero-side-stack">
              <div className="hero-side-card">
                <div className="panel-title-row compact">
                  <div>
                    {props.showEnvironment !== false ? <span className="panel-kicker">{t({ en: "Environment", zh: "环境" })}</span> : null}
                    <h2>{props.sideTitle || t({ en: "Context", zh: "上下文" })}</h2>
                  </div>
                  {props.showEnvironment !== false ? <span className="badge">{appEnv.appUrl}</span> : null}
                </div>
                {props.sideBody}
              </div>

              <TableOfContents
                items={isHome ? undefined : props.toc}
                title={t({ en: "On this page", zh: "本页目录" })}
              />
            </div> : null}
          </section>

          {props.children}
        </Content>
      </main>

      <SiteFooter name={brand} description={presentation ? (locale === "zh" ? presentation.description_zh : presentation.description) || undefined : undefined} />
    </div>
  );
}
