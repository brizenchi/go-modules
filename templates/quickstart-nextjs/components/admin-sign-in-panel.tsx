"use client";

import { useEffect, useRef, useState, type FormEvent } from "react";
import { getCapabilities, loginAdmin } from "@/lib/api";
import { readSession, SESSION_EVENT, SESSION_KEY, writeSession } from "@/lib/auth";
import { adminPasswordEnabled, createAdminPasswordLogin, describeAdminLoginFailure, type AdminLoginFailure } from "@/lib/admin-password-login";
import { useI18n } from "@/lib/i18n";

const failureMessages: Record<AdminLoginFailure, { en: string; zh: string }> = {
  credentials: { en: "The administrator email or password is incorrect.", zh: "管理员邮箱或密码不正确。" },
  limited: { en: "Too many sign-in attempts. Please wait a few minutes before trying again.", zh: "登录尝试过于频繁，请稍等几分钟后重试。" },
  disabled: { en: "Administrator sign-in is not available yet. Contact the website owner.", zh: "管理员登录尚未启用，请联系网站负责人。" },
  unavailable: { en: "Unable to connect to the sign-in service. Please try again.", zh: "暂时无法连接登录服务，请重试。" }
};

export function AdminSignInPanel() {
  const { t } = useI18n();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [availability, setAvailability] = useState<"loading" | "enabled" | "disabled" | "error">("loading");
  const [failure, setFailure] = useState<AdminLoginFailure | null>(null);
  const [busy, setBusy] = useState(false);
  const [retry, setRetry] = useState(0);
  const mounted = useRef(false);
  const attempt = useRef(0);
  const [login] = useState(() => createAdminPasswordLogin({
    authenticate: loginAdmin,
    readSessionToken: () => readSession()?.token || "",
    commit: (session) => { setPassword(""); writeSession(session); }
  }));

  useEffect(() => {
    mounted.current = true;
    const counter = attempt;
    const sessionChanged = (event: Event) => {
      if (event.type === "storage") {
        const key = (event as StorageEvent).key;
        if (key !== null && key !== SESSION_KEY) return;
      }
      counter.current++;
      login.cancel();
      setPassword("");
      setBusy(false);
      setFailure(null);
    };
    window.addEventListener(SESSION_EVENT, sessionChanged);
    window.addEventListener("storage", sessionChanged);
    return () => {
      mounted.current = false;
      counter.current++;
      login.cancel();
      window.removeEventListener(SESSION_EVENT, sessionChanged);
      window.removeEventListener("storage", sessionChanged);
    };
  }, [login]);

  useEffect(() => {
    let current = true;
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), 8000);
    setAvailability("loading");
    void getCapabilities(controller.signal).then((capabilities) => {
      if (current) setAvailability(adminPasswordEnabled(capabilities) ? "enabled" : "disabled");
    }).catch(() => { if (current) setAvailability("error"); }).finally(() => clearTimeout(timer));
    return () => { current = false; clearTimeout(timer); controller.abort(); };
  }, [retry]);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (busy || availability !== "enabled") return;
    const current = ++attempt.current;
    setBusy(true);
    setFailure(null);
    try {
      await login.login(email, password);
    } catch (error) {
      if (mounted.current && attempt.current === current) setFailure(describeAdminLoginFailure(error));
    } finally {
      if (mounted.current && attempt.current === current) { setPassword(""); setBusy(false); }
    }
  }

  if (availability === "loading") return <p role="status">{t({ en: "Loading administrator sign-in…", zh: "正在加载管理员登录…" })}</p>;
  if (availability !== "enabled") return <div className="sign-in-stack compact">
    <div className="notice" role={availability === "error" ? "alert" : "status"}>{t(failureMessages[availability === "error" ? "unavailable" : "disabled"])}</div>
    <button type="button" className="button" onClick={() => setRetry((value) => value + 1)}>{t({ en: "Try again", zh: "重试" })}</button>
  </div>;

  return <form className="sign-in-stack compact" onSubmit={submit} aria-busy={busy}>
    <div className="field">
      <label htmlFor="admin-sign-in-email">{t({ en: "Administrator email", zh: "管理员邮箱" })}</label>
      <input id="admin-sign-in-email" name="email" type="email" autoComplete="username" required maxLength={254} value={email} disabled={busy} onChange={(event) => setEmail(event.target.value)} />
    </div>
    <div className="field">
      <label htmlFor="admin-sign-in-password">{t({ en: "Password", zh: "密码" })}</label>
      <input id="admin-sign-in-password" name="password" type="password" autoComplete="current-password" required maxLength={72} value={password} disabled={busy} onChange={(event) => setPassword(event.target.value)} />
    </div>
    {failure ? <div className="notice error" role="alert">{t(failureMessages[failure])}</div> : null}
    <button type="submit" className="button primary wide" disabled={busy}>{t(busy ? { en: "Signing in…", zh: "登录中…" } : { en: "Sign in to administration", zh: "登录管理后台" })}</button>
  </form>;
}
