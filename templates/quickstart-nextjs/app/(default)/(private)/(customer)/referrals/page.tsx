"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { SiteShell } from "@/components/site-shell";
import { EmptyState, Notice, Panel } from "@/components/ui";
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
import { useI18n } from "@/lib/i18n";
import { SignInDialog } from "@/components/sign-in-dialog";
import styles from "./referrals.module.css";
import { usePageReturn } from "@/components/use-page-return";
import { formatDate } from "@/lib/format";
import {
  describeRequestFailure,
  idleResource,
  loadingResource,
  startResourceRefresh,
  reconcileResourceRefresh,
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
      eyebrow: "Invite friends", title: "Referral rewards", description: "",
      rewardUnit: "Credits", you: "You", friend: "Friend", signup: "Signup", payment: "First paid subscription",
      link: "Your invite link", shareLink: "Invitation link", copy: "Copy link", copied: "Link copied.",
      copyFailed: "Couldn't copy the link. Select the link and copy it manually.",
      stats: "Invitation overview", total: "Registered", activated: "Paid", pending: "Not paid", credits: "Reward credits", expired: "Expired",
      history: "Invitation history", person: "Invited account", state: "Status", joined: "Registered", deadline: "Activation deadline", activatedAt: "Activated", noDeadline: "No time limit",
      loadingLink: "Loading invite link…", loadingStats: "Loading overview…", loadingList: "Loading invitations…",
      emptyLink: "Your invite link is not ready yet.", emptyStats: "No invitation data yet.", emptyList: "No invitations yet.",
      signIn: "Sign in to invite friends", signInHelp: "Sign in to get your invite link.", signInButton: "Sign in",
      refresh: "Try again", previous: "Previous", next: "Next", page: "Page", of: "of", records: "invitations",
      unavailable: "Invitations are currently unavailable", unavailableDetail: "Please check back later.",
      missingLink: "Your invitation link is not ready", missingLinkDetail: "Please contact support.",
      pendingHelp: "Awaiting a qualifying payment.", activatedHelp: "Qualifying payment recorded.", expiredHelp: "The activation deadline has passed."
    },
    zh: {
      eyebrow: "邀请好友", title: "推荐奖励", description: "",
      rewardUnit: "奖励积分", you: "你", friend: "好友", signup: "注册", payment: "首次付费订阅",
      link: "我的邀请链接", shareLink: "邀请链接", copy: "复制链接", copied: "链接已复制。", copyFailed: "复制失败，请选中链接后手动复制。",
      stats: "邀请概况", total: "已注册", activated: "已付费", pending: "未付费", credits: "奖励积分", expired: "已过期",
      history: "邀请记录", person: "受邀账号", state: "状态", joined: "注册时间", deadline: "激活截止", activatedAt: "激活时间", noDeadline: "不限时",
      loadingLink: "正在读取邀请链接…", loadingStats: "正在读取邀请概况…", loadingList: "正在读取邀请记录…",
      emptyLink: "邀请链接暂未就绪。", emptyStats: "暂无邀请数据。", emptyList: "暂无邀请记录。",
      signIn: "登录后邀请好友", signInHelp: "登录后获取专属邀请链接。", signInButton: "登录",
      refresh: "重试", previous: "上一页", next: "下一页", page: "第", of: "/", records: "条邀请",
      unavailable: "邀请功能暂未开放", unavailableDetail: "请稍后再来查看。",
      missingLink: "邀请链接暂未就绪", missingLinkDetail: "请联系网站支持人员。",
      pendingHelp: "等待符合条件的首次付费。", activatedHelp: "已记录符合条件的付费。", expiredHelp: "已超过激活期限。"
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

  const loadCodeState = useCallback(async (token: string, background = false) => {
    if (readSession()?.token !== token) return;
    const generation = beginRequestGeneration(codeGenerationRef);
    setCodeState(current => startResourceRefresh(current, background));
    const nextState = await settleResource(getReferralCode(token), "Referral link");
    if (
      readSession()?.token === token
      && isCurrentRequestGeneration(codeGenerationRef, generation)
    ) {
      setCodeState(current => reconcileResourceRefresh(current, nextState, background));
    }
  }, []);

  const loadStatsState = useCallback(async (token: string, background = false) => {
    if (readSession()?.token !== token) return;
    const generation = beginRequestGeneration(statsGenerationRef);
    setStatsState(current => startResourceRefresh(current, background));
    const nextState = await settleResource(getReferralStats(token), "Referral statistics");
    if (
      readSession()?.token === token
      && isCurrentRequestGeneration(statsGenerationRef, generation)
    ) {
      setStatsState(current => reconcileResourceRefresh(current, nextState, background));
    }
  }, []);

  const loadListState = useCallback(async (token: string, requestedPage = 1, background = false) => {
    if (readSession()?.token !== token) return;
    const generation = beginRequestGeneration(listGenerationRef);
    setListState(current => startResourceRefresh(current, background));
    const nextState = await settleResource(
      listReferrals(token, requestedPage),
      "Referral history"
    );
    if (
      readSession()?.token === token
      && isCurrentRequestGeneration(listGenerationRef, generation)
    ) {
      setListState(current => reconcileResourceRefresh(current, nextState, background));
    }
  }, []);

  const load = useCallback(async (token: string, background = false, requestedPage = 1) => {
    await Promise.all([
      loadCodeState(token, background),
      loadStatsState(token, background),
      loadListState(token, requestedPage, background)
    ]);
  }, [loadCodeState, loadListState, loadStatsState]);
  const updateOnReturn = useCallback(() => { if (sessionToken && referralEnabled) void load(sessionToken, true, page); }, [sessionToken, referralEnabled, load, page]);
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
    >
      <div className={styles.layout}>
        {sessionReady && !session ? (
          <div className={styles.full}>
            <SignInRequired title={copy.signIn} message={copy.signInHelp} actionLabel={copy.signInButton} onSignIn={() => setSignInOpen(true)} />
          </div>
        ) : null}

        {referralCapabilityFailure ? (
          <div className={styles.full}>
            <ResourceFailure
              failure={capabilitiesState.status === "error" ? displayFailure(referralCapabilityFailure) : referralCapabilityFailure}
              retryLabel={copy.refresh}
              onRetry={capabilitiesState.status === "error" ? () => void loadCapabilitiesState() : undefined}
            />
          </div>
        ) : null}

        {session && !referralCapabilityFailure ? <>
        <Panel className={styles.card} title={copy.link}>
          <div id="my-link" />
          {code ? (
            <div className={styles.linkField}>
              <label className="sr-only" htmlFor="referral-link">{copy.shareLink}</label>
              <div className={styles.copyRow}>
                <input id="referral-link" type="text" readOnly value={code.link || ""} onFocus={event => event.currentTarget.select()} />
                <button className="button primary" disabled={!code.link} onClick={() => void copyLink()}>{copy.copy}</button>
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
          <table className={styles.rewards}>
            <thead><tr><th scope="col">{copy.rewardUnit}</th><th scope="col">{copy.you}</th><th scope="col">{copy.friend}</th></tr></thead>
            <tbody>
              <tr><th scope="row">{copy.signup}</th><td>+20</td><td>+10</td></tr>
              <tr><th scope="row">{copy.payment}</th><td>+20</td><td>+10</td></tr>
            </tbody>
          </table>
          {status ? <div role="status"><Notice tone="success">{status}</Notice></div> : null}
          {actionFailure ? <ResourceFailure failure={actionFailure} /> : null}
        </Panel>

        <Panel className={styles.card} title={copy.stats}>
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

        <Panel className={`${styles.card} ${styles.full}`} title={copy.history}>
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
              {totalPages > 1 ? <nav className={styles.pagination} aria-label={copy.history}>
                <span aria-live="polite">{copy.page} {page} {copy.of} {totalPages} · {listState.data?.total} {copy.records}</span>
                <div className="button-row">
                  <button className="button" type="button" disabled={!session || page <= 1 || listState.status === "loading"} onClick={() => session && void loadListState(session.token, page - 1)}>{copy.previous}</button>
                  <button className="button" type="button" disabled={!session || page >= totalPages || listState.status === "loading"} onClick={() => session && void loadListState(session.token, page + 1)}>{copy.next}</button>
                </div>
              </nav> : null}
            </>
          ) : listState.status !== "error" ? (
            <EmptyState>{listState.status === "loading" ? copy.loadingList : session ? copy.emptyList : copy.signInHelp}</EmptyState>
          ) : null}
        </Panel>
        </> : null}
      </div>
      <SignInDialog open={signInOpen} onClose={() => setSignInOpen(false)} />
    </SiteShell>
  );
}
