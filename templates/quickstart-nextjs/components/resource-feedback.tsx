import type { RequestFailure } from "../lib/request-state";

export function ResourceFailure({
  failure,
  onRetry,
  retryLabel = "Try again",
  compact = false
}: {
  failure: RequestFailure;
  onRetry?: () => void;
  retryLabel?: string;
  compact?: boolean;
}) {
  return (
    <div className={`resource-feedback ${failure.kind}${compact ? " compact" : ""}`} role="alert">
      <span className="resource-feedback-mark" aria-hidden="true">
        {failure.kind === "auth" ? "↗" : failure.kind === "disabled" ? "—" : "!"}
      </span>
      <div className="resource-feedback-copy">
        <strong>{failure.title}</strong>
        <p>{failure.message}</p>
        {!compact ? (
          <div className="resource-feedback-actions">
            {failure.kind === "auth" ? (
              <a className="button primary" href="/login">Sign in</a>
            ) : failure.retryable && onRetry ? (
              <button className="button" type="button" onClick={onRetry}>{retryLabel}</button>
            ) : null}
          </div>
        ) : null}
      </div>
    </div>
  );
}

export function SignInRequired({ message, title = "Sign in to continue", actionLabel = "Open sign in", onSignIn }: { message: string; title?: string; actionLabel?: string; onSignIn?: () => void }) {
  return (
    <div className="resource-feedback auth" role="status">
      <span className="resource-feedback-mark" aria-hidden="true">↗</span>
      <div className="resource-feedback-copy">
        <strong>{title}</strong>
        <p>{message}</p>
        <div className="resource-feedback-actions">
          {onSignIn ? <button className="button primary" type="button" onClick={onSignIn}>{actionLabel}</button> : <a className="button primary" href="/login">{actionLabel}</a>}
        </div>
      </div>
    </div>
  );
}
