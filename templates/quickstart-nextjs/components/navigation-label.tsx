"use client";

import { useLinkStatus } from "next/link";
import { useI18n } from "@/lib/i18n";
import { NavigationFeedback } from "./navigation-feedback";

export function NavigationLabel({ label, className, statusClassName }: { label: string; className?: string; statusClassName?: string }) {
  const { pending } = useLinkStatus();
  const { t } = useI18n();
  return <NavigationFeedback pending={pending} label={label}
    pendingLabel={t({ en: `Opening ${label}…`, zh: `正在打开${label}…` })}
    className={className} statusClassName={statusClassName} />;
}
