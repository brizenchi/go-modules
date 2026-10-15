export type TableSort = { key: string; direction: "asc" | "desc" };

export function nextTableSort(current: TableSort | undefined, key: string): TableSort {
  return { key, direction: current?.key === key && current.direction === "asc" ? "desc" : "asc" };
}

export function tablePageState(page: number, limit: number, total: number) {
  const size = Number.isSafeInteger(limit) && limit > 0 ? limit : 10;
  const count = Number.isSafeInteger(total) && total > 0 ? total : 0;
  const pages = Math.max(1, Math.ceil(count / size));
  const current = Number.isSafeInteger(page) ? Math.min(pages, Math.max(1, page)) : 1;
  return { page: current, pages, previous: current > 1, next: current < pages };
}
