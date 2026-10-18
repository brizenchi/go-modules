import type { SubscriptionView } from "./api";

/** One login's in-memory presentation snapshot; purchase checks always use the API. */
export function createPricingSubscriptionCache(load: (token: string) => Promise<SubscriptionView>, now: () => number = Date.now) {
  let owner = "";
  let data: SubscriptionView | null = null;
  let loadedAt = 0;
  let generation = 0;
  let pending: Promise<SubscriptionView> | null = null;
  function clear() { owner = ""; data = null; loadedAt = 0; pending = null; generation++; }
  function invalidate() { loadedAt = -Infinity; pending = null; generation++; }
  return {
    clear,
    invalidate,
    peek(token: string) { return token && owner === token ? data : null; },
    load(token: string, force = false): Promise<SubscriptionView> {
      if (owner !== token) { clear(); owner = token; }
      if (force) invalidate();
      if (pending) return pending;
      if (!force && data && now() - loadedAt < 60_000) return Promise.resolve(data);
      const current = generation;
      const request = load(token).then(value => {
        if (generation === current && owner === token) { data = value; loadedAt = now(); }
        return value;
      }).finally(() => { if (generation === current) pending = null; });
      pending = request;
      return request;
    }
  };
}
