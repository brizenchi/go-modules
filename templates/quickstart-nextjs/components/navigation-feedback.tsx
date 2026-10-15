/** Router-owned pending state; the destination label stays readable throughout. */
export function NavigationFeedback({ pending, label, pendingLabel, className, statusClassName }: {
  pending: boolean;
  label: string;
  pendingLabel: string;
  className?: string;
  statusClassName?: string;
}) {
  return <span className={className} data-navigation-pending={pending ? "true" : undefined}>
    <span>{label}</span>
    {pending ? <span className={statusClassName} role="status">
      <svg viewBox="0 0 20 20" width="14" height="14" fill="none" aria-hidden="true"><circle cx="10" cy="10" r="7" stroke="currentColor" strokeWidth="2" opacity=".2" /><path d="M10 3a7 7 0 0 1 7 7" stroke="currentColor" strokeWidth="2" strokeLinecap="round" /></svg>
      <span className="sr-only">{pendingLabel}</span>
    </span> : null}
  </span>;
}
