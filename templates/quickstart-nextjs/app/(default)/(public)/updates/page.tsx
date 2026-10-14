import { ProductUpdates } from "@/components/content/updates";
import { publicMetadata } from "@/lib/seo";

export const metadata = publicMetadata("Product updates", "Follow changes to the SaaS starter, with links to the features and guides.", "/updates");
export default function UpdatesPage() { return <ProductUpdates />; }
