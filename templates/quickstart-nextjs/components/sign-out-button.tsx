"use client";

import { useEffect, useRef, useState } from "react";
import { logout } from "@/lib/api";
import { readSession, type AuthSession } from "@/lib/auth";
import { accountLogoutFailure } from "@/lib/account-actions";
import { useI18n } from "@/lib/i18n";
import { InterfaceIcon } from "./interface-icon";
import styles from "./account-menu.module.css";

export function SignOutButton({ session, className, onSignedOut }: { session: AuthSession; className?: string; onSignedOut?: () => void }) {
  const { t, locale } = useI18n();
  const inFlight = useRef(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  useEffect(() => { setError(""); }, [session.token]);
  async function signOut() {
    if (inFlight.current) return;
    const token = session.token;
    inFlight.current = true;
    setBusy(true); setError("");
    try {
      await logout(token);
      const current = readSession();
      if (!current || current.token === token) onSignedOut?.();
    } catch (failure) {
      if (readSession()?.token === token) setError(accountLogoutFailure(failure, locale));
    } finally {
      inFlight.current = false; setBusy(false);
    }
  }
  return <>
    <button data-account-item className={className || styles.item} type="button" disabled={busy} aria-busy={busy} onClick={() => void signOut()}>
      <InterfaceIcon name="logout" />{busy ? t({ en: "Signing out…", zh: "退出中…" }) : t({ en: "Sign out", zh: "退出登录" })}
    </button>
    {error ? <p className={styles.error} role="alert">{error}</p> : null}
  </>;
}
