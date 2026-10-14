import { notFound } from "next/navigation";
import { PublicPage } from "@/components/public-page";
import { publicPageMetadata } from "@/lib/seo";
import { marketingRoutes } from "@/content/marketing";
export const dynamicParams = false;
export function generateStaticParams() { return marketingRoutes.map(page => page.path).map(path => ({ slug: path === "/" ? [] : path.slice(1).split("/") })); }
type Props = { params: Promise<{ slug?: string[] }> };
export async function generateMetadata({ params }: Props) {
 const { slug = [] } = await params;
 const metadata = publicPageMetadata(`/${slug.join("/")}`, "en");
 if (!metadata) notFound();
 return metadata;
}
export default async function Page({ params }: Props) {
 const { slug = [] } = await params;
 return <PublicPage path={`/${slug.join("/")}`} />;
}
