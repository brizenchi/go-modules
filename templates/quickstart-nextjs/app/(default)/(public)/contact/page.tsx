import { ContactContent } from "@/components/content/contact";
import { publicMetadata } from "@/lib/seo";

export const metadata = publicMetadata("Contact and support", "Find product guides and the support channels configured for this site.", "/contact");
export default function ContactPage() { return <ContactContent />; }
