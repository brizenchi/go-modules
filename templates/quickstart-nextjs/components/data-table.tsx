"use client";

import { useEffect, type ReactNode } from "react";
import { nextTableSort, tablePageState, type TableSort } from "../lib/table-state";

export type DataColumn<T> = {
  key: string;
  heading: ReactNode;
  cell: (row: T) => ReactNode;
  align?: "start" | "end";
  sortable?: boolean;
};
type DataTableProps<T> = {
  caption: string;
  columns: readonly DataColumn<T>[];
  rows: readonly T[];
  rowKey: (row: T) => string | number;
  loading?: boolean;
  loadingLabel: string;
  empty: ReactNode;
  error?: ReactNode;
  sort?: TableSort;
  onSort?: (sort: TableSort) => void;
  pagination?: { page: number; limit: number; total: number; onPage: (page: number) => void; previousLabel: string; nextLabel: string; label: (page: number, pages: number) => string };
};
export function DataTable<T>({ caption, columns, rows, rowKey, loading = false, loadingLabel, empty, error, sort, onSort, pagination }: DataTableProps<T>) {
  const page = pagination ? tablePageState(pagination.page, pagination.limit, pagination.total) : null;
  useEffect(() => {
    if (!loading && !error && pagination && page && pagination.page !== page.page) pagination.onPage(page.page);
  }, [loading,error,pagination,page]);
  return <div className="data-table">
    <div className="data-table-scroll" role="region" aria-label={caption} tabIndex={0}>
      <table className="table" aria-busy={loading}>
        <caption className="sr-only">{caption}</caption>
        <thead><tr>{columns.map(column => <th key={column.key} scope="col" className={column.align === "end" ? "data-table-end" : undefined}
          aria-sort={column.sortable && onSort ? sort?.key === column.key ? sort.direction === "asc" ? "ascending" : "descending" : "none" : undefined}>
          {column.sortable && onSort ? <button type="button" className="data-table-sort" disabled={loading} onClick={() => onSort(nextTableSort(sort, column.key))}>{column.heading}<span aria-hidden="true">{sort?.key === column.key ? sort.direction === "asc" ? "↑" : "↓" : "↕"}</span></button> : column.heading}
        </th>)}</tr></thead>
        <tbody>{rows.map(row => <tr key={rowKey(row)}>{columns.map(column => <td key={column.key} className={column.align === "end" ? "data-table-end" : undefined}>{column.cell(row)}</td>)}</tr>)}
          {rows.length === 0 ? <tr><td colSpan={Math.max(1,columns.length)} className="data-table-empty">{error || (loading ? <span role="status">{loadingLabel}</span> : empty)}</td></tr> : null}
        </tbody>
      </table>
    </div>
    {rows.length > 0 && error ? error : null}
    {loading && rows.length > 0 ? <p role="status" className="data-table-status">{loadingLabel}</p> : null}
    {pagination && page ? <nav className="data-table-pagination" aria-label={caption}>
      <span aria-live="polite">{pagination.label(page.page,page.pages)}</span>
      <div className="button-row"><button className="button" type="button" disabled={loading || !page.previous} onClick={() => pagination.onPage(page.page-1)}>{pagination.previousLabel}</button><button className="button" type="button" disabled={loading || !page.next} onClick={() => pagination.onPage(page.page+1)}>{pagination.nextLabel}</button></div>
    </nav> : null}
  </div>;
}
