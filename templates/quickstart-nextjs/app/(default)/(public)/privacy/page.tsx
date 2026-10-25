import { PolicyContent } from "@/components/content/policy";
import { managedMetadata } from "@/lib/public-content";

export const revalidate = 30;
export async function generateMetadata() { return managedMetadata("/privacy", "en"); }
export default function PrivacyPage() { return <PolicyContent kind="privacy" />; }
