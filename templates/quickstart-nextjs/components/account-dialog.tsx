"use client";

import { useEffect, useRef, useState } from "react";
import { ResponsiveDialog } from "@/components/responsive-dialog";
import { InterfaceIcon } from "@/components/interface-icon";
import { logout } from "@/lib/api";
import { readSession, type AuthSession } from "@/lib/auth";
import { accountLogoutFailure } from "@/lib/account-actions";
import { useI18n } from "@/lib/i18n";
import styles from "./account-dialog.module.css";

export function AccountDialog({ open, session, onClose }: {
  open: boolean; session: AuthSession | null; onClose: () => void;
}) {
  const { locale, t } = useI18n();
  const inFlight = useRef(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  useEffect(() => { setError(""); }, [open, session?.token]);

  async function signOut(dismiss: () => void) {
    if (!session || inFlight.current) return;
    const token = session.token;
    inFlight.current = true;
    setBusy(true); setError("");
    try {
      await logout(token);
      if (!readSession() || readSession()?.token === token) dismiss();
    } catch (failure) {
      if (readSession()?.token === token) setError(accountLogoutFailure(failure, locale));
    } finally {
      inFlight.current = false; setBusy(false);
    }
  }

  return <ResponsiveDialog open={open && !!session} onClose={onClose}
    title={t({ en: "Your account", zh: "账户信息" })}
    description={t({ en: "Your sign-in identity for this workspace.", zh: "用于登录和接收账户通知。" })}
    closeLabel={t({ en: "Close", zh: "关闭" })} icon={<InterfaceIcon name="account" />}>
    {(dismiss) => <>
    <dl className={styles.identity}><dt>{t({ en: "Email", zh: "登录邮箱" })}</dt><dd>{session?.user.email}</dd></dl>
    {error ? <p role="alert" className={styles.error}>{error}</p> : null}
    <div className={styles.footer}>
      <button className={styles.signOut} type="button" disabled={busy} onClick={() => void signOut(dismiss)}>
        <InterfaceIcon name="logout" />{busy ? t({ en: "Signing out…", zh: "退出中…" }) : t({ en: "Sign out", zh: "退出登录" })}
      </button>
      <button className="button secondary" type="button" onClick={dismiss}>{t({ en: "Done", zh: "完成" })}</button>
    </div>
    </>}
  </ResponsiveDialog>;
}
