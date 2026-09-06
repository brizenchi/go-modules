import type { MetadataRoute } from "next";
import { canonicalURL, privatePaths } from "@/lib/seo";

export default function robots(): MetadataRoute.Robots {
  return {
    rules: { userAgent: "*", allow: "/", disallow: [...privatePaths, "/*?*ref=", "/*?*token=", "/*?*code=", "/*?*checkout="] },
    sitemap: canonicalURL("/sitemap.xml")
  };
}
