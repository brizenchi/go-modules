"use client";

import { useI18n } from "@/lib/i18n";
import styles from "./customer-shell.module.css";

export function CustomerPageLoading() {
  const { t } = useI18n();
  return <section className={styles.pageLoading} aria-busy="true" role="status">
    <span className="sr-only">{t({ en: "Loading page…", zh: "正在加载页面…" })}</span>
    <div className={styles.loadingHeading} aria-hidden="true"><span /><span /></div>
    <div className={styles.loadingPanel} aria-hidden="true">{[0, 1, 2].map(row => <div key={row}><span /><span /></div>)}</div>
  </section>;
}
