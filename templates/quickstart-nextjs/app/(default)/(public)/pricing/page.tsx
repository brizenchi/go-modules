import { ProductPricing } from "@/components/content/pricing";
import { managedMetadata } from "@/lib/public-content";

export const revalidate = 30;
export async function generateMetadata() { return managedMetadata("/pricing", "en"); }
export default function PricingPage() { return <ProductPricing />; }
