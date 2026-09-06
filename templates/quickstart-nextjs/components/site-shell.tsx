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
import { clearSessionIfToken, readSession, SESSION_EVENT, writeSession, type AuthSession } from "@/lib/auth";
import { formatDate } from "@/lib/format";
import { useI18n } from "@/lib/i18n";
import { humanizeSegment } from "@/lib/locale";
import {
  describeRequestFailure,
  failedResource,
  loadingResource,
  type ResourceState
} from "@/lib/request-state";
import { SignInDialog } from "@/components/sign-in-dialog";

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
  showEnvironment?: boolean;
  children: React.ReactNode;
  actions?: React.ReactNode;
  breadcrumbs?: Array<{ href?: string; label: string }>;
  toc?: TOCItem[];
  accountMenuData?: Partial<AccountSummary>;
};

type AccountMenuData = Partial<AccountSummary>;

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
  details
}: {
  session: AuthSession | null;
  details?: AccountMenuData;
}) {
  const { t } = useI18n();
  const [open, setOpen] = useState(false);
  const [busy, setBusy] = useState(false);
  const [signInOpen, setSignInOpen] = useState(false);

  useEffect(() => {
    setOpen(false);
    setBusy(false);
  }, [session?.token]);

  async function handleLogout() {
    if (busy) {
      return;
    }

    const requestToken = session?.token || "";
    setBusy(true);
    try {
      if (requestToken) {
        await logout(requestToken);
      }
    } catch {
      // Local sign-out should still succeed even if backend logout fails.
    } finally {
      if (requestToken) {
        clearSessionIfToken(requestToken);
      } else {
        writeSession(null);
      }
      const currentToken = readSession()?.token || "";
      if (!currentToken || currentToken === requestToken) {
        setBusy(false);
        setOpen(false);
      }
    }
  }

  if (!session) {
    return (
      <>
        <div className="nav-actions">
          <button className="button primary" type="button" onClick={() => setSignInOpen(true)}>
            {t({ en: "Sign In", zh: "登录" })}
          </button>
        </div>
        <SignInDialog open={signInOpen} onClose={() => setSignInOpen(false)} />
      </>
    );
  }

  const initials = (session.user.username || session.user.email || session.user.id)
    .slice(0, 2)
    .toUpperCase();
  function stateText<T>(state: ResourceState<T> | undefined, format: (data: T) => string): string {
    if (!state || state.status === "idle" || state.status === "loading") {
      return t({ en: "Loading...", zh: "加载中..." });
    }
    if (state.status === "ready") {
      return format(state.data);
    }
    switch (state.failure.kind) {
      case "auth":
        return t({ en: "Sign in again", zh: "请重新登录" });
      case "disabled":
        return t({ en: "Not enabled", zh: "尚未启用" });
      case "configuration":
        return t({ en: "Setup required", zh: "需要配置" });
      case "network":
        return t({ en: "API unreachable", zh: "无法连接 API" });
      case "unavailable":
        return t({ en: "Temporarily unavailable", zh: "暂时不可用" });
      default:
        return t({ en: "Request failed", zh: "请求失败" });
    }
  }

  const capabilities = details?.capabilities?.data;
  const accountText = capabilities && !capabilities.account.enabled
    ? t({ en: "Not enabled on this API", zh: "API 尚未启用" })
    : t({ en: "Profile, identity and security", zh: "个人资料、身份与安全" });
  const subscriptionText = capabilities && !capabilities.billing.enabled
    ? t({ en: "Stripe setup required", zh: "需要配置 Stripe" })
    : stateText(details?.subscription, (value) => {
    const subscription = value as SubscriptionView;
    return `${subscription.plan} · ${subscription.status}`;
  });
  const referralText = capabilities && !capabilities.referral.enabled
    ? t({ en: "Not enabled", zh: "尚未启用" })
    : stateText(details?.referralStats, (value) => {
    const stats = value as ReferralStats;
    return `${stats.activated}/${stats.total_referred}`;
  });
  const subscriptionFailure = capabilities && !capabilities.billing.enabled
    ? "configuration"
    : details?.subscription?.status === "error"
    ? details.subscription.failure.kind
    : "";
  const referralFailure = capabilities && !capabilities.referral.enabled
    ? "disabled"
    : details?.referralStats?.status === "error"
    ? details.referralStats.failure.kind
    : "";

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
              <small>{accountText}</small>
            </Link>
            <Link className="popover-link-card" href="/billing" onClick={() => setOpen(false)}>
              <span>{t({ en: "Subscription", zh: "订阅管理" })}</span>
              <small className={subscriptionFailure ? `menu-resource-state ${subscriptionFailure}` : ""}>{subscriptionText}</small>
            </Link>
            <Link className="popover-link-card" href="/referrals" onClick={() => setOpen(false)}>
              <span>{t({ en: "Referral Center", zh: "推荐中心" })}</span>
              <small className={referralFailure ? `menu-resource-state ${referralFailure}` : ""}>
                {details?.referralStats?.status === "ready"
                  ? `${t({ en: "Activated / total", zh: "已激活 / 总数" })}: ${referralText}`
                  : referralText}
              </small>
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
  const isHome = pathname === "/";
  const [session, setSession] = useState<AuthSession | null>(null);
  const [accountDetails, setAccountDetails] = useState<AccountSummary | null>(null);
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
      return () => {
        cancelled = true;
      };
    }

    setAccountDetails({
      capabilities: loadingResource(),
      subscription: loadingResource(),
      referralStats: loadingResource()
    });

    void loadAccountSummary(session.token)
      .then((details) => {
        if (!cancelled) {
          setAccountDetails(details);
        }
      })
      .catch(() => {
        if (!cancelled) {
          const failure = describeRequestFailure(new Error("Account summary request failed."), "Account summary");
          setAccountDetails({
            capabilities: failedResource(failure),
            subscription: failedResource(failure),
            referralStats: failedResource(failure)
          });
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
    capabilities: props.accountMenuData?.capabilities ?? accountDetails?.capabilities,
    subscription: props.accountMenuData?.subscription ?? accountDetails?.subscription,
    referralStats: props.accountMenuData?.referralStats ?? accountDetails?.referralStats
  };

  return (
    <div className="app-shell">
      <header className="topbar">
        <div className="topbar-inner">
          <Link className="brand-lockup" href="/">
            <span className="brand-mark" aria-hidden="true">
              <svg viewBox="0 0 28 28" role="img">
                <path d="M6 8.5 14 4l8 4.5v9L14 22l-8-4.5v-9Z" />
                <path d="m9.5 10.5 4.5-2.6 4.5 2.6v5L14 18l-4.5-2.5v-5Z" />
              </svg>
            </span>
            <span className="brand-copy">
              <span className="brand-title">{appEnv.appName}</span>
              <span className="brand-subtitle">{t({ en: "Powered by go-modules", zh: "由 go-modules 驱动" })}</span>
            </span>
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
            />
          </div>
        </div>
      </header>

      <main className={`page-shell${isHome ? " home-page-shell" : ""}`}>
        <section className={`hero-grid${isHome ? " home-hero" : ""}`}>
          <div className="hero-main-card">
            {!isHome ? <Breadcrumbs items={breadcrumbs} /> : null}
            <span className="eyebrow">{props.eyebrow}</span>
            <h1>{props.title}</h1>
            <p>{props.description}</p>
            {props.actions ? <div className="hero-actions">{props.actions}</div> : null}
          </div>

          <div className="hero-side-stack">
            <div className="hero-side-card">
              <div className="panel-title-row compact">
                <div>
                  {props.showEnvironment !== false ? <span className="panel-kicker">{t({ en: "Environment", zh: "环境" })}</span> : null}
                  <h3>{props.sideTitle || t({ en: "Context", zh: "上下文" })}</h3>
                </div>
                {props.showEnvironment !== false ? <span className="badge">{appEnv.appUrl}</span> : null}
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

      <footer className="site-footer">
        <div className="site-footer-inner">
          <Link className="footer-brand" href="/">
            <span className="brand-mark small" aria-hidden="true">
              <svg viewBox="0 0 28 28" role="img">
                <path d="M6 8.5 14 4l8 4.5v9L14 22l-8-4.5v-9Z" />
                <path d="m9.5 10.5 4.5-2.6 4.5 2.6v5L14 18l-4.5-2.5v-5Z" />
              </svg>
            </span>
            <span>{appEnv.appName}</span>
          </Link>
          <nav className="footer-links" aria-label="Footer">
            <Link href="/docs">{t({ en: "Documentation", zh: "文档" })}</Link>
            <Link href="/pricing">{t({ en: "Pricing", zh: "价格" })}</Link>
            <Link href="/login">{t({ en: "Sign in", zh: "登录" })}</Link>
          </nav>
          <p>{t({ en: "A production-minded SaaS starter.", zh: "面向生产环境的 SaaS 启动模板。" })}</p>
        </div>
      </footer>
    </div>
  );
}
