import { PolicyContent } from "@/components/content/policy";
import { publicMetadata } from "@/lib/seo";

export const metadata = publicMetadata("Terms", "Editable service terms covering accounts, subscriptions, credits, and support for your product.", "/terms");
export default function TermsPage() { return <PolicyContent kind="terms" />; }
