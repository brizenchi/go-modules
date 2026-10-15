"use client";

import type { ReactNode } from "react";
import { adminNavigationGroups, adminSectionDescriptions, adminSectionNames, type AdminSection } from "@/lib/admin-navigation";
import { useI18n } from "@/lib/i18n";
import { InterfaceIcon } from "./interface-icon";
import { PendingLabel } from "./pending-label";
import styles from "./admin-shell.module.css";

export function AdminPageHeader({ section, onRefresh, busy, disabled, actions }: { section: AdminSection; onRefresh?: () => void; busy?: boolean; disabled?: boolean; actions?: ReactNode }) {
  const { t } = useI18n();
  const group = adminNavigationGroups.find(item => item.sections.includes(section));
  return <div className={styles.pageHeader}><div>{group ? <span className={styles.eyebrow}>{t(group.label)}</span> : null}<h1>{t(adminSectionNames[section])}</h1><p>{t(adminSectionDescriptions[section])}</p></div>
    <div className={styles.pageActions}>{actions}{onRefresh ? <button type="button" className="button" disabled={busy || disabled} aria-busy={busy} onClick={onRefresh}>{busy ? <PendingLabel>{t({ en: "Refreshing…", zh: "刷新中…" })}</PendingLabel> : <><InterfaceIcon name="refresh" />{t({ en: "Refresh", zh: "刷新" })}</>}</button> : null}</div>
  </div>;
}
