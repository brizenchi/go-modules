import { notFound } from "next/navigation";
import { PublicPage } from "@/components/public-page";
import { publicContentPaths, publicPageMetadata } from "@/lib/seo";
export const dynamicParams = false;
export function generateStaticParams() { return publicContentPaths().map(path => ({ slug: path === "/" ? [] : path.slice(1).split("/") })); }
type Props = { params: Promise<{ slug?: string[] }> };
export async function generateMetadata({ params }: Props) {
 const { slug = [] } = await params;
 const metadata = publicPageMetadata(`/${slug.join("/")}`, "zh");
 if (!metadata) notFound();
 return metadata;
}
export default async function Page({ params }: Props) {
 const { slug = [] } = await params;
 return <PublicPage path={`/${slug.join("/")}`} />;
}
