/** API snapshots are JSON values; object key order does not change their content. */
function equalData(left: unknown, right: unknown): boolean {
  if (Object.is(left, right)) return true;
  if (Array.isArray(left) || Array.isArray(right)) {
    return Array.isArray(left) && Array.isArray(right) && left.length === right.length
      && left.every((value: unknown, index: number) => equalData(value, right[index]));
  }
  if (!left || !right || typeof left !== "object" || typeof right !== "object") return false;
  const a = left as Record<string, unknown>;
  const b = right as Record<string, unknown>;
  const keys = Object.keys(a);
  return keys.length === Object.keys(b).length
    && keys.every(key => Object.hasOwn(b, key) && equalData(a[key], b[key]));
}

export function retainEqualData<T>(previous: T | null, next: T): T {
  return previous !== null && equalData(previous, next) ? previous : next;
}
