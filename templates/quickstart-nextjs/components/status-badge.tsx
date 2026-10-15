import type { ReactNode } from "react";
import type { StatusTone } from "../lib/admin-presentation";
import styles from "./status-badge.module.css";

export function StatusBadge({ tone = "neutral", children }: { tone?: StatusTone; children: ReactNode }) {
  return <span className={`${styles.badge} ${styles[tone]}`}>{children}</span>;
}
