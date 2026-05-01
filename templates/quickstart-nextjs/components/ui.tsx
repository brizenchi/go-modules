import Link from "next/link";

type PanelProps = {
  title: string;
  subtitle?: string;
  children: React.ReactNode;
  actions?: React.ReactNode;
  className?: string;
};

export function Panel({ title, subtitle, children, actions, className = "span-12" }: PanelProps) {
  return (
    <section className={`panel ${className}`}>
      <div className="panel-title-row">
        <div>
          <h2>{title}</h2>
          {subtitle ? <p className="muted">{subtitle}</p> : null}
        </div>
        {actions}
      </div>
      {children}
    </section>
  );
}

type NoticeProps = {
  tone?: "default" | "success" | "error";
  children: React.ReactNode;
};

export function Notice({ tone = "default", children }: NoticeProps) {
  const className = tone === "default" ? "notice" : `notice ${tone}`;
  return <div className={className}>{children}</div>;
}

export function LabelPill({ children }: { children: React.ReactNode }) {
  return <span className="label-pill">{children}</span>;
}

export function DetailRows({ rows }: { rows: Array<{ label: string; value: React.ReactNode }> }) {
  return (
    <div className="details-list">
      {rows.map((row) => (
        <div className="details-row" key={row.label}>
          <span className="detail-label">{row.label}</span>
          <span>{row.value}</span>
        </div>
      ))}
    </div>
  );
}

export function EmptyState({ children }: { children: React.ReactNode }) {
  return <div className="empty-state">{children}</div>;
}

export function CTAButton({
  href,
  children,
  primary = false
}: {
  href: string;
  children: React.ReactNode;
  primary?: boolean;
}) {
  return (
    <Link className={`button${primary ? " primary" : ""}`} href={href}>
      {children}
    </Link>
  );
}

export function PageSection({
  id,
  title,
  description,
  children
}: {
  id?: string;
  title: string;
  description?: string;
  children: React.ReactNode;
}) {
  return (
    <section className="page-section" id={id}>
      <div className="section-head">
        <h2>{title}</h2>
        {description ? <p>{description}</p> : null}
      </div>
      {children}
    </section>
  );
}
