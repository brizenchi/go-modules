import {notFound} from "next/navigation";
import {PublicPage} from "@/components/public-page";
import {managedMetadata, publishedPaths} from "@/lib/public-content";
import { articles } from "@/content/articles";
export const dynamicParams=true;
export const revalidate=30;
export async function generateStaticParams(){const paths=await publishedPaths("en",articles.map(({slug})=>`/blog/${slug}`));return paths.filter(path=>path.startsWith("/blog/")).map(path=>({slug:path.slice(6)}));}
type Props={params:Promise<{slug:string}>};
export async function generateMetadata({params}:Props){const{slug}=await params;const result=await managedMetadata(`/blog/${slug}`,"en");if(!result)notFound();return result;}
export default async function Page({params}:Props){const{slug}=await params;return <PublicPage path={`/blog/${slug}`} locale="en"/>;}
