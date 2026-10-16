"use client";

import Link from "next/link";
import { useCallback, useEffect, useRef, useState } from "react";
import { SiteShell } from "@/components/site-shell";
import { EmptyState, Notice, Panel, DetailRows } from "@/components/ui";
import { ResourceFailure, SignInRequired } from "@/components/resource-feedback";
import {
  getCapabilities,
  getReferralCode,
  getReferralStats,
  listReferrals,
  type CapabilitiesView,
  type ReferralCodeResult,
  type ReferralStats
} from "@/lib/api";
import { readSession, SESSION_EVENT } from "@/lib/auth";
import { appEnv } from "@/lib/env";
import { useI18n } from "@/lib/i18n";
import { SignInDialog } from "@/components/sign-in-dialog";
import styles from "./referrals.module.css";
import { usePageReturn } from "@/components/use-page-return";
import { formatDate } from "@/lib/format";
import {
  describeRequestFailure,
  idleResource,
  loadingResource,
  settleResource,
  type RequestFailure,
  type ResourceState
} from "@/lib/request-state";
import {
  beginRequestGeneration,
  invalidateRequestGeneration,
  isCurrentRequestGeneration
} from "@/lib/request-generation";

export default function ReferralsPage() {
  const { locale, t } = useI18n();
  const dateLocale = locale === "zh" ? "zh-CN" : "en-US";
  const [signInOpen, setSignInOpen] = useState(false);
  const copy = t({
    en: {
      eyebrow: "Invite friends", title: "Referral rewards",
      description: "Invite new users and track each invitation from signup to activation. Rewards, when offered, follow the published site rules.",
      rules: "How invitations work", signup: "New accounts only", signupDetail: "An invitation is linked once, at signup. You cannot invite yourself or reassign an existing account.",
      qualify: "First paid subscription", qualifyDetail: "Signup creates a pending invitation. A free trial alone does not qualify; the subscription must become paid within the activation window.",
      reward: "Reward policy", rewardDetail: "On invited signup, the inviter gets 20 credits and the friend gets 10. A qualifying first paid subscription grants another 20 / 10. Each stage is rewarded once.",
      link: "My invite link", linkHelp: "Copy your personal link and share it with a friend who has not registered yet.", code: "Invite code", shareLink: "Invitation link",
      copy: "Copy link", copied: "Invite link copied. Ready to share.", copyFailed: "Couldn't copy the link. Select the full link above and copy it manually.",
      stats: "Invitation overview", statsHelp: "See registrations, qualifying subscriptions, and recorded reward credits.",
      total: "Registered", activated: "Activated", pending: "Awaiting subscription", credits: "Paid activation rewards", expired: "Expired",
      history: "Invitation history", historyHelp: "Signup records the invitation. Activation records a qualifying subscription.",
      person: "Invited account", state: "Status", joined: "Registered", deadline: "Activation deadline", activatedAt: "Activated", noDeadline: "No time limit",
      loadingLink: "Loading your invite link…", loadingStats: "Loading your invitation overview…", loadingList: "Loading invitations…",
      emptyLink: "Your invite link will appear here.", emptyStats: "Your invitation overview will appear here.", emptyList: "No invitations yet. Share your link; a new registration will appear here.",
      signIn: "Sign in to invite friends", signInHelp: "Create your own invitation link and follow your friends' progress.", signInButton: "Sign in",
      refresh: "Try again", previous: "Previous", next: "Next", page: "Page", of: "of", records: "invitations",
      unavailable: "Invitations are currently unavailable", unavailableDetail: "You can still use your account. Check back when invitations are available.",
      missingLink: "Your invitation link is not ready", missingLinkDetail: "Please contact support to have sharing enabled for this site.",
      trialNote: "A free trial alone does not activate an invitation. After qualifying paid activation, both users can view their rewards in Credits.",
      demoNote: "Demo: registrations and invitation records are real. Payments use Stripe test mode. Open your link in another browser or private window, register a new account, complete a qualifying test subscription, then return here to see the updated records.",
      testHelp: "View test payment instructions", pendingHelp: "Registered; awaiting a qualifying paid subscription.", activatedHelp: "Qualifying subscription recorded.", expiredHelp: "The activation deadline passed without a qualifying subscription."
    },
    zh: {
      eyebrow: "邀请好友", title: "推荐奖励",
      description: "邀请新用户注册，从注册到激活，每一笔邀请都有记录可查。好友通过邀请注册后，邀请人获得 20 积分、好友获得 10 积分，无需购买。首次符合条件的付费订阅再分别奖励 20 / 10，每个阶段仅发一次。",
      rules: "邀请如何生效", signup: "首次注册时绑定", signupDetail: "每个新账号只能绑定一次邀请。不能邀请自己，已有账号不能重新绑定。",
      qualify: "首次符合条件的付费订阅", qualifyDetail: "好友注册后先进入待激活状态。仅开通免费试用不会激活邀请，需要在激活期限内完成付费订阅。",
      reward: "奖励规则", rewardDetail: "好友通过邀请注册后，邀请人获得 20 积分、好友获得 10 积分，无需购买。首次符合条件的付费订阅再分别奖励 20 / 10，每个阶段仅发一次。",
      link: "我的邀请链接", linkHelp: "复制你的专属链接，分享给尚未注册的好友。", code: "邀请码", shareLink: "邀请链接",
      copy: "复制邀请链接", copied: "邀请链接已复制，可以分享给好友了。", copyFailed: "复制失败，请选中上方完整链接并手动复制。",
      stats: "邀请概况", statsHelp: "查看注册人数、订阅激活情况和记录的奖励积分。",
      total: "已注册", activated: "已激活", pending: "待订阅激活", credits: "付费激活奖励", expired: "已过期",
      history: "邀请记录", historyHelp: "好友注册后显示邀请记录；完成符合条件的订阅后更新激活状态。",
      person: "受邀账号", state: "状态", joined: "注册时间", deadline: "激活截止时间", activatedAt: "激活时间", noDeadline: "不限时",
      loadingLink: "正在生成邀请链接…", loadingStats: "正在加载邀请概况…", loadingList: "正在加载邀请记录…",
      emptyLink: "你的专属邀请链接将在这里显示。", emptyStats: "邀请概况将在这里显示。", emptyList: "还没有邀请记录。把链接分享给好友，好友首次注册后会出现在这里。",
      signIn: "登录后邀请好友", signInHelp: "获取你的专属邀请链接，查看好友的注册和激活进度。", signInButton: "登录",
      refresh: "重试", previous: "上一页", next: "下一页", page: "第", of: "/", records: "条邀请",
      unavailable: "邀请功能暂未开放", unavailableDetail: "你可以继续使用账号，稍后再来查看邀请功能。",
      missingLink: "邀请链接暂未就绪", missingLinkDetail: "请联系网站支持人员开通分享链接。",
      trialNote: "注册奖励无需付费；免费试用不会触发额外付费奖励。双方在“我的积分”查看两类奖励。",
      demoNote: "演示说明：注册和邀请记录都是真实的，支付使用 Stripe 测试环境。请在另一浏览器或无痕窗口打开链接，用新账号注册并完成符合条件的测试订阅，再回到这里查看更新后的记录。",
      testHelp: "查看测试支付说明", pendingHelp: "好友已注册，等待符合条件的付费订阅。", activatedHelp: "已记录符合条件的订阅激活。", expiredHelp: "激活期限内未完成符合条件的订阅。"
    }
  });
  const [session, setSession] = useState<ReturnType<typeof readSession>>(null);
  const [sessionReady, setSessionReady] = useState(false);
  const [capabilitiesState, setCapabilitiesState] = useState<ResourceState<CapabilitiesView>>(idleResource());
  const [codeState, setCodeState] = useState<ResourceState<ReferralCodeResult>>(idleResource());
  const [statsState, setStatsState] = useState<ResourceState<ReferralStats>>(idleResource());
  const [listState, setListState] = useState<ResourceState<Awaited<ReturnType<typeof listReferrals>>>>(idleResource());
  const [status, setStatus] = useState("");
  const [actionFailure, setActionFailure] = useState<RequestFailure | null>(null);
  const capabilitiesGenerationRef = useRef(0);
  const capabilitiesMountedRef = useRef(false);
  const codeGenerationRef = useRef(0);
  const statsGenerationRef = useRef(0);
  const listGenerationRef = useRef(0);
  const copyGenerationRef = useRef(0);
  const code = codeState.data;
  const stats = statsState.data;
  const items = listState.data?.items || [];
  const page = listState.data?.page || 1;
  const totalPages = Math.max(1, Math.ceil((listState.data?.total || 0) / (listState.data?.limit || 20)));
  const referralEnabled = capabilitiesState.status === "ready" && capabilitiesState.data.referral.enabled;
  const referralCapabilityFailure: RequestFailure | null = capabilitiesState.status === "error"
    ? capabilitiesState.failure
    : capabilitiesState.status === "ready" && !capabilitiesState.data.referral.enabled
      ? {
          kind: "disabled",
          title: copy.unavailable,
          message: copy.unavailableDetail,
          retryable: false
        }
      : null;
  const sessionToken = session?.token || "";

  function displayFailure(failure: RequestFailure): RequestFailure {
    return {
      ...failure,
      title: t({ en: "Invitation details are unavailable", zh: "暂时无法加载邀请信息" }),
      message: t({ en: "Please try again shortly. If this continues, contact support.", zh: "请稍后重试。如果仍无法加载，请联系网站支持人员。" })
    };
  }

  const loadCodeState = useCallback(async (token: string) => {
    if (readSession()?.token !== token) return;
    const generation = beginRequestGeneration(codeGenerationRef);
    setCodeState(loadingResource());
    const nextState = await settleResource(getReferralCode(token), "Referral link");
    if (
      readSession()?.token === token
      && isCurrentRequestGeneration(codeGenerationRef, generation)
    ) {
      setCodeState(nextState);
    }
  }, []);

  const loadStatsState = useCallback(async (token: string) => {
    if (readSession()?.token !== token) return;
    const generation = beginRequestGeneration(statsGenerationRef);
    setStatsState(loadingResource());
    const nextState = await settleResource(getReferralStats(token), "Referral statistics");
    if (
      readSession()?.token === token
      && isCurrentRequestGeneration(statsGenerationRef, generation)
    ) {
      setStatsState(nextState);
    }
  }, []);

  const loadListState = useCallback(async (token: string, requestedPage = 1) => {
    if (readSession()?.token !== token) return;
    const generation = beginRequestGeneration(listGenerationRef);
    setListState((current) => loadingResource(current.data));
    const nextState = await settleResource(
      listReferrals(token, requestedPage),
      "Referral history"
    );
    if (
      readSession()?.token === token
      && isCurrentRequestGeneration(listGenerationRef, generation)
    ) {
      setListState(nextState);
    }
  }, []);

  const load = useCallback(async (token: string) => {
    await Promise.all([
      loadCodeState(token),
      loadStatsState(token),
      loadListState(token)
    ]);
  }, [loadCodeState, loadListState, loadStatsState]);
  const updateOnReturn = useCallback(() => { if (sessionToken && referralEnabled) void load(sessionToken); }, [sessionToken, referralEnabled, load]);
  usePageReturn(updateOnReturn);

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
    capabilitiesMountedRef.current = true;
    void loadCapabilitiesState();
    return () => {
      capabilitiesMountedRef.current = false;
      invalidateRequestGeneration(capabilitiesGenerationRef);
    };
  }, []);

  useEffect(() => {
    if (!sessionToken || !referralEnabled) {
      invalidateRequestGeneration(codeGenerationRef);
      invalidateRequestGeneration(statsGenerationRef);
      invalidateRequestGeneration(listGenerationRef);
      invalidateRequestGeneration(copyGenerationRef);
      setCodeState(idleResource());
      setStatsState(idleResource());
      setListState(idleResource());
      setStatus("");
      setActionFailure(null);
      return;
    }
    void load(sessionToken);
    return () => {
      invalidateRequestGeneration(codeGenerationRef);
      invalidateRequestGeneration(statsGenerationRef);
      invalidateRequestGeneration(listGenerationRef);
      invalidateRequestGeneration(copyGenerationRef);
    };
  }, [sessionToken, referralEnabled, load]);

  async function loadCapabilitiesState() {
    if (!capabilitiesMountedRef.current) return;
    const generation = beginRequestGeneration(capabilitiesGenerationRef);
    setCapabilitiesState((current) => loadingResource(current.data));
    const nextState = await settleResource(getCapabilities(), "API capabilities");
    if (
      capabilitiesMountedRef.current
      && isCurrentRequestGeneration(capabilitiesGenerationRef, generation)
    ) {
      setCapabilitiesState(nextState);
    }
  }

  async function copyLink() {
    if (!code?.link) {
      return;
    }
    const requestToken = sessionToken;
    const generation = beginRequestGeneration(copyGenerationRef);
    setActionFailure(null);
    setStatus("");
    try {
      await navigator.clipboard.writeText(code.link);
      if (
        readSession()?.token === requestToken
        && isCurrentRequestGeneration(copyGenerationRef, generation)
      ) {
        setStatus(copy.copied);
      }
    } catch (error) {
      if (
        readSession()?.token === requestToken
        && isCurrentRequestGeneration(copyGenerationRef, generation)
      ) {
        setActionFailure({ ...describeRequestFailure(error, "Clipboard access"), message: copy.copyFailed });
      }
    }
  }

  return (
    <SiteShell
      eyebrow={copy.eyebrow}
      title={copy.title}
      description={copy.description}
      accountMenuData={{ capabilities: capabilitiesState, referralStats: statsState }}
      sideTitle={copy.rules}
      showEnvironment={false}
      sideBody={<DetailRows rows={[
        { label: copy.signup, value: t({ en: "One invitation per new account.", zh: "每个新账号仅绑定一次邀请。" }) },
        { label: t({ en: "Paid subscription", zh: "好友订阅" }), value: t({ en: "Your friend subscribes within the activation window.", zh: "好友在奖励期限内首次完成付费订阅。" }) },
        { label: copy.reward, value: t({ en: "On invited signup, the inviter gets 20 credits and the friend gets 10. A qualifying first paid subscription grants another 20 / 10. Each stage is rewarded once.", zh: "好友通过邀请注册后，邀请人获得 20 积分、好友获得 10 积分，无需购买。首次符合条件的付费订阅再分别奖励 20 / 10，每个阶段仅发一次。" }) }
      ]} />}
      toc={[
        { id: "my-link", label: copy.link },
        { id: "stats", label: copy.stats },
        { id: "history", label: copy.history }
      ]}
    >
      <div className="page-grid">
        {sessionReady && !session ? (
          <div className="span-12">
            <SignInRequired title={copy.signIn} message={copy.signInHelp} actionLabel={copy.signInButton} onSignIn={() => setSignInOpen(true)} />
          </div>
        ) : null}

        {referralCapabilityFailure ? (
          <div className="span-12">
            <ResourceFailure
              failure={capabilitiesState.status === "error" ? displayFailure(referralCapabilityFailure) : referralCapabilityFailure}
              retryLabel={copy.refresh}
              onRetry={capabilitiesState.status === "error" ? () => void loadCapabilitiesState() : undefined}
            />
          </div>
        ) : null}

        <Panel className="span-5" title={copy.link} subtitle={copy.linkHelp}>
          <div id="my-link" />
          {code ? (
            <div className="details-list">
              <div className="details-row">
                <strong>{copy.code}</strong>
                <span className="inline-code">{code.code}</span>
              </div>
              <div className="details-row">
                <strong>{copy.shareLink}</strong>
                <span className={`inline-code ${styles.shareLink}`}>{code.link || "—"}</span>
              </div>
              {!code.link ? (
                <ResourceFailure
                  failure={{
                    kind: "configuration",
                    title: copy.missingLink,
                    message: copy.missingLinkDetail,
                    retryable: false
                  }}
                />
              ) : null}
            </div>
          ) : codeState.status === "error" ? (
            <ResourceFailure
              failure={displayFailure(codeState.failure)}
              retryLabel={copy.refresh}
              onRetry={session ? () => void loadCodeState(session.token) : undefined}
            />
          ) : (
            <EmptyState>
              {codeState.status === "loading"
                ? copy.loadingLink
                : session
                  ? copy.emptyLink
                  : copy.signInHelp}
            </EmptyState>
          )}
          <div className="button-row">
            <button className="button primary" disabled={!code?.link} onClick={() => void copyLink()}>
              {copy.copy}
            </button>
          </div>
          {status ? <div role="status"><Notice tone="success">{status}</Notice></div> : null}
          {actionFailure ? <ResourceFailure failure={actionFailure} /> : null}
        </Panel>

        <Panel className="span-7" title={copy.stats} subtitle={copy.statsHelp}>
          <div id="stats" />
          {stats ? (
            <div className={`stats-grid ${styles.stats}`}>
              <div className="stat-card">
                <span className="stat-label">{copy.total}</span>
                <span className="stat-value">{stats.total_referred}</span>
              </div>
              <div className="stat-card">
                <span className="stat-label">{copy.activated}</span>
                <span className="stat-value">{stats.activated}</span>
              </div>
              <div className="stat-card">
                <span className="stat-label">{copy.pending}</span>
                <span className="stat-value">{stats.pending}</span>
              </div>
              <div className="stat-card">
                <span className="stat-label">{copy.credits}</span>
                <span className="stat-value">{stats.total_reward_credits}</span>
              </div>
            </div>
          ) : statsState.status === "error" ? (
            <ResourceFailure
              failure={displayFailure(statsState.failure)}
              retryLabel={copy.refresh}
              onRetry={session ? () => void loadStatsState(session.token) : undefined}
            />
          ) : (
            <EmptyState>{statsState.status === "loading" ? copy.loadingStats : copy.emptyStats}</EmptyState>
          )}
        </Panel>

        <Panel className="span-12" title={copy.history} subtitle={copy.historyHelp}>
          <div id="history" />
          {listState.status === "error" ? (
            <ResourceFailure failure={displayFailure(listState.failure)} retryLabel={copy.refresh} onRetry={session ? () => void loadListState(session.token, page) : undefined} />
          ) : null}
          {items.length > 0 ? (
            <>
              <div className={styles.history} aria-busy={listState.status === "loading"}>
                <table className="table">
                  <thead><tr>
                    <th>{copy.person}</th><th>{copy.state}</th><th>{copy.credits}</th>
                    <th>{copy.joined}</th><th>{copy.deadline}</th><th>{copy.activatedAt}</th>
                  </tr></thead>
                  <tbody>{items.map((item) => (
                    <tr key={item.id}>
                      <td><span className={styles.account} title={item.referee_id}>{item.referee_id}</span></td>
                      <td><span className={`${styles.status} ${styles[item.status] || ""}`} title={item.status === "pending" ? copy.pendingHelp : item.status === "activated" ? copy.activatedHelp : item.status === "expired" ? copy.expiredHelp : undefined}>
                        {item.status === "pending" ? copy.pending : item.status === "activated" ? copy.activated : item.status === "expired" ? copy.expired : item.status}
                      </span></td>
                      <td>{item.reward_credits}</td><td>{formatDate(item.created_at, dateLocale)}</td>
                      <td>{item.expires_at ? formatDate(item.expires_at, dateLocale) : copy.noDeadline}</td>
                      <td>{formatDate(item.activated_at, dateLocale)}</td>
                    </tr>
                  ))}</tbody>
                </table>
              </div>
              <nav className={styles.pagination} aria-label={copy.history}>
                <span aria-live="polite">{copy.page} {page} {copy.of} {totalPages} · {listState.data?.total} {copy.records}</span>
                <div className="button-row">
                  <button className="button" type="button" disabled={!session || page <= 1 || listState.status === "loading"} onClick={() => session && void loadListState(session.token, page - 1)}>{copy.previous}</button>
                  <button className="button" type="button" disabled={!session || page >= totalPages || listState.status === "loading"} onClick={() => session && void loadListState(session.token, page + 1)}>{copy.next}</button>
                </div>
              </nav>
            </>
          ) : listState.status !== "error" ? (
            <EmptyState>{listState.status === "loading" ? copy.loadingList : session ? copy.emptyList : copy.signInHelp}</EmptyState>
          ) : null}
          <p className="muted">{copy.signupDetail} {copy.trialNote} {copy.rewardDetail}</p>
        </Panel>
        {appEnv.demoMode ? <div className="span-12"><Notice>
          <p>{copy.demoNote}</p><Link className="button" href="/#test-payment">{copy.testHelp}</Link>
        </Notice></div> : null}
      </div>
      <SignInDialog open={signInOpen} onClose={() => setSignInOpen(false)} />
    </SiteShell>
  );
}
