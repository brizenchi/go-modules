import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { SitePresentationProvider } from "./site-presentation";
import { NavigationData } from "./navigation-data";
import { SiteExtensions } from "./site-extensions";
import { PublicPage } from "./public-page";
import { LocaleProvider } from "@/lib/i18n";
import { appEnv } from "@/lib/env";
import { siteOrigin } from "@/lib/seo";
import { isEditorialPath } from "@/lib/content-discovery";
import { managedMetadata, publishedIndex, websitePresentation } from "@/lib/public-content";
import { languageTags, type ContentLocale } from "@/lib/locale";

// Shared by the routes of article languages without an interface translation
// (/ja, /ko, /es, /fr, /de): only published articles exist there, in that
// language, with the English interface. Add a language by copying one folder.

export const articleLanguageMetadata: Metadata = { metadataBase: new URL(siteOrigin), applicationName: appEnv.appName };

export async function ArticleLanguageLayout({ locale, children }: { locale: ContentLocale; children: React.ReactNode }) {
  const presentation = await websitePresentation();
  return (
    <html lang={languageTags[locale]} suppressHydrationWarning>
      <body>
        <SitePresentationProvider value={presentation}><NavigationData locales={["en"]}><LocaleProvider initialLocale="en">{children}<SiteExtensions /></LocaleProvider></NavigationData></SitePresentationProvider>
      </body>
    </html>
  );
}

export async function articleLanguageParams(locale: ContentLocale): Promise<Array<{ slug: string[] }>> {
  const rows = await publishedIndex(locale);
  return (rows || []).filter(row => isEditorialPath(row.path)).map(row => ({ slug: row.path.slice(1).split("/") }));
}

export async function articleLanguageMetadataFor(locale: ContentLocale, slug: string[]): Promise<Metadata> {
  const metadata = await managedMetadata(`/${slug.join("/")}`, locale);
  if (!metadata) notFound();
  return metadata;
}

export function ArticleLanguagePage({ locale, slug }: { locale: ContentLocale; slug: string[] }) {
  return <PublicPage path={`/${slug.join("/")}`} contentLocale={locale} />;
}
