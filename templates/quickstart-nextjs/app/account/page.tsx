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
import { formatDate, maskToken } from "@/lib/format";
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
      setStatus("No profile changes to save.");
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
      setStatus("Profile saved.");
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
      setStatus("Session refreshed.");
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
      setStatus("WebSocket ticket issued.");
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
      ? { kind: "disabled", title: "Account settings are not enabled", message: "Deploy a backend with the account profile capability enabled.", retryable: false }
      : null;

  return (
    <SiteShell
      eyebrow="Account Settings"
      title="Keep your identity and account access up to date."
      description="Update the profile fields supported by the API, review account details, and manage the session on this device."
      accountMenuData={{ capabilities: capabilitiesState }}
      sideTitle="Account security"
      sideBody={<DetailRows rows={[
        { label: "Email", value: <span>Verified by the authentication provider</span> },
        { label: "Session", value: <span>Stored only on this device</span> },
        { label: "Profile", value: <span>Saved through the authenticated API</span> }
      ]} />}
      toc={[
        { id: "profile", label: "Profile" },
        { id: "security", label: "Security" },
        { id: "developer-access", label: "Developer access" }
      ]}
    >
      <div className="page-grid">
        {sessionReady && !session ? <div className="span-12"><SignInRequired message="Sign in to view and update your account." /></div> : null}
        {capabilityFailure ? <div className="span-12"><ResourceFailure failure={capabilityFailure} onRetry={capabilitiesState.status === "error" ? () => void loadCapabilities() : undefined} /></div> : null}

        <Panel className="span-7" title="Profile" subtitle="Username and avatar URL are the editable fields supported by this backend.">
          <div id="profile" />
          {profile ? (
            <div className="input-row">
              <div className="field">
                <label htmlFor="account-email">Email</label>
                <input id="account-email" value={profile.email} readOnly aria-readonly="true" />
              </div>
              <div className="field">
                <label htmlFor="account-username">Username</label>
                <input id="account-username" maxLength={100} value={username} onChange={(event) => setUsername(event.target.value)} />
              </div>
              <div className="field">
                <label htmlFor="account-avatar">Avatar URL</label>
                <input id="account-avatar" type="url" maxLength={512} placeholder="https://example.com/avatar.png" value={avatarURL} onChange={(event) => setAvatarURL(event.target.value)} />
              </div>
              <div className="button-row">
                <button className="button primary" type="button" disabled={!profileDirty || busy !== ""} onClick={() => void handleSaveProfile()}>{busy === "profile" ? "Saving..." : "Save changes"}</button>
                <button className="button" type="button" disabled={!profileDirty || busy !== ""} onClick={() => { setUsername(profile.username); setAvatarURL(profile.avatar_url); }}>Reset</button>
              </div>
            </div>
          ) : profileState.status === "error" ? (
            <ResourceFailure failure={profileState.failure} onRetry={session ? () => void loadProfile(session.token) : undefined} />
          ) : (
            <EmptyState>{profileState.status === "loading" ? "Loading profile..." : "Sign in to load your profile."}</EmptyState>
          )}
          {status ? <Notice tone="success">{status}</Notice> : null}
          {actionFailure ? <ResourceFailure failure={actionFailure} /> : null}
        </Panel>

        <Panel className="span-5" title="Account details" subtitle="Read-only identity and product state.">
          {profile ? <DetailRows rows={[
            { label: "User ID", value: <span className="inline-code">{profile.id}</span> },
            { label: "Email status", value: <LabelPill>{profile.email_verified ? "Verified" : "Unverified"}</LabelPill> },
            { label: "Role", value: <span>{profile.role || "user"}</span> },
            { label: "Credits", value: <span>{profile.credits}</span> },
            { label: "Created", value: <span>{formatDate(profile.created_at)}</span> },
            { label: "Updated", value: <span>{formatDate(profile.updated_at)}</span> }
          ]} /> : <EmptyState>Account details appear after the profile loads.</EmptyState>}
        </Panel>

        <Panel className="span-7" title="Session & security" subtitle="Refresh or end the current session on this device.">
          <div id="security" />
          {session ? <DetailRows rows={[
            { label: "Signed in as", value: <span>{session.user.email}</span> },
            { label: "Access token", value: <span className="inline-code">{maskToken(session.token)}</span> },
            { label: "Expires", value: <span>{formatDate(session.expires_at)}</span> }
          ]} /> : <EmptyState>No active session.</EmptyState>}
          <div className="button-row">
            <button className="button" type="button" disabled={!session || busy !== ""} onClick={() => void handleRefresh()}>{busy === "refresh" ? "Refreshing..." : "Refresh session"}</button>
            <button className="button danger" type="button" disabled={busy !== ""} onClick={() => void handleLogout()}>{busy === "logout" ? "Signing out..." : "Sign out"}</button>
          </div>
        </Panel>

        <Panel className="span-5" title="Developer access" subtitle="Issue a short-lived ticket only when a browser WebSocket needs it.">
          <div id="developer-access" />
          {ticket ? <DetailRows rows={[
            { label: "Ticket", value: <span className="inline-code">{ticket.value}</span> },
            { label: "Expires", value: <span>{formatDate(ticket.expiresAt)}</span> }
          ]} /> : <EmptyState>No WebSocket ticket issued.</EmptyState>}
          <div className="button-row">
            <button className="button" type="button" disabled={!session || busy !== ""} onClick={() => void handleIssueTicket()}>{busy === "ticket" ? "Issuing..." : "Issue ticket"}</button>
          </div>
        </Panel>
      </div>
    </SiteShell>
  );
}
