/** Return a boundary focus target; null preserves the browser's normal Tab order. */
export function dialogFocusTarget(current: number, count: number, backwards: boolean): number | null {
  if (count < 1) return null;
  if (current < 0) return backwards ? count - 1 : 0;
  if (backwards && current === 0) return count - 1;
  if (!backwards && current === count - 1) return 0;
  return null;
}
