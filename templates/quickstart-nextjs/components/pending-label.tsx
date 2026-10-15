import type { ReactNode } from "react";
import styles from "./pending-label.module.css";

export function PendingLabel({ children }: { children: ReactNode }) {
  return <span className={styles.label} role="status"><span className={styles.spinner} aria-hidden="true" />{children}</span>;
}
