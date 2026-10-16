import { notFound } from "next/navigation";
import { PublicPage } from "@/components/public-page";
import { managedMetadata, publishedPaths } from "@/lib/public-content";
import { marketingRoutes } from "@/content/marketing";
export const dynamicParams = true;
export const revalidate = 30;
export async function generateStaticParams() {
 const paths=await publishedPaths("en",marketingRoutes.map(page=>page.path));
 return paths.filter(path=>path !== "/" && path !== "/updates" && !/^\/(blog|docs)(?:\/|$)/.test(path)).map(path=>({slug:path.slice(1).split("/")}));
}
type Props = { params: Promise<{ slug?: string[] }> };
export async function generateMetadata({ params }: Props) {
 const { slug = [] } = await params;
 const metadata = await managedMetadata(`/${slug.join("/")}`, "en");
 if (!metadata) notFound();
 return metadata;
}
export default async function Page({ params }: Props) {
 const { slug = [] } = await params;
 return <PublicPage path={`/${slug.join("/")}`} locale="en" />;
}
