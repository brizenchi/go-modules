import type { Metadata } from "next";
import { articles } from "../content/articles";
import { appEnv } from "./env";

export const siteDescription = "Launch your SaaS faster with a free Next.js and Go template. Authentication, Stripe subscriptions, credits, referrals, Resend email, and a public website are already integrated.";
export const siteOrigin = new URL(appEnv.appUrl).origin;
export const publicPaths = ["/", "/pricing", "/docs", "/blog", "/updates", "/contact", "/privacy", "/terms"];
export const privatePaths = ["/account", "/dashboard", "/billing", "/referrals", "/credits", "/files", "/notes", "/admin", "/login", "/invite", "/oauth", "/api"];

export function canonicalURL(path: string): string {
  // Canonicals never carry invitation codes, checkout markers, or other queries.
  if (!path.startsWith("/") || path.startsWith("//") || path.includes("\\")) throw new Error("Canonical path must be local.");
  const url = new URL(path, siteOrigin);
  if (url.origin !== siteOrigin) throw new Error("Canonical path must stay on the site origin.");
  url.search = "";
  url.hash = "";
  return url.toString();
}

export function publicMetadata(title: string, description: string, path: string, article?: { publishedAt: string }): Metadata {
  const url = canonicalURL(path);
  const fullTitle = `${title} · ${appEnv.appName}`;
  return {
    title: fullTitle,
    description,
    alternates: { canonical: url },
    openGraph: {
      title: fullTitle, description, url, siteName: appEnv.appName,
      type: article ? "article" : "website",
      locale: "en_US",
      images: [{ url: canonicalURL("/opengraph-image"), width: 1200, height: 630, alt: appEnv.appName }],
      ...(article ? { publishedTime: article.publishedAt } : {})
    },
    twitter: { card: "summary_large_image", title: fullTitle, description, images: [canonicalURL("/opengraph-image")] }
  };
}

export function sitemapEntries(): Array<{ url: string; lastModified?: string }> {
  return [
    ...publicPaths.map((path) => ({ url: canonicalURL(path) })),
    ...articles.map((article) => ({ url: canonicalURL(`/blog/${article.slug}`), lastModified: article.publishedAt }))
  ];
}
