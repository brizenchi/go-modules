import { notFound } from "next/navigation";
import { OperatorConsole, type OperatorSection } from "@/components/operator-console";
const sections = ["users", "orders", "subscriptions", "referrals", "credits", "settings", "audit"];
export function generateStaticParams() { return sections.map((section) => ({ section })); }
export default async function AdminSectionPage({ params }: { params: Promise<{ section: string }> }) {
  const { section } = await params;
  if (!sections.includes(section)) notFound();
  return <OperatorConsole section={section as OperatorSection} />;
}
