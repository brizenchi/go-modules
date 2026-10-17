export const PAGE_RETURN_STALE_MS = 60_000;

/** Ignore brief switches and consume each actual return only once. */
export function createPageReturnRefresh(now: () => number = Date.now) {
  let awaySince: number | null = null;
  return {
    leave() { awaySince ??= now(); },
    shouldRefresh(): boolean {
      const since = awaySince;
      awaySince = null;
      return since !== null && now() - since >= PAGE_RETURN_STALE_MS;
    }
  };
}
