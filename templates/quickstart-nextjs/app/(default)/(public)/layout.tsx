import { publicPageMetadata } from "@/lib/seo";
export const metadata = publicPageMetadata("/", "en");
export default function PublicLayout({ children }: { children: React.ReactNode }) { return children; }
