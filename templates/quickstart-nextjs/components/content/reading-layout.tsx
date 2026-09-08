import type { ReactNode } from "react";
import Link from "next/link";
import styles from "./reading-layout.module.css";

export function ReadingLayout({ children, toc, title, supportLabel }: {
  children: ReactNode;
  toc: { id: string; label: string }[];
  title: string;
  supportLabel: string;
}) {
  return (
    <div className={styles.layout}>
      <div className={styles.body}>{children}</div>
      <aside className={styles.aside}>
        <nav aria-label={title}>
          <h2>{title}</h2>
          <ol>{toc.map((item) => <li key={item.id}><a href={`#${item.id}`}>{item.label}</a></li>)}</ol>
        </nav>
        <Link className={styles.support} href="/contact">{supportLabel} <span aria-hidden="true">↗</span></Link>
      </aside>
    </div>
  );
}
