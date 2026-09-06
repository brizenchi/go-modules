"use client";

import Link from "next/link";
import {
  useCallback,
  useEffect,
  useRef,
  useState
} from "react";
import { ResourceFailure, SignInRequired } from "@/components/resource-feedback";
import { SignInDialog } from "@/components/sign-in-dialog";
import { SiteShell } from "@/components/site-shell";
import { loadAccountSummary, type AccountSummary } from "@/lib/account-summary";
import {
  getAccountProfile,
  getReferralCode,
  type AccountProfile,
  type CapabilitiesView,
  type ReferralCodeResult,
  type ReferralStats,
  type SubscriptionView
} from "@/lib/api";
import { readSession, SESSION_EVENT } from "@/lib/auth";
import {
  buildDashboardChecklist,
  humanizePlan,
  profileCompletion,
  type DashboardChecklistItem
} from "@/lib/dashboard";
import { formatDate } from "@/lib/format";
import { useI18n } from "@/lib/i18n";
import {
  beginRequestGeneration,
  invalidateRequestGeneration,
  isCurrentRequestGeneration
} from "@/lib/request-generation";
import {
  idleResource,
  loadingResource,
  settleResource,
  type ResourceState
} from "@/lib/request-state";
import styles from "./dashboard.module.css";

function emptySummary(): AccountSummary {
  return {
    capabilities: idleResource<CapabilitiesView>(),
    subscription: idleResource<SubscriptionView>(),
    referralStats: idleResource<ReferralStats>()
  };
}

function MetricSkeleton() {
  return (
    <div className={styles.skeleton} aria-label="Loading">
      <span />
      <span />
    </div>
  );
}

export default function DashboardPage() {
  const { locale, t } = useI18n();
  const dateLocale = locale === "zh" ? "zh-CN" : "en-US";
  const [session, setSession] = useState<ReturnType<typeof readSession>>(null);
  const [sessionReady, setSessionReady] = useState(false);
  const [summary, setSummary] = useState<AccountSummary>(() => emptySummary());
  const [profileState, setProfileState] = useState<ResourceState<AccountProfile>>(idleResource());
  const [codeState, setCodeState] = useState<ResourceState<ReferralCodeResult>>(idleResource());
  const [signInOpen, setSignInOpen] = useState(false);
  const [copied, setCopied] = useState(false);
  const generationRef = useRef(0);
  const token = session?.token || "";

  const loadDashboard = useCallback(async (requestToken: string) => {
    const generation = beginRequestGeneration(generationRef);
    setCopied(false);
    setSummary((current) => ({
      capabilities: loadingResource(current.capabilities.data),
      subscription: loadingResource(current.subscription.data),
      referralStats: loadingResource(current.referralStats.data)
    }));
    setProfileState((current) => loadingResource(current.data));
    setCodeState((current) => loadingResource(current.data));

    const [nextSummary, nextProfile, nextCode] = await Promise.all([
      loadAccountSummary(requestToken),
      settleResource(getAccountProfile(requestToken), "Account profile"),
      settleResource(getReferralCode(requestToken), "Referral link")
    ]);

    if (
      readSession()?.token !== requestToken
      || !isCurrentRequestGeneration(generationRef, generation)
    ) {
      return;
    }
    setSummary(nextSummary);
    setProfileState(nextProfile);
    setCodeState(nextCode);
  }, []);

  useEffect(() => {
    const sync = () => {
      setSession(readSession());
      setSessionReady(true);
    };
    sync();
    window.addEventListener("storage", sync);
    window.addEventListener(SESSION_EVENT, sync);
    return () => {
      window.removeEventListener("storage", sync);
      window.removeEventListener(SESSION_EVENT, sync);
    };
  }, []);

  useEffect(() => {
    if (!token) {
      invalidateRequestGeneration(generationRef);
      setSummary(emptySummary());
      setProfileState(idleResource());
      setCodeState(idleResource());
      setCopied(false);
      return;
    }
    void loadDashboard(token);
    return () => invalidateRequestGeneration(generationRef);
  }, [loadDashboard, token]);

  const capabilities = summary.capabilities.data;
  const profile = profileState.data || null;
  const subscription = summary.subscription.data || null;
  const referralStats = summary.referralStats.data || null;
  const referralCode = codeState.data || null;
  const billingEnabled = capabilities?.billing.enabled ?? true;
  const referralEnabled = capabilities?.referral.enabled ?? true;
  const completion = profileCompletion(profile);
  const checklist = buildDashboardChecklist({
    profile,
    subscription,
    referralLink: referralCode?.link || "",
    billingEnabled,
    referralEnabled
  });
  const completedItems = checklist.filter((item) => item.complete).length;
  const checklistProgress = checklist.length > 0
    ? Math.round((completedItems / checklist.length) * 100)
    : 0;
  const displayName = profile?.username || session?.user.username || "";

  function checklistCopy(item: DashboardChecklistItem) {
    if (item.key === "profile") {
      return {
        title: t({ en: "Complete your profile", zh: "完善账户资料" }),
        detail: t({ en: "Add a display name and confirm your identity.", zh: "补充显示名称并确认账户身份。" })
      };
    }
    if (item.key === "billing") {
      return {
        title: t({ en: "Choose your plan", zh: "选择适合的套餐" }),
        detail: t({ en: "Activate a subscription for this workspace.", zh: "为当前工作区启用正式订阅。" })
      };
    }
    return {
      title: t({ en: "Get your invitation ready", zh: "获取专属邀请链接" }),
      detail: t({ en: "Share your link and track every reward.", zh: "分享链接并跟踪每一笔奖励。" })
    };
  }

  async function copyReferralLink() {
    if (!referralCode?.link) {
      return;
    }
    try {
      await navigator.clipboard.writeText(referralCode.link);
      setCopied(true);
    } catch {
      setCopied(false);
    }
  }

  const pageTitle = displayName
    ? t({ en: `Welcome back, ${displayName}.`, zh: `欢迎回来，${displayName}。` })
    : t({ en: "Your workspace, at a glance.", zh: "你的工作区，一目了然。" });

  return (
    <SiteShell
      eyebrow={t({ en: "Workspace overview", zh: "工作台总览" })}
      title={pageTitle}
      description={t({
        en: "See your plan, account balance, referral progress, and the next action worth taking.",
        zh: "在一个页面查看套餐、账户积分、推荐进度和下一步操作。"
      })}
      accountMenuData={summary}
      sideTitle={t({ en: "Workspace readiness", zh: "工作区准备度" })}
      sideBody={token ? (
        <div className={styles.readinessMini}>
          <strong>{checklistProgress}%</strong>
          <span>{completedItems} / {checklist.length} {t({ en: "essentials complete", zh: "项关键设置已完成" })}</span>
          <div><span style={{ width: `${checklistProgress}%` }} /></div>
        </div>
      ) : (
        <p>{t({ en: "Sign in to load live workspace data.", zh: "登录后即可加载工作区实时数据。" })}</p>
      )}
      actions={token ? (
        <button className="button" type="button" onClick={() => void loadDashboard(token)}>
          {t({ en: "Refresh data", zh: "刷新数据" })}
        </button>
      ) : undefined}
    >
      <div className={styles.dashboard}>
        {sessionReady && !session ? (
          <SignInRequired
            title={t({ en: "Open your workspace", zh: "登录进入工作台" })}
            message={t({ en: "Your plan, credits, referrals, and account settings appear here after sign-in.", zh: "登录后即可在这里管理套餐、积分、推荐奖励和账户设置。" })}
            actionLabel={t({ en: "Sign in", zh: "立即登录" })}
            onSignIn={() => setSignInOpen(true)}
          />
        ) : null}

        {session ? (
          <>
            <section className={styles.metrics} aria-label={t({ en: "Account metrics", zh: "账户指标" })}>
              <article className={`${styles.metric} ${styles.planMetric}`}>
                <div className={styles.metricTopline}>
                  <span>{t({ en: "Current plan", zh: "当前套餐" })}</span>
                  <span className={`${styles.stateDot} ${styles[subscription?.status || "idle"] || ""}`}>
                    {subscription?.status || t({ en: "Loading", zh: "加载中" })}
                  </span>
                </div>
                {summary.subscription.status === "loading" || summary.subscription.status === "idle" ? <MetricSkeleton /> : null}
                {summary.subscription.status === "ready" ? (
                  <>
                    <strong>{humanizePlan(subscription?.plan || "")}</strong>
                    <p>{subscription?.billing_cycle
                      ? subscription.billing_cycle === "yearly"
                        ? t({ en: "yearly billing", zh: "按年计费" })
                        : t({ en: "monthly billing", zh: "按月计费" })
                      : t({ en: "Flexible account access", zh: "灵活的账户权益" })}</p>
                  </>
                ) : null}
                {summary.subscription.status === "error" ? <ResourceFailure compact failure={summary.subscription.failure} /> : null}
                <Link href="/billing">{t({ en: "Manage subscription", zh: "管理订阅" })}<span>↗</span></Link>
              </article>

              <article className={styles.metric}>
                <div className={styles.metricTopline}>
                  <span>{t({ en: "Available credits", zh: "可用积分" })}</span>
                  <span className={styles.metricGlyph}>＋</span>
                </div>
                {profileState.status === "loading" || profileState.status === "idle" ? <MetricSkeleton /> : null}
                {profileState.status === "ready" ? (
                  <>
                    <strong>{profile?.credits.toLocaleString() || "0"}</strong>
                    <p>{t({ en: "Ready to use across your product.", zh: "可用于当前产品内的功能。" })}</p>
                  </>
                ) : null}
                {profileState.status === "error" ? <ResourceFailure compact failure={profileState.failure} /> : null}
                <Link href="/billing">{t({ en: "Get more credits", zh: "获取更多积分" })}<span>↗</span></Link>
              </article>

              <article className={styles.metric}>
                <div className={styles.metricTopline}>
                  <span>{t({ en: "Activated referrals", zh: "已激活推荐" })}</span>
                  <span className={styles.metricGlyph}>↗</span>
                </div>
                {summary.referralStats.status === "loading" || summary.referralStats.status === "idle" ? <MetricSkeleton /> : null}
                {summary.referralStats.status === "ready" ? (
                  <>
                    <strong>{referralStats?.activated.toLocaleString() || "0"}</strong>
                    <p>{t({ en: `${referralStats?.pending || 0} pending · ${referralStats?.total_reward_credits || 0} credits earned`, zh: `${referralStats?.pending || 0} 个待激活 · 已获 ${referralStats?.total_reward_credits || 0} 积分` })}</p>
                  </>
                ) : null}
                {summary.referralStats.status === "error" ? <ResourceFailure compact failure={summary.referralStats.failure} /> : null}
                <Link href="/referrals">{t({ en: "Open referral center", zh: "打开推荐中心" })}<span>↗</span></Link>
              </article>

              <article className={styles.metric}>
                <div className={styles.metricTopline}>
                  <span>{t({ en: "Profile complete", zh: "资料完成度" })}</span>
                  <span className={styles.metricGlyph}>◎</span>
                </div>
                {profileState.status === "loading" || profileState.status === "idle" ? <MetricSkeleton /> : null}
                {profileState.status === "ready" ? (
                  <>
                    <strong>{completion}%</strong>
                    <div className={styles.completionBar}><span style={{ width: `${completion}%` }} /></div>
                  </>
                ) : null}
                {profileState.status === "error" ? <ResourceFailure compact failure={profileState.failure} /> : null}
                <Link href="/account">{t({ en: "Update account", zh: "完善账户" })}<span>↗</span></Link>
              </article>
            </section>

            <section className={styles.detailGrid}>
              <article className={styles.setupCard}>
                <div className={styles.sectionHeading}>
                  <div>
                    <span>{t({ en: "Getting started", zh: "开始使用" })}</span>
                    <h2>{t({ en: "A short path to a ready account.", zh: "用最短路径完成账户设置。" })}</h2>
                  </div>
                  <strong>{completedItems}/{checklist.length}</strong>
                </div>
                <div className={styles.checklist}>
                  {checklist.map((item, index) => {
                    const itemCopy = checklistCopy(item);
                    return (
                      <Link href={item.href} className={styles.checkItem} key={item.key}>
                        <span className={`${styles.checkMark}${item.complete ? ` ${styles.complete}` : ""}`}>
                          {item.complete ? "✓" : String(index + 1).padStart(2, "0")}
                        </span>
                        <span>
                          <strong>{itemCopy.title}</strong>
                          <small>{itemCopy.detail}</small>
                        </span>
                        <span aria-hidden="true">↗</span>
                      </Link>
                    );
                  })}
                </div>
              </article>

              <article className={styles.nextCard}>
                <span className={styles.cardKicker}>{t({ en: "Next billing moment", zh: "下一个账单节点" })}</span>
                <div className={styles.dateBlock}>
                  <strong>{subscription?.current_period_end ? formatDate(subscription.current_period_end, dateLocale) : "—"}</strong>
                  <span>{subscription?.cancel_at_period_end
                    ? t({ en: "Subscription ends at this date", zh: "订阅将在该日期结束" })
                    : t({ en: "Current billing period end", zh: "当前计费周期结束时间" })}</span>
                </div>
                <div className={styles.nextActions}>
                  <Link className="button primary" href="/billing">{t({ en: "Billing details", zh: "查看账单" })}</Link>
                  <Link className="button" href="/account">{t({ en: "Account settings", zh: "账户设置" })}</Link>
                </div>
              </article>
            </section>

            <section className={styles.inviteStrip}>
              <div>
                <span className={styles.cardKicker}>{t({ en: "Your invitation", zh: "你的专属邀请" })}</span>
                <h2>{t({ en: "Good products grow through people.", zh: "让好产品通过真实分享自然增长。" })}</h2>
                <p>{t({ en: "Invite a friend and follow registration, activation, and rewards from one place.", zh: "邀请好友，并在一个页面跟踪注册、激活和奖励进度。" })}</p>
              </div>
              <div className={styles.inviteAction}>
                <code>{referralCode?.code || "—"}</code>
                <button className="button primary" type="button" disabled={!referralCode?.link} onClick={() => void copyReferralLink()}>
                  {copied ? t({ en: "Link copied", zh: "链接已复制" }) : t({ en: "Copy invite link", zh: "复制邀请链接" })}
                </button>
              </div>
              {codeState.status === "error" ? <ResourceFailure compact failure={codeState.failure} /> : null}
            </section>
          </>
        ) : sessionReady ? null : (
          <div className={styles.loadingPage}><MetricSkeleton /></div>
        )}
      </div>
      <SignInDialog
        open={signInOpen}
        onClose={() => setSignInOpen(false)}
        onSuccess={() => setSignInOpen(false)}
      />
    </SiteShell>
  );
}
