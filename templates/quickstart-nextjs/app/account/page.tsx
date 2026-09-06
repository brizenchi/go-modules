"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { ResourceFailure, SignInRequired } from "@/components/resource-feedback";
import { SiteShell } from "@/components/site-shell";
import { DetailRows, EmptyState, LabelPill, Notice, Panel } from "@/components/ui";
import {
  getAccountProfile,
  getCapabilities,
  issueWSTicket,
  logout,
  refreshSession,
  updateAccountProfile,
  type AccountProfile,
  type CapabilitiesView,
  type UpdateAccountProfilePayload
} from "@/lib/api";
import { clearSessionIfToken, readSession, SESSION_EVENT, writeSession } from "@/lib/auth";
import { formatDate } from "@/lib/format";
import { useI18n } from "@/lib/i18n";
import {
  describeRequestFailure,
  idleResource,
  loadingResource,
  readyResource,
  settleResource,
  type RequestFailure,
  type ResourceState
} from "@/lib/request-state";
import {
  beginRequestGeneration,
  invalidateRequestGeneration,
  isCurrentRequestGeneration
} from "@/lib/request-generation";

function validateProfile(username: string, avatarURL: string): RequestFailure | null {
  if (username.length > 100) {
    return { kind: "configuration", title: "Username is too long", message: "Use 100 characters or fewer.", retryable: false };
  }
  if (avatarURL.length > 512) {
    return { kind: "configuration", title: "Avatar URL is too long", message: "Use 512 characters or fewer.", retryable: false };
  }
  if (avatarURL) {
    try {
      const parsed = new URL(avatarURL);
      if (parsed.protocol !== "http:" && parsed.protocol !== "https:") throw new Error("invalid protocol");
    } catch {
      return { kind: "configuration", title: "Avatar URL is invalid", message: "Enter an absolute http(s) URL or leave it empty.", retryable: false };
    }
  }
  return null;
}

export default function AccountPage() {
  const router = useRouter();
  const { locale, t } = useI18n();
  const dateLocale = locale === "zh" ? "zh-CN" : "en-US";
  const [session, setSession] = useState<ReturnType<typeof readSession>>(null);
  const [sessionReady, setSessionReady] = useState(false);
  const [capabilitiesState, setCapabilitiesState] = useState<ResourceState<CapabilitiesView>>(idleResource());
  const [profileState, setProfileState] = useState<ResourceState<AccountProfile>>(idleResource());
  const [username, setUsername] = useState("");
  const [avatarURL, setAvatarURL] = useState("");
  const [ticket, setTicket] = useState<{ value: string; expiresAt: string } | null>(null);
  const [busy, setBusy] = useState<"" | "profile" | "refresh" | "logout" | "ticket">("");
  const [status, setStatus] = useState("");
  const [actionFailure, setActionFailure] = useState<RequestFailure | null>(null);
  const capabilitiesGenerationRef = useRef(0);
  const capabilitiesMountedRef = useRef(false);
  const profileGenerationRef = useRef(0);
  const actionGenerationRef = useRef(0);
  const profile = profileState.data;
  const accountEnabled = capabilitiesState.status === "ready" && capabilitiesState.data.account.enabled;
  const profileDirty = Boolean(profile)
    && (username.trim() !== profile?.username || avatarURL.trim() !== profile?.avatar_url);

  const loadProfile = useCallback(async (token: string) => {
    const generation = beginRequestGeneration(profileGenerationRef);
    setProfileState(loadingResource());
    const nextState = await settleResource(getAccountProfile(token), "Account profile");
    if (
      readSession()?.token !== token
      || !isCurrentRequestGeneration(profileGenerationRef, generation)
    ) return;
    setProfileState(nextState);
    if (nextState.status === "ready") {
      setUsername(nextState.data.username || "");
      setAvatarURL(nextState.data.avatar_url || "");
    }
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
    capabilitiesMountedRef.current = true;
    void loadCapabilities();
    return () => {
      capabilitiesMountedRef.current = false;
      invalidateRequestGeneration(capabilitiesGenerationRef);
    };
  }, []);

  useEffect(() => {
    invalidateRequestGeneration(actionGenerationRef);
    setTicket(null);
    setBusy("");
    setStatus("");
    setActionFailure(null);
    if (!session?.token || !accountEnabled) {
      invalidateRequestGeneration(profileGenerationRef);
      setProfileState(idleResource());
      setUsername("");
      setAvatarURL("");
      return;
    }
    void loadProfile(session.token);
    return () => {
      invalidateRequestGeneration(profileGenerationRef);
      invalidateRequestGeneration(actionGenerationRef);
    };
  }, [session?.token, accountEnabled, loadProfile]);

  async function loadCapabilities() {
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

  async function handleSaveProfile() {
    if (!session || !profile) return;
    const requestToken = session.token;
    const nextUsername = username.trim();
    const nextAvatarURL = avatarURL.trim();
    const validationFailure = validateProfile(nextUsername, nextAvatarURL);
    if (validationFailure) {
      setActionFailure(validationFailure);
      return;
    }
    const payload: UpdateAccountProfilePayload = {};
    if (nextUsername !== profile.username) payload.username = nextUsername;
    if (nextAvatarURL !== profile.avatar_url) payload.avatar_url = nextAvatarURL;
    if (Object.keys(payload).length === 0) {
      setStatus(t({ en: "No profile changes to save.", zh: "没有需要保存的资料变更。" }));
      return;
    }
    const profileGeneration = beginRequestGeneration(profileGenerationRef);
    const actionGeneration = beginRequestGeneration(actionGenerationRef);
    setBusy("profile");
    setStatus("");
    setActionFailure(null);
    try {
      const updated = await updateAccountProfile(requestToken, payload);
      const currentSession = readSession();
      if (
        !currentSession
        || currentSession.token !== requestToken
        || !isCurrentRequestGeneration(profileGenerationRef, profileGeneration)
        || !isCurrentRequestGeneration(actionGenerationRef, actionGeneration)
      ) return;
      setProfileState(readyResource(updated));
      setUsername(updated.username || "");
      setAvatarURL(updated.avatar_url || "");
      writeSession({
        ...currentSession,
        user: { ...currentSession.user, username: updated.username, avatar: updated.avatar_url }
      });
      setStatus(t({ en: "Profile saved.", zh: "账户资料已保存。" }));
    } catch (error) {
      if (
        readSession()?.token === requestToken
        && isCurrentRequestGeneration(profileGenerationRef, profileGeneration)
        && isCurrentRequestGeneration(actionGenerationRef, actionGeneration)
      ) {
        setActionFailure(describeRequestFailure(error, "Account profile"));
      }
    } finally {
      if (
        readSession()?.token === requestToken
        && isCurrentRequestGeneration(actionGenerationRef, actionGeneration)
      ) {
        setBusy("");
      }
    }
  }

  async function handleRefresh() {
    if (!session) return;
    const requestToken = session.token;
    const actionGeneration = beginRequestGeneration(actionGenerationRef);
    setBusy("refresh");
    setStatus("");
    setActionFailure(null);
    try {
      const refreshed = await refreshSession(requestToken);
      const currentSession = readSession();
      if (
        !currentSession
        || currentSession.token !== requestToken
        || !isCurrentRequestGeneration(actionGenerationRef, actionGeneration)
      ) return;
      writeSession(refreshed);
      setStatus(t({ en: "Session refreshed.", zh: "当前登录会话已刷新。" }));
    } catch (error) {
      if (
        readSession()?.token === requestToken
        && isCurrentRequestGeneration(actionGenerationRef, actionGeneration)
      ) {
        setActionFailure(describeRequestFailure(error, "Session refresh"));
      }
    } finally {
      if (
        readSession()?.token === requestToken
        && isCurrentRequestGeneration(actionGenerationRef, actionGeneration)
      ) {
        setBusy("");
      }
    }
  }

  async function handleLogout() {
    const requestToken = session?.token || "";
    const actionGeneration = beginRequestGeneration(actionGenerationRef);
    setBusy("logout");
    try {
      if (requestToken) await logout(requestToken);
    } catch {
      // Local sign-out must still succeed if the backend or token is unavailable.
    } finally {
      if (requestToken) {
        clearSessionIfToken(requestToken);
      } else {
        writeSession(null);
      }
      if (
        !readSession()
        && isCurrentRequestGeneration(actionGenerationRef, actionGeneration)
      ) {
        setTicket(null);
        setBusy("");
        router.push("/login");
      }
    }
  }

  async function handleIssueTicket() {
    if (!session) return;
    const requestToken = session.token;
    const actionGeneration = beginRequestGeneration(actionGenerationRef);
    setBusy("ticket");
    setTicket(null);
    setStatus("");
    setActionFailure(null);
    try {
      const result = await issueWSTicket(requestToken);
      if (
        readSession()?.token !== requestToken
        || !isCurrentRequestGeneration(actionGenerationRef, actionGeneration)
      ) return;
      setTicket({ value: result.ticket, expiresAt: result.expires_at });
      setStatus(t({ en: "WebSocket ticket issued.", zh: "临时连接凭证已生成。" }));
    } catch (error) {
      if (
        readSession()?.token === requestToken
        && isCurrentRequestGeneration(actionGenerationRef, actionGeneration)
      ) {
        setActionFailure(describeRequestFailure(error, "WebSocket ticket"));
      }
    } finally {
      if (
        readSession()?.token === requestToken
        && isCurrentRequestGeneration(actionGenerationRef, actionGeneration)
      ) {
        setBusy("");
      }
    }
  }

  const capabilityFailure: RequestFailure | null = capabilitiesState.status === "error"
    ? capabilitiesState.failure
    : capabilitiesState.status === "ready" && !capabilitiesState.data.account.enabled
      ? {
          kind: "disabled",
          title: t({ en: "Account settings are not enabled", zh: "账户设置暂未启用" }),
          message: t({ en: "Ask the site administrator to enable account profiles.", zh: "请联系网站管理员启用账户资料功能。" }),
          retryable: false
        }
      : null;

  return (
    <SiteShell
      eyebrow={t({ en: "Account", zh: "账户设置" })}
      title={t({ en: "Your profile, security, and access.", zh: "管理你的资料、安全与访问。" })}
      description={t({ en: "Keep your identity current and manage the session on this device.", zh: "更新对外展示信息，并管理当前设备上的登录状态。" })}
      accountMenuData={{ capabilities: capabilitiesState }}
      sideTitle={t({ en: "Account status", zh: "账户状态" })}
      sideBody={<DetailRows rows={[
        { label: t({ en: "Email", zh: "邮箱" }), value: <span>{t({ en: "Protected by your sign-in provider", zh: "由登录服务安全保护" })}</span> },
        { label: t({ en: "Session", zh: "登录状态" }), value: <span>{t({ en: "Active on this device", zh: "当前设备已登录" })}</span> },
        { label: t({ en: "Profile", zh: "个人资料" }), value: <span>{t({ en: "Private until you choose to share it", zh: "默认仅自己可见" })}</span> }
      ]} />}
      toc={[
        { id: "profile", label: t({ en: "Profile", zh: "个人资料" }) },
        { id: "security", label: t({ en: "Security", zh: "会话安全" }) },
        { id: "developer-access", label: t({ en: "Advanced access", zh: "高级访问" }) }
      ]}
    >
      <div className="page-grid">
        {sessionReady && !session ? <div className="span-12"><SignInRequired message={t({ en: "Sign in to view and update your account.", zh: "登录后即可查看并更新账户设置。" })} /></div> : null}
        {capabilityFailure ? <div className="span-12"><ResourceFailure failure={capabilityFailure} onRetry={capabilitiesState.status === "error" ? () => void loadCapabilities() : undefined} /></div> : null}

        <Panel className="span-7" title={t({ en: "Profile", zh: "个人资料" })} subtitle={t({ en: "Choose how your name and avatar appear across the product.", zh: "设置你在产品中显示的名称和头像。" })}>
          <div id="profile" />
          {profile ? (
            <div className="input-row">
              <div className="field">
                <label htmlFor="account-email">{t({ en: "Email", zh: "邮箱" })}</label>
                <input id="account-email" value={profile.email} readOnly aria-readonly="true" />
              </div>
              <div className="field">
                <label htmlFor="account-username">{t({ en: "Display name", zh: "显示名称" })}</label>
                <input id="account-username" maxLength={100} value={username} onChange={(event) => setUsername(event.target.value)} />
              </div>
              <div className="field">
                <label htmlFor="account-avatar">{t({ en: "Avatar URL", zh: "头像链接" })}</label>
                <input id="account-avatar" type="url" maxLength={512} placeholder="https://example.com/avatar.png" value={avatarURL} onChange={(event) => setAvatarURL(event.target.value)} />
              </div>
              <div className="button-row">
                <button className="button primary" type="button" disabled={!profileDirty || busy !== ""} onClick={() => void handleSaveProfile()}>{busy === "profile" ? t({ en: "Saving...", zh: "保存中..." }) : t({ en: "Save changes", zh: "保存修改" })}</button>
                <button className="button" type="button" disabled={!profileDirty || busy !== ""} onClick={() => { setUsername(profile.username); setAvatarURL(profile.avatar_url); }}>{t({ en: "Reset", zh: "重置" })}</button>
              </div>
            </div>
          ) : profileState.status === "error" ? (
            <ResourceFailure failure={profileState.failure} onRetry={session ? () => void loadProfile(session.token) : undefined} />
          ) : (
            <EmptyState>{profileState.status === "loading" ? t({ en: "Loading profile...", zh: "正在加载账户资料..." }) : t({ en: "Sign in to load your profile.", zh: "登录后即可加载账户资料。" })}</EmptyState>
          )}
          {status ? <Notice tone="success">{status}</Notice> : null}
          {actionFailure ? <ResourceFailure failure={actionFailure} /> : null}
        </Panel>

        <Panel className="span-5" title={t({ en: "Account details", zh: "账户详情" })} subtitle={t({ en: "Your identity, role, balance, and account history.", zh: "查看身份、角色、积分余额和账户记录。" })}>
          {profile ? <DetailRows rows={[
            { label: t({ en: "User ID", zh: "用户 ID" }), value: <span className="inline-code">{profile.id}</span> },
            { label: t({ en: "Email status", zh: "邮箱状态" }), value: <LabelPill>{profile.email_verified ? t({ en: "Verified", zh: "已验证" }) : t({ en: "Unverified", zh: "未验证" })}</LabelPill> },
            { label: t({ en: "Role", zh: "账户角色" }), value: <span>{profile.role || "user"}</span> },
            { label: t({ en: "Credits", zh: "可用积分" }), value: <span>{profile.credits}</span> },
            { label: t({ en: "Created", zh: "创建时间" }), value: <span>{formatDate(profile.created_at, dateLocale)}</span> },
            { label: t({ en: "Updated", zh: "更新时间" }), value: <span>{formatDate(profile.updated_at, dateLocale)}</span> }
          ]} /> : <EmptyState>{t({ en: "Account details appear after the profile loads.", zh: "账户资料加载后将在这里显示详情。" })}</EmptyState>}
        </Panel>

        <Panel className="span-7" title={t({ en: "Session & security", zh: "会话与安全" })} subtitle={t({ en: "Review, refresh, or end this device session.", zh: "查看、刷新或结束当前设备的登录会话。" })}>
          <div id="security" />
          {session ? <DetailRows rows={[
            { label: t({ en: "Signed in as", zh: "当前账户" }), value: <span>{session.user.email}</span> },
            { label: t({ en: "Session credential", zh: "会话凭证" }), value: <span>{t({ en: "Stored securely on this device", zh: "已安全保存在当前设备" })}</span> },
            { label: t({ en: "Expires", zh: "有效期至" }), value: <span>{formatDate(session.expires_at, dateLocale)}</span> }
          ]} /> : <EmptyState>{t({ en: "No active session.", zh: "当前没有有效登录会话。" })}</EmptyState>}
          <div className="button-row">
            <button className="button" type="button" disabled={!session || busy !== ""} onClick={() => void handleRefresh()}>{busy === "refresh" ? t({ en: "Refreshing...", zh: "刷新中..." }) : t({ en: "Refresh session", zh: "刷新会话" })}</button>
            <button className="button danger" type="button" disabled={busy !== ""} onClick={() => void handleLogout()}>{busy === "logout" ? t({ en: "Signing out...", zh: "正在退出..." }) : t({ en: "Sign out", zh: "退出登录" })}</button>
          </div>
        </Panel>

        <Panel className="span-5" title={t({ en: "Advanced access", zh: "高级访问" })} subtitle={t({ en: "Create a short-lived credential for an approved live connection.", zh: "为已授权的实时连接生成短时有效凭证。" })}>
          <div id="developer-access" />
          {ticket ? <DetailRows rows={[
            { label: t({ en: "Ticket", zh: "临时凭证" }), value: <span className="inline-code">{ticket.value}</span> },
            { label: t({ en: "Expires", zh: "有效期至" }), value: <span>{formatDate(ticket.expiresAt, dateLocale)}</span> }
          ]} /> : <EmptyState>{t({ en: "No temporary credential issued.", zh: "尚未生成临时连接凭证。" })}</EmptyState>}
          <div className="button-row">
            <button className="button" type="button" disabled={!session || busy !== ""} onClick={() => void handleIssueTicket()}>{busy === "ticket" ? t({ en: "Issuing...", zh: "生成中..." }) : t({ en: "Create credential", zh: "生成临时凭证" })}</button>
          </div>
        </Panel>
      </div>
    </SiteShell>
  );
}
