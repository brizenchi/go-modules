import { Documentation } from "@/components/content/docs";
import { publicMetadata } from "@/lib/seo";

export const metadata = publicMetadata("Setup documentation", "Configure domains, authentication, Resend, Stripe, invitations, and public content for your SaaS.", "/docs");
export default function DocsPage() { return <Documentation />; }
