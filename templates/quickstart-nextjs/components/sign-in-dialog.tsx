"use client";

import { useEffect, useState } from "react";
import { createPortal } from "react-dom";
import { SignInPanel } from "@/components/sign-in-panel";
import type { AuthSession } from "@/lib/auth";
import { appEnv } from "@/lib/env";
import { useI18n } from "@/lib/i18n";

type SignInDialogProps = {
  open: boolean;
  onClose: () => void;
  onSuccess?: (session: AuthSession) => void;
};

export function SignInDialog({ open, onClose, onSuccess }: SignInDialogProps) {
  const { t } = useI18n();
  const [mounted, setMounted] = useState(false);

  useEffect(() => {
    setMounted(true);
  }, []);

  useEffect(() => {
    if (!open) {
      return;
    }

    const previous = document.body.style.overflow;
    document.body.style.overflow = "hidden";

    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        onClose();
      }
    };

    window.addEventListener("keydown", onKeyDown);
    return () => {
      document.body.style.overflow = previous;
      window.removeEventListener("keydown", onKeyDown);
    };
  }, [open, onClose]);

  if (!open) {
    return null;
  }

  if (!mounted) {
    return null;
  }

  return createPortal(
    <div className="dialog-backdrop" onClick={onClose}>
      <div
        className="dialog-card"
        role="dialog"
        aria-modal="true"
        aria-label={t({ en: "Sign in", zh: "登录" })}
        onClick={(event) => event.stopPropagation()}
      >
        <button className="dialog-close" type="button" onClick={onClose} aria-label={t({ en: "Close login dialog", zh: "关闭登录弹窗" })}>
          <svg viewBox="0 0 24 24" aria-hidden="true">
            <line x1="18" y1="6" x2="6" y2="18" />
            <line x1="6" y1="6" x2="18" y2="18" />
          </svg>
        </button>

        <div className="dialog-body">
          <div className="dialog-intro">
            <h2>{t({ en: `Welcome to ${appEnv.appName}!`, zh: `欢迎使用 ${appEnv.appName}` })}</h2>
            <p>
              {t({
                en: "Choose one of the sign-in methods currently enabled for this site.",
                zh: "请选择当前网站已启用的登录方式。"
              })}
            </p>
          </div>

          <div className="dialog-auth-block">
            <p className="dialog-auth-copy">
              {t({ en: "Available methods are loaded from the backend before you continue.", zh: "继续前会先从后端加载实际可用的登录方式。" })}
            </p>
            <SignInPanel
              compact
              onSuccess={(session) => {
                onClose();
                onSuccess?.(session);
              }}
            />
          </div>

          <p className="dialog-support-copy">
            {t({
              en: "By continuing you accept this site's authentication flow and session storage policy.",
              zh: "继续即表示你接受当前网站的登录流程与会话存储策略。"
            })}
          </p>
        </div>
      </div>
    </div>,
    document.body
  );
}
