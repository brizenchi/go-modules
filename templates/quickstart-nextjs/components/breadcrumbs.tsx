import Link from "next/link";
import { breadcrumbStructuredData, type BreadcrumbItem } from "../lib/structured-data";
import { StructuredData } from "./structured-data";

export function Breadcrumbs({ items, path, label, structured = false }: {
  items: BreadcrumbItem[];
  path: string;
  label: string;
  structured?: boolean;
}) {
  if (items.length < 2) return null;
  return (
    <nav className="content-breadcrumbs" aria-label={label}>
      <ol>
        {items.map((item, index) => (
          <li key={`${item.label}-${index}`}>
            {index > 0 ? <span className="content-breadcrumb-separator" aria-hidden="true">/</span> : null}
            {index === items.length - 1
              ? <span aria-current="page">{item.label}</span>
              : item.href ? <Link href={item.href}>{item.label}</Link> : <span>{item.label}</span>}
          </li>
        ))}
      </ol>
      {structured ? <StructuredData data={breadcrumbStructuredData(items, path)} /> : null}
    </nav>
  );
}
