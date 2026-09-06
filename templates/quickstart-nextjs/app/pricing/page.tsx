import { ProductPricing } from "@/components/content/pricing";
import { publicMetadata } from "@/lib/seo";

export const metadata = publicMetadata("Pricing", "Compare subscriptions, lifetime access, and credit packages in the example SaaS catalog. The starter template is free.", "/pricing");
export default function PricingPage() { return <ProductPricing />; }
