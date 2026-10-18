"use client";

import { OperatorConsole } from "./operator-console";
import type { AdminSection } from "@/lib/admin-navigation";

// Keep the route's client boundary distinct from the shared console graph.
export default function AdminConsoleEntry({ section }: { section: AdminSection }) {
  return <OperatorConsole section={section} />;
}
