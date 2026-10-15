import {PublicPage} from "@/components/public-page";
import {managedMetadata} from "@/lib/public-content";
export const revalidate=30;
export async function generateMetadata(){return managedMetadata("/blog","en");}
export default function Page(){return <PublicPage path="/blog" locale="en"/>;}
