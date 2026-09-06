"use client";

import { useCallback, useEffect, useRef, useState, type ReactNode } from "react";
import { ApiError } from "@/lib/api";
import { readSession, SESSION_EVENT, type AuthSession } from "@/lib/auth";
import { useI18n } from "@/lib/i18n";
import { SignInDialog } from "./sign-in-dialog";
import type { PageResult } from "@/lib/operations-api";
import styles from "./console-kit.module.css";

export function useConsoleSession() {
  const [session, setSession] = useState<AuthSession | null>(null);
  const [ready, setReady] = useState(false);
  useEffect(() => {
    const sync = () => { setSession(readSession()); setReady(true); };
    sync(); window.addEventListener(SESSION_EVENT, sync); window.addEventListener("storage", sync);
    return () => { window.removeEventListener(SESSION_EVENT, sync); window.removeEventListener("storage", sync); };
  }, []);
  return { session, ready };
}

export function useConsoleResource<T>(token: string, loader: (token: string) => Promise<T>) {
  const [result, setResult] = useState<{ owner: string; data: T | null; loading: boolean; error: unknown }>({ owner: "", data: null, loading: false, error: null });
  const generation = useRef(0);
  const refresh = useCallback(async () => {
    if (!token || readSession()?.token !== token) return;
    const current = ++generation.current;
    setResult((previous) => ({ owner: token, data: previous.owner === token ? previous.data : null, loading: true, error: null }));
    try {
      const data = await loader(token);
      if (generation.current === current && readSession()?.token === token) setResult({ owner: token, data, loading: false, error: null });
    } catch (error) {
      if (generation.current === current && readSession()?.token === token) setResult({ owner: token, data: null, loading: false, error });
    }
  }, [loader, token]);
  useEffect(() => { const counter = generation; void refresh(); return () => { counter.current++; }; }, [refresh]);
  return { ...(result.owner === token ? result : { data: null, error: null, loading: Boolean(token) }), refresh };
}

export function useConsoleAction(token: string) {
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");
  const [error, setError] = useState<unknown>(null);
  const generation = useRef(0);
  const running = useRef(false);
  useEffect(() => { const counter = generation; counter.current++; running.current = false; setBusy(false); setMessage(""); setError(null); return () => { counter.current++; }; }, [token]);
  async function run<T>(request: () => Promise<T>, onSuccess: (data: T) => void) {
    if (running.current || !token || readSession()?.token !== token) return;
    const current = ++generation.current;
    running.current = true; setBusy(true); setError(null); setMessage("");
    try {
      const data = await request();
      if (generation.current === current && readSession()?.token === token) onSuccess(data);
    } catch (failure) {
      if (generation.current === current && readSession()?.token === token) setError(failure);
    } finally {
      if (generation.current === current) { running.current = false; setBusy(false); }
    }
  }
  return { busy, message, error, setMessage, run };
}

export function ConsoleGate({ session, ready, admin = false, children }: { session: AuthSession | null; ready: boolean; admin?: boolean; children: ReactNode }) {
  const { t } = useI18n();
  const [open, setOpen] = useState(false);
  if (!ready) return <p role="status">{t({ en: "Loading your account…", zh: "正在读取账号…" })}</p>;
  if (!session) return <div className={styles.empty}>
    <h2>{t({ en: "Sign in to continue", zh: "登录后继续" })}</h2>
    <p>{t({ en: "Your files, credits and activity belong to your account.", zh: "登录后查看属于你的文件、积分和操作记录。" })}</p>
    <button className="button primary" type="button" onClick={() => setOpen(true)}>{t({ en: "Sign in", zh: "登录" })}</button>
    <SignInDialog open={open} onClose={() => setOpen(false)} />
  </div>;
  if (admin && session.user.role !== "admin") return <div className={styles.empty} role="alert"><h2>{t({ en: "Administrator access required", zh: "需要管理员权限" })}</h2><p>{t({ en: "This account cannot access operator controls.", zh: "当前账号无权访问运营管理功能。" })}</p></div>;
  return <>{children}</>;
}

export function ConsoleError({ error, retry }: { error: unknown; retry?: () => void }) {
  const { t } = useI18n();
  if (!error) return null;
  const api = error instanceof ApiError ? error : null;
  const message = api?.status === 403 ? t({ en: "You don't have permission for this action.", zh: "你没有执行此操作的权限。" })
    : api?.status === 401 ? t({ en: "Please sign in again.", zh: "请重新登录。" })
    : api?.status === 409 && api.message === "price_changed" ? t({ en: "The export price changed. Refresh the price, then confirm again. No credits were charged.", zh: "导出价格已变更，请刷新价格后重新确认。本次没有扣除积分。" })
    : api?.status === 409 && api.message === "insufficient_credits" ? t({ en: "There are not enough credits for this export. Check your balance or get more credits.", zh: "当前积分不足，请查看余额或购买积分后重试。" })
    : api?.status === 409 ? t({ en: "This request conflicts with an earlier action. Refresh the page and check the latest record.", zh: "本次请求与已有操作冲突，请刷新页面并查看最新记录。" })
    : api?.status === 400 || api?.status === 422 ? t({ en: "Check the entered values and try again.", zh: "请检查填写内容后重试。" })
    : api?.status === 503 ? t({ en: "This feature is not ready yet. Please contact support.", zh: "此功能暂未就绪，请联系支持人员。" })
    : t({ en: "Unable to complete the request. Please try again.", zh: "请求未能完成，请重试。" });
  return <div className={styles.error} role="alert"><span>{message}</span>{retry ? <button className="button" type="button" onClick={retry}>{t({ en: "Try again", zh: "重试" })}</button> : null}</div>;
}

export function Pagination({ result, loading, onPage }: { result: Pick<PageResult<unknown>, "page" | "limit" | "total">; loading: boolean; onPage: (page: number) => void }) {
  const { t } = useI18n();
  const pages = Math.max(1, Math.ceil(result.total / result.limit));
  return <nav className={styles.pagination} aria-label={t({ en: "Pagination", zh: "分页" })}>
    <span aria-live="polite">{t({ en: `${result.total} records · Page ${result.page} of ${pages}`, zh: `共 ${result.total} 条 · 第 ${result.page} / ${pages} 页` })}</span>
    <div className="button-row"><button className="button" type="button" disabled={loading || result.page <= 1} onClick={() => onPage(result.page - 1)}>{t({ en: "Previous", zh: "上一页" })}</button><button className="button" type="button" disabled={loading || result.page >= pages} onClick={() => onPage(result.page + 1)}>{t({ en: "Next", zh: "下一页" })}</button></div>
  </nav>;
}

export function newIntentKey(): string { return crypto.randomUUID(); }
export function downloadText(filename: string, content: string) {
  const url = URL.createObjectURL(new Blob([content], { type: "text/markdown;charset=utf-8" }));
  const link = document.createElement("a"); link.href = url; link.download = filename; link.click();
  setTimeout(() => URL.revokeObjectURL(url), 1000);
}
export { styles as consoleStyles };
