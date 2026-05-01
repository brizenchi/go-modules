"use client";

import { useEffect, useState } from "react";
import { SiteShell } from "@/components/site-shell";
import { EmptyState, Notice, Panel, DetailRows } from "@/components/ui";
import {
  ApiError,
  getReferralCode,
  getReferralStats,
  listReferrals,
  type ReferralItem,
  type ReferralStats
} from "@/lib/api";
import { readSession, SESSION_EVENT } from "@/lib/auth";
import { appEnv } from "@/lib/env";
import { formatDate } from "@/lib/format";

function messageFromError(error: unknown): string {
  if (error instanceof ApiError) {
    return error.message;
  }
  if (error instanceof Error) {
    return error.message;
  }
  return "unexpected error";
}

export default function ReferralsPage() {
  const [session, setSession] = useState<ReturnType<typeof readSession>>(null);
  const [code, setCode] = useState<{ code: string; link: string } | null>(null);
  const [stats, setStats] = useState<ReferralStats | null>(null);
  const [items, setItems] = useState<ReferralItem[]>([]);
  const [busy, setBusy] = useState(false);
  const [status, setStatus] = useState("");
  const [error, setError] = useState("");

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
    if (!session) {
      return;
    }
    void load(session.token);
  }, [session]);

  async function load(token: string) {
    setBusy(true);
    setError("");

    try {
      const [codeData, statsData, listData] = await Promise.all([
        getReferralCode(token),
        getReferralStats(token),
        listReferrals(token)
      ]);
      setCode(codeData);
      setStats(statsData);
      setItems(listData.items);
    } catch (err) {
      setError(messageFromError(err));
    } finally {
      setBusy(false);
    }
  }

  async function copyLink() {
    if (!code?.link) {
      return;
    }
    await navigator.clipboard.writeText(code.link);
    setStatus("Referral link copied.");
  }

  return (
    <SiteShell
      eyebrow="Referral Center"
      title="Turn the reusable referral module into a user-facing growth console."
      description="This page gives the product shell a real referral center: a shareable invite link, performance stats, and history. In the matching quickstart backend, attribution and reward activation are already wired end-to-end."
      accountMenuData={{ referralStats: stats }}
      sideTitle="Host contract"
      sideBody={
        <DetailRows
          rows={[
            { label: "Referral base link", value: <span className="inline-code">{`${appEnv.appUrl}/invite?ref=`}</span> },
            { label: "Backend read routes", value: <span className="inline-code">/referral/code • /referral/stats • /referral/list</span> },
            { label: "Signup attribution", value: <span>Already wired by the quickstart templates when login completes.</span> }
          ]}
        />
      }
      toc={[
        { id: "my-link", label: "My referral link" },
        { id: "stats", label: "Stats" },
        { id: "history", label: "History" }
      ]}
    >
      <div className="page-grid">
        <Panel className="span-5" title="My referral link" subtitle="Loaded from GET /referral/code.">
          <div id="my-link" />
          {code ? (
            <div className="details-list">
              <div className="details-row">
                <strong>Code</strong>
                <span className="inline-code">{code.code}</span>
              </div>
              <div className="details-row">
                <strong>Share link</strong>
                <span className="inline-code">{code.link || "-"}</span>
              </div>
            </div>
          ) : (
            <EmptyState>{session ? "No referral code loaded yet." : "Sign in to load your referral data."}</EmptyState>
          )}
          <div className="button-row">
            <button className="button primary" disabled={!code?.link} onClick={() => void copyLink()}>
              Copy Link
            </button>
          </div>
        </Panel>

        <Panel className="span-7" title="Referral stats" subtitle="Loaded from GET /referral/stats.">
          <div id="stats" />
          {stats ? (
            <div className="stats-grid">
              <div className="stat-card">
                <span className="stat-label">Total referred</span>
                <span className="stat-value">{stats.total_referred}</span>
              </div>
              <div className="stat-card">
                <span className="stat-label">Activated</span>
                <span className="stat-value">{stats.activated}</span>
              </div>
              <div className="stat-card">
                <span className="stat-label">Pending</span>
                <span className="stat-value">{stats.pending}</span>
              </div>
              <div className="stat-card">
                <span className="stat-label">Reward credits</span>
                <span className="stat-value">{stats.total_reward_credits}</span>
              </div>
            </div>
          ) : (
            <EmptyState>{busy ? "Loading referral stats..." : "No stats loaded yet."}</EmptyState>
          )}
        </Panel>

        <Panel className="span-12" title="Referral history" subtitle="Loaded from GET /referral/list.">
          <div id="history" />
          {items.length > 0 ? (
            <table className="table">
              <thead>
                <tr>
                  <th>Code</th>
                  <th>Referee</th>
                  <th>Status</th>
                  <th>Reward</th>
                  <th>Created</th>
                  <th>Activated</th>
                </tr>
              </thead>
              <tbody>
                {items.map((item) => (
                  <tr key={item.id}>
                    <td>{item.code}</td>
                    <td>{item.referee_id}</td>
                    <td>{item.status}</td>
                    <td>{item.reward_credits}</td>
                    <td>{formatDate(item.created_at)}</td>
                    <td>{formatDate(item.activated_at)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          ) : (
            <EmptyState>{busy ? "Loading referrals..." : "No referrals yet."}</EmptyState>
          )}
        </Panel>

        <Panel className="span-12" title="Integration boundary" subtitle="What stays reusable, and what consumers still replace.">
          <p>
            `go-modules/modules/referral` owns referral schema, code generation, read APIs, and activation events. The quickstart templates add the remaining host glue needed for a runnable product reference: browser capture, signup-time attribution, and reward activation after Stripe subscription activation. Consumers still replace reward policy, product copy, and any custom user model, but they do not need to invent the core referral flow.
          </p>
        </Panel>

        {status ? <div className="span-12"><Notice tone="success">{status}</Notice></div> : null}
        {error ? <div className="span-12"><Notice tone="error">{error}</Notice></div> : null}
      </div>
    </SiteShell>
  );
}
