import { ContactContent } from "@/components/content/contact";
import { managedMetadata } from "@/lib/public-content";

export const revalidate = 30;
export async function generateMetadata() { return managedMetadata("/contact", "en"); }
export default function ContactPage() { return <ContactContent />; }
