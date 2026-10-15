"use client";

import { useEffect, useId, useRef, useState } from "react";
import { usePathname } from "next/navigation";
import { LocalizedLink as Link } from "@/components/localized-link";
import { AccountDialog } from "@/components/account-dialog";
import { SignInDialog } from "@/components/sign-in-dialog";
import { InterfaceIcon } from "@/components/interface-icon";
import { userLabel } from "@/lib/api";
import type { AuthSession } from "@/lib/auth";
import { useI18n } from "@/lib/i18n";
import { disclosureFocusIndex } from "@/lib/disclosure-focus";
import accountStyles from "./account-menu.module.css";

export function AccountMenu({ session, onSignInSuccess }: {
  session: AuthSession | null;
  onSignInSuccess?: (session: AuthSession) => void;
}) {
  const { t } = useI18n();
  const pathname = usePathname();
  const [open, setOpen] = useState(false);
  const [accountOpen, setAccountOpen] = useState(false);
  const [signInOpen, setSignInOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);
  const triggerRef = useRef<HTMLButtonElement>(null);
  const popoverId = useId();
  useEffect(() => {
    if (!open) return;
    const outside = (event: PointerEvent) => {
      if (event.target instanceof Node && !menuRef.current?.contains(event.target)) setOpen(false);
    };
    const escape = (event: KeyboardEvent) => {
      if (event.key === "Escape") { event.preventDefault(); setOpen(false); triggerRef.current?.focus(); }
    };
    document.addEventListener("pointerdown", outside);
    document.addEventListener("keydown", escape);
    return () => { document.removeEventListener("pointerdown", outside); document.removeEventListener("keydown", escape); };
  }, [open]);
  useEffect(() => { setOpen(false); setAccountOpen(false); }, [session?.token]);
  useEffect(() => { setOpen(false); setAccountOpen(false); setSignInOpen(false); }, [pathname]);
  if (!session) return <>
    <button className="button primary" type="button" onClick={() => setSignInOpen(true)}>{t({ en: "Sign in", zh: "登录" })}</button>
    <SignInDialog open={signInOpen} onClose={() => setSignInOpen(false)} onSuccess={onSignInSuccess} />
  </>;
  return <>
    <div className="account-menu" ref={menuRef} onBlur={(event) => {
      if (!event.currentTarget.contains(event.relatedTarget)) setOpen(false);
    }}>
      <button ref={triggerRef} className={accountStyles.trigger} type="button" aria-expanded={open} aria-controls={popoverId}
        aria-label={t({ en: "Open account options", zh: "打开账户选项" })} onClick={() => setOpen((value) => !value)} onKeyDown={(event) => {
          if (event.key === "ArrowDown") { event.preventDefault(); setOpen(true); requestAnimationFrame(() => menuRef.current?.querySelector<HTMLElement>("[data-account-item]")?.focus()); }
        }}>
        <span className="avatar-badge">{(session.user.username || session.user.email || session.user.id).slice(0, 2).toUpperCase()}</span>
        <span className={accountStyles.triggerName}>{userLabel(session.user)}</span><InterfaceIcon className={accountStyles.chevron} name="chevron" />
      </button>
      {open ? <div className={accountStyles.popover} id={popoverId} onKeyDown={(event) => {
          const items = Array.from(event.currentTarget.querySelectorAll<HTMLElement>("[data-account-item]"));
          const index = disclosureFocusIndex(event.key, items.indexOf(document.activeElement as HTMLElement), items.length);
          if (index !== null) { event.preventDefault(); items[index]?.focus(); }
        }}>
        <div className={accountStyles.identity}><strong>{session.user.username || t({ en: "Personal account", zh: "个人账户" })}</strong><span>{session.user.email}</span></div>
        <button data-account-item className={accountStyles.item} type="button" onClick={() => { setOpen(false); setAccountOpen(true); }}><InterfaceIcon name="account" />{t({ en: "Account information", zh: "账户信息" })}</button>
        <Link data-account-item className={accountStyles.item} href="/account" onClick={() => setOpen(false)}><InterfaceIcon name="account" />{t({ en: "My account", zh: "账户中心" })}</Link>
        <Link data-account-item className={accountStyles.item} href="/billing" onClick={() => setOpen(false)}><InterfaceIcon name="billing" />{t({ en: "Subscription & billing", zh: "订阅与账单" })}</Link>
        <Link data-account-item className={accountStyles.item} href="/referrals" onClick={() => setOpen(false)}><InterfaceIcon name="referrals" />{t({ en: "Referral rewards", zh: "推荐奖励" })}</Link>
      </div> : null}
    </div>
    <AccountDialog open={accountOpen} session={session} onClose={() => { setAccountOpen(false); triggerRef.current?.focus(); }} />
  </>;
}
