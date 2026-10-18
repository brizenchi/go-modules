import { redirect } from "next/navigation";
import { notFound } from "next/navigation";
import AdminConsoleEntry from "@/components/admin-console-entry";
import { adminSections, type AdminSection } from "@/lib/admin-navigation";
const sections: readonly string[] = adminSections.filter((section) => section !== "overview");
export function generateStaticParams() { return sections.map((section) => ({ section })); }
export default async function AdminSectionPage({ params }: { params: Promise<{ section: string }> }) {
  const { section } = await params;
  if (!sections.includes(section)) notFound();
  if (section==="orders")redirect("/admin/users?view=payments");
  if (section==="referrals")redirect("/admin/users?view=invitations");
  if (section==="subscriptions"||section==="credits")redirect("/admin/users");
  if (section==="integrations")redirect("/admin/settings");
  return <AdminConsoleEntry section={section as AdminSection} />;
}
