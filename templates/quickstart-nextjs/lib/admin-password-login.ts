import { ApiError, type CapabilitiesView } from "./api";
import type { AuthSession } from "./auth";

export function adminPasswordEnabled(capabilities: CapabilitiesView): boolean {
  return capabilities.auth?.admin_password_enabled === true;
}

export type AdminLoginFailure = "credentials" | "limited" | "disabled" | "unavailable";

export function describeAdminLoginFailure(error: unknown): AdminLoginFailure {
  if (error instanceof ApiError) {
    if (error.status === 401 || error.status === 403) return "credentials";
    if (error.status === 429) return "limited";
    if (error.status === 503 && error.message === "admin password login is not configured") return "disabled";
  }
  return "unavailable";
}

type AdminLoginDependencies = {
  authenticate: (email: string, password: string, signal: AbortSignal) => Promise<AuthSession>;
  readSessionToken: () => string;
  commit: (session: AuthSession) => void;
};

// The panel cancels on unmount and every session change, including a sign-in
// followed by logout that would otherwise leave the same empty session token.
export function createAdminPasswordLogin(deps: AdminLoginDependencies) {
  let generation = 0;
  let active: AbortController | null = null;
  return {
    cancel() {
      generation++;
      active?.abort();
      active = null;
    },
    async login(email: string, password: string): Promise<"committed" | "stale" | "busy"> {
      if (active) return "busy";
      const current = ++generation;
      const initialToken = deps.readSessionToken();
      const controller = new AbortController();
      active = controller;
      const timer = setTimeout(() => controller.abort(), 15000);
      const isCurrent = () => current === generation && deps.readSessionToken() === initialToken;
      try {
        const session = await deps.authenticate(email, password, controller.signal);
        if (!isCurrent()) return "stale";
        if (controller.signal.aborted) throw new Error("Administrator sign-in timed out");
        if (session.user.role !== "admin") throw new ApiError("invalid admin email or password", 403, 403);
        // Commit may synchronously emit the session event and cancel this panel.
        // No credentials are kept after the request or written to session storage.
        deps.commit(session);
        return "committed";
      } catch (error) {
        if (!isCurrent()) return "stale";
        throw error;
      } finally {
        clearTimeout(timer);
        if (current === generation) active = null;
      }
    }
  };
}
