"use client";

import Link from "next/link";
import {
  useEffect,
  useMemo,
  useState
} from "react";
import { usePathname } from "next/navigation";
import { appEnv } from "@/lib/env";
import { logout, userLabel, type ReferralStats, type SubscriptionView } from "@/lib/api";
import { loadAccountSummary, type AccountSummary } from "@/lib/account-summary";
import { readSession, SESSION_EVENT, writeSession, type AuthSession } from "@/lib/auth";
import { formatDate } from "@/lib/format";
import { useI18n } from "@/lib/i18n";
import { humanizeSegment } from "@/lib/locale";

type NavItem = {
  href: string;
  label: {
    en: string;
    zh: string;
  };
};

const topNav: NavItem[] = [
  { href: "/", label: { en: "Overview", zh: "总览" } },
  { href: "/pricing", label: { en: "Pricing", zh: "价格" } },
  { href: "/docs", label: { en: "Docs", zh: "文档" } }
];

type TOCItem = {
  id: string;
  label: string;
};

type SiteShellProps = {
  eyebrow: string;
  title: string;
  description: string;
  sideTitle?: string;
  sideBody?: React.ReactNode;
  children: React.ReactNode;
  actions?: React.ReactNode;
  breadcrumbs?: Array<{ href?: string; label: string }>;
  toc?: TOCItem[];
  accountMenuData?: Partial<AccountSummary>;
};

type AccountMenuData = {
  subscription?: SubscriptionView | null;
  referralStats?: ReferralStats | null;
};

function Breadcrumbs({
  items
}: {
  items?: Array<{ href?: string; label: string }>;
}) {
  if (!items || items.length === 0) {
    return null;
  }

  return (
    <nav className="breadcrumbs" aria-label="Breadcrumb">
      {items.map((item, index) => {
        const isLast = index === items.length - 1;
        return (
          <span className="breadcrumb-item" key={`${item.label}-${index}`}>
            {item.href && !isLast ? <Link href={item.href}>{item.label}</Link> : <span>{item.label}</span>}
            {!isLast ? <span className="breadcrumb-sep">/</span> : null}
          </span>
        );
      })}
    </nav>
  );
}

function LocaleSwitch() {
  const { locale, setLocale } = useI18n();

  return (
    <div className="locale-switch" aria-label="Language selector">
      {(["en", "zh"] as const).map((value) => (
        <button
          key={value}
          className={`locale-chip${locale === value ? " active" : ""}`}
          type="button"
          onClick={() => setLocale(value)}
        >
          {value === "en" ? "EN" : "中文"}
        </button>
      ))}
    </div>
  );
}

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

function AccountMenu({
  session,
  details,
  detailsLoading
}: {
  session: AuthSession | null;
  details?: AccountMenuData;
  detailsLoading?: boolean;
}) {
  const { t } = useI18n();
  const [open, setOpen] = useState(false);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    setOpen(false);
  }, [session?.token]);

  async function handleLogout() {
    if (busy) {
      return;
    }

    setBusy(true);
    try {
      if (session?.token) {
        await logout(session.token);
      }
    } catch {
      // Local sign-out should still succeed even if backend logout fails.
    } finally {
      writeSession(null);
      setBusy(false);
      setOpen(false);
    }
  }

  if (!session) {
    return (
      <div className="nav-actions">
        <Link className="button" href="/pricing">
          {t({ en: "View Pricing", zh: "查看价格" })}
        </Link>
        <Link className="button primary" href="/login">
          {t({ en: "Sign In", zh: "登录" })}
        </Link>
      </div>
    );
  }

  const initials = (session.user.username || session.user.email || session.user.id)
    .slice(0, 2)
    .toUpperCase();
  const subscriptionText = details?.subscription
    ? `${details.subscription.plan} · ${details.subscription.status}`
    : detailsLoading
      ? t({ en: "Loading...", zh: "加载中..." })
      : t({ en: "Unavailable", zh: "不可用" });
  const referralText = details?.referralStats
    ? `${details.referralStats.activated}/${details.referralStats.total_referred}`
    : detailsLoading
      ? t({ en: "Loading...", zh: "加载中..." })
      : t({ en: "Unavailable", zh: "不可用" });

  return (
    <div className="account-menu">
      <button
        className="avatar-button"
        type="button"
        onClick={() => setOpen((value) => !value)}
      >
        <span className="avatar-badge">{initials}</span>
        <span className="avatar-copy">
          <strong>{userLabel(session.user)}</strong>
          <span>{session.user.email}</span>
        </span>
      </button>

      {open ? (
        <div className="account-popover">
          <div className="account-popover-head">
            <span className="panel-kicker">{t({ en: "Workspace", zh: "工作区" })}</span>
            <strong>{appEnv.appName}</strong>
          </div>

          <div className="account-popover-grid">
            <Link className="popover-link-card" href="/account" onClick={() => setOpen(false)}>
              <span>{t({ en: "Settings", zh: "设置" })}</span>
              <small>{t({ en: "Identity, tokens, websocket ticket", zh: "身份、令牌、WebSocket 凭证" })}</small>
            </Link>
            <Link className="popover-link-card" href="/billing" onClick={() => setOpen(false)}>
              <span>{t({ en: "Subscription", zh: "订阅管理" })}</span>
              <small>{subscriptionText}</small>
            </Link>
            <Link className="popover-link-card" href="/referrals" onClick={() => setOpen(false)}>
              <span>{t({ en: "Referral Center", zh: "推荐中心" })}</span>
              <small>{t({ en: "Activated / total", zh: "已激活 / 总数" })}: {referralText}</small>
            </Link>
          </div>

          <div className="account-popover-meta">
            <div>
              <span className="panel-kicker">{t({ en: "Session expires", zh: "会话到期" })}</span>
              <strong>{formatDate(session.expires_at)}</strong>
            </div>
          </div>

          <button
            className="button danger wide"
            type="button"
            disabled={busy}
            onClick={() => void handleLogout()}
          >
            {busy ? t({ en: "Signing out...", zh: "退出中..." }) : t({ en: "Sign Out", zh: "退出登录" })}
          </button>
        </div>
      ) : null}
    </div>
  );
}

export function SiteShell(props: SiteShellProps) {
  const pathname = usePathname();
  const [session, setSession] = useState<AuthSession | null>(null);
  const [accountDetails, setAccountDetails] = useState<AccountSummary | null>(null);
  const [accountDetailsLoading, setAccountDetailsLoading] = useState(false);
  const { t } = useI18n();

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

  useEffect(() => {
    let cancelled = false;

    if (!session?.token) {
      setAccountDetails(null);
      setAccountDetailsLoading(false);
      return () => {
        cancelled = true;
      };
    }

    setAccountDetailsLoading(true);

    void loadAccountSummary(session.token)
      .then((details) => {
        if (!cancelled) {
          setAccountDetails(details);
        }
      })
      .catch(() => {
        if (!cancelled) {
          setAccountDetails({
            subscription: null,
            referralStats: null
          });
        }
      })
      .finally(() => {
        if (!cancelled) {
          setAccountDetailsLoading(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [session?.token]);

  const breadcrumbs = useMemo(
    () => props.breadcrumbs && props.breadcrumbs.length > 0 ? props.breadcrumbs : buildBreadcrumbs(pathname),
    [pathname, props.breadcrumbs]
  );

  const navItems = useMemo(() => topNav, []);
  const mergedAccountDetails: AccountMenuData = {
    subscription: props.accountMenuData?.subscription ?? accountDetails?.subscription ?? null,
    referralStats: props.accountMenuData?.referralStats ?? accountDetails?.referralStats ?? null
  };

  return (
    <div className="app-shell">
      <header className="topbar">
        <div className="topbar-inner">
          <Link className="brand-lockup" href="/">
            <span className="brand-kicker">go-modules</span>
            <span className="brand-title">{appEnv.appName}</span>
            <span className="brand-subtitle">{t({ en: "SaaS frontend template", zh: "SaaS 前端模板" })}</span>
          </Link>

          <nav className="main-nav" aria-label="Primary">
            {navItems.map((item) => (
              <Link
                key={item.href}
                href={item.href}
                className={`main-nav-link${pathname === item.href ? " active" : ""}`}
              >
                {t(item.label)}
              </Link>
            ))}
          </nav>

          <div className="topbar-tools">
            <LocaleSwitch />
            <AccountMenu
              session={session}
              details={mergedAccountDetails}
              detailsLoading={accountDetailsLoading}
            />
          </div>
        </div>
      </header>

      <main className="page-shell">
        <section className="hero-grid">
          <div className="hero-main-card">
            <Breadcrumbs items={breadcrumbs} />
            <span className="eyebrow">{props.eyebrow}</span>
            <h1>{props.title}</h1>
            <p>{props.description}</p>
            {props.actions ? <div className="hero-actions">{props.actions}</div> : null}
          </div>

          <div className="hero-side-stack">
            <div className="hero-side-card">
              <div className="panel-title-row compact">
                <div>
                  <span className="panel-kicker">{t({ en: "Environment", zh: "环境" })}</span>
                  <h3>{props.sideTitle || t({ en: "Context", zh: "上下文" })}</h3>
                </div>
                <span className="badge">{appEnv.appUrl}</span>
              </div>
              {props.sideBody}
            </div>

            <TableOfContents
              items={props.toc}
              title={t({ en: "On this page", zh: "本页目录" })}
            />
          </div>
        </section>

        {props.children}
      </main>
    </div>
  );
}
