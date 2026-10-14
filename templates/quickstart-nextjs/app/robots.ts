import type { MetadataRoute } from "next";
import { canonicalURL } from "@/lib/seo";

export default function robots(): MetadataRoute.Robots {
  return {
    // Workspace HTML must remain crawlable so robots can read its noindex metadata.
    // Authentication still controls access to all account data independently.
    rules: { userAgent: "*", allow: "/", disallow: ["/api", "/*?*ref=", "/*?*token=", "/*?*code=", "/*?*checkout="] },
    sitemap: canonicalURL("/sitemap.xml")
  };
}
