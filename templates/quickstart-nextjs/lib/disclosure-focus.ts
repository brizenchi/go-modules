/** Resolve optional arrow-key navigation without changing native Tab behavior. */
export function disclosureFocusIndex(key: string, current: number, count: number): number | null {
  if (count < 1) return null;
  if (key === "Home") return 0;
  if (key === "End") return count - 1;
  if (key === "ArrowDown") return current < 0 ? 0 : (current + 1) % count;
  if (key === "ArrowUp") return current < 0 ? count - 1 : (current + count - 1) % count;
  return null;
}
