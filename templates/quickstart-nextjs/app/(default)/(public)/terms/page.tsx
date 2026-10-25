import { PolicyContent } from "@/components/content/policy";
import { managedMetadata } from "@/lib/public-content";

export const revalidate = 30;
export async function generateMetadata() { return managedMetadata("/terms", "en"); }
export default function TermsPage() { return <PolicyContent kind="terms" />; }
