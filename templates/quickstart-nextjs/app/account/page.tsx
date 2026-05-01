"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { SiteShell } from "@/components/site-shell";
import { EmptyState, Notice, Panel, DetailRows } from "@/components/ui";
import {
  ApiError,
  issueWSTicket,
  logout,
  refreshSession
} from "@/lib/api";
import {
  readSession,
  SESSION_EVENT,
  writeSession
} from "@/lib/auth";
import { formatDate, maskToken } from "@/lib/format";

function messageFromError(error: unknown): string {
  if (error instanceof ApiError) {
    return error.message;
  }
  if (error instanceof Error) {
    return error.message;
  }
  return "unexpected error";
}

export default function AccountPage() {
  const router = useRouter();
  const [status, setStatus] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState<"" | "refresh" | "logout" | "ticket">("");
  const [ticket, setTicket] = useState<{ value: string; expiresAt: string } | null>(null);
  const [session, setSession] = useState<ReturnType<typeof readSession>>(null);

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

  async function handleRefresh() {
    if (!session) {
      setError("sign in first");
      return;
    }
    setBusy("refresh");
    setError("");
    setStatus("");

    try {
      const nextSession = await refreshSession(session.token);
      writeSession(nextSession);
      setSession(nextSession);
      setStatus("Session refreshed.");
    } catch (err) {
      setError(messageFromError(err));
    } finally {
      setBusy("");
    }
  }

  async function handleLogout() {
    if (!session) {
      writeSession(null);
      router.push("/login");
      return;
    }
    setBusy("logout");
    setError("");
    setStatus("");

    try {
      await logout(session.token);
      writeSession(null);
      setTicket(null);
      setStatus("Session cleared locally.");
      router.push("/login");
    } catch (err) {
      setError(messageFromError(err));
    } finally {
      setBusy("");
    }
  }

  async function handleIssueTicket() {
    if (!session) {
      setError("sign in first");
      return;
    }
    setBusy("ticket");
    setError("");
    setStatus("");

    try {
      const res = await issueWSTicket(session.token);
      setTicket({ value: res.ticket, expiresAt: res.expires_at });
      setStatus("WS ticket issued.");
    } catch (err) {
      setError(messageFromError(err));
    } finally {
      setBusy("");
    }
  }

  return (
    <SiteShell
      eyebrow="Account Settings"
      title="Use the account page as a real settings surface, not just a debug screen."
      description="This page still validates JWT refresh, logout, and ticket issuance, but it is framed as the starting point for an actual settings area. In a real product, profile preferences, security controls, and workspace settings would grow from here."
      sideTitle="Account contract"
      sideBody={
        <DetailRows
          rows={[
            { label: "Refresh", value: <span className="inline-code">POST /auth/refresh</span> },
            { label: "Logout", value: <span className="inline-code">POST /auth/logout</span> },
            { label: "WS ticket", value: <span className="inline-code">POST /websocket/ticket</span> }
          ]}
        />
      }
      toc={[
        { id: "session", label: "Current session" },
        { id: "ticket", label: "WebSocket ticket" }
      ]}
    >
      <div className="page-grid">
        <Panel className="span-7" title="Current session" subtitle="Loaded from localStorage.">
          <div id="session" />
          {session ? (
            <div className="details-list">
              <div className="details-row">
                <strong>User ID</strong>
                <span className="inline-code">{session.user.id}</span>
              </div>
              <div className="details-row">
                <strong>Email</strong>
                <span>{session.user.email}</span>
              </div>
              <div className="details-row">
                <strong>Role</strong>
                <span>{session.user.role || "user"}</span>
              </div>
              <div className="details-row">
                <strong>Username</strong>
                <span>{session.user.username || "-"}</span>
              </div>
              <div className="details-row">
                <strong>Token</strong>
                <span className="inline-code">{maskToken(session.token)}</span>
              </div>
              <div className="details-row">
                <strong>Expires</strong>
                <span>{formatDate(session.expires_at)}</span>
              </div>
            </div>
          ) : (
            <EmptyState>No session found. Open `/login` first.</EmptyState>
          )}

          <div className="button-row">
            <button className="button primary" disabled={busy !== ""} onClick={handleRefresh}>
              {busy === "refresh" ? "Refreshing..." : "Refresh Session"}
            </button>
            <button className="button" disabled={busy !== ""} onClick={handleIssueTicket}>
              {busy === "ticket" ? "Issuing..." : "Issue WS Ticket"}
            </button>
            <button className="button danger" disabled={busy !== ""} onClick={handleLogout}>
              {busy === "logout" ? "Logging out..." : "Logout"}
            </button>
          </div>

          {status ? <Notice tone="success">{status}</Notice> : null}
          {error ? <Notice tone="error">{error}</Notice> : null}
        </Panel>

        <Panel className="span-5" title="WebSocket ticket" subtitle="Useful when your host app has privileged browser->WS entry points.">
          <div id="ticket" />
          {ticket ? (
            <div className="details-list">
              <div className="details-row">
                <strong>Ticket</strong>
                <span className="inline-code">{ticket.value}</span>
              </div>
              <div className="details-row">
                <strong>Expires</strong>
                <span>{formatDate(ticket.expiresAt)}</span>
              </div>
            </div>
          ) : (
            <EmptyState>No ticket issued yet.</EmptyState>
          )}
          <p className="footer-note">
            The auth module only issues the ticket. How your host verifies scope and turns that into product behavior remains host-specific.
          </p>
        </Panel>
      </div>
    </SiteShell>
  );
}
