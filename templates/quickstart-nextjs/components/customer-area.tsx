"use client";

import { useEffect, useState, type ReactNode } from "react";
import { usePathname } from "next/navigation";
import { useSitePresentation } from "./site-presentation";
import { AccountMenu } from "@/components/account-menu";
import { CustomerShell } from "@/components/customer-shell";
import { LocaleSwitch } from "@/components/locale-switch";
import { readSession, SESSION_EVENT, type AuthSession } from "@/lib/auth";
import { appEnv } from "@/lib/env";
import { activeWorkspaceHref } from "@/lib/workspace";

/** Mounted by the shared route layout, so navigation does not recreate the frame. */
export function CustomerArea({ children }: { children: ReactNode }) {
  const presentation=useSitePresentation();
  const pathname = usePathname() || "/account";
  const [session, setSession] = useState<AuthSession | null>(null);
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
  return <CustomerShell brand={presentation?.brand_name || appEnv.appName} activeHref={activeWorkspaceHref(pathname)}
    accountMenu={<AccountMenu session={session} />} localeControl={<LocaleSwitch />}>
    {children}
  </CustomerShell>;
}
