export type RequestGenerationRef = { current: number };

export function beginRequestGeneration(ref: RequestGenerationRef): number {
  ref.current += 1;
  return ref.current;
}

export function invalidateRequestGeneration(ref: RequestGenerationRef): void {
  ref.current += 1;
}

export function isCurrentRequestGeneration(
  ref: RequestGenerationRef,
  generation: number
): boolean {
  return ref.current === generation;
}
