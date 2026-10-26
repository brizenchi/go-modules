import { appEnv } from "@/lib/env";
import { llmsText } from "@/lib/llms";
import { discoveryIndex, managedMetadata, publishedDirectories, websitePresentation } from "@/lib/public-content";
import { siteConfig } from "@/content/site";

export const revalidate = 30;

export async function GET() {
  const [entries, directories, metadata, site] = await Promise.all([discoveryIndex("en"), publishedDirectories("en"), managedMetadata("/", "en"), websitePresentation()]);
  const description = typeof metadata?.description === "string" && metadata.description ? metadata.description : siteConfig.description.en;
  const body = llmsText({ name: site?.brand_name || appEnv.appName, description, entries, directories });
  return new Response(body, { headers: { "Content-Type": "text/plain; charset=utf-8" } });
}
