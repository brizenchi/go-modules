"use client";

import { LocalizedLink as Link } from "@/components/localized-link";
import { ResponsiveDialog } from "@/components/responsive-dialog";
import { SignInPanel } from "@/components/sign-in-panel";
import { InterfaceIcon } from "@/components/interface-icon";
import type { AuthSession } from "@/lib/auth";
import { useI18n } from "@/lib/i18n";
import { resolveSignInDestination } from "@/lib/oauth-flow";
import styles from "./sign-in-dialog.module.css";

type SignInDialogProps = {
  open: boolean;
  onClose: () => void;
  onSuccess?: (session: AuthSession) => void;
  returnTo?: string;
};

export function SignInDialog({ open, onClose, onSuccess, returnTo }: SignInDialogProps) {
  const { t } = useI18n();
  return <ResponsiveDialog open={open} onClose={onClose}
    title={t({ en: "Sign in", zh: "登录" })}
    description={t({ en: "Sign in or create an account to continue.", zh: "登录或创建账户，继续使用。" })}
    closeLabel={t({ en: "Close login dialog", zh: "关闭登录弹窗" })}
    icon={<InterfaceIcon name="account" />}>
    <div className={styles.form}>
      <SignInPanel compact
        returnTo={open && typeof window !== "undefined" ? resolveSignInDestination(`${window.location.pathname}${window.location.search}${window.location.hash}`, returnTo) : undefined}
        onSuccess={(session) => { onClose(); onSuccess?.(session); }} />
    </div>
    <p className={styles.support}>
      <Link href="/terms">{t({ en: "Terms of service", zh: "服务条款" })}</Link><span aria-hidden="true">·</span>
      <Link href="/privacy">{t({ en: "Privacy policy", zh: "隐私政策" })}</Link>
    </p>
  </ResponsiveDialog>;
}
