import { PolicyContent } from "@/components/content/policy";
import { publicMetadata } from "@/lib/seo";

export const metadata = publicMetadata("Privacy", "Editable privacy information for the SaaS starter. Adapt it to your actual service.", "/privacy");
export default function PrivacyPage() { return <PolicyContent kind="privacy" />; }
