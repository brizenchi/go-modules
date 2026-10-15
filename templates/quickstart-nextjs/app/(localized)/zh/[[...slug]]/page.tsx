import { notFound } from "next/navigation";
import { PublicPage } from "@/components/public-page";
import { publicContentPaths } from "@/lib/seo";
import { managedMetadata, publishedPaths } from "@/lib/public-content";
export const dynamicParams = true;
export const revalidate = 30;
export async function generateStaticParams() {
 const paths=await publishedPaths("zh",publicContentPaths());
 return paths.map(path=>({slug:path==="/"?[]:path.slice(1).split("/")}));
}
type Props = { params: Promise<{ slug?: string[] }> };
export async function generateMetadata({ params }: Props) {
 const { slug = [] } = await params;
 const metadata = await managedMetadata(`/${slug.join("/")}`, "zh");
 if (!metadata) notFound();
 return metadata;
}
export default async function Page({ params }: Props) {
 const { slug = [] } = await params;
 return <PublicPage path={`/${slug.join("/")}`} locale="zh" />;
}
