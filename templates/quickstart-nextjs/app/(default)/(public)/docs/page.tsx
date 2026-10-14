import { Documentation } from "@/components/content/docs";
import { publicMetadata } from "@/lib/seo";

export const metadata = publicMetadata("Template user guide", "Learn what the free SaaS template includes, try sign-in, payments and referrals, then configure your own website and administrator account.", "/docs");
export default function DocsPage() { return <Documentation />; }
