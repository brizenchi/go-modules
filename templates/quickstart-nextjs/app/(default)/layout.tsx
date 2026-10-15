import { SitePresentationProvider } from "@/components/site-presentation";
import { websitePresentation } from "@/lib/public-content";
import type { Metadata } from "next";
import "@fontsource-variable/manrope";
import "@fontsource-variable/newsreader";
import { appEnv } from "@/lib/env";
import { SiteExtensions } from "@/components/site-extensions";
import { LocaleProvider } from "@/lib/i18n";
import { siteOrigin } from "@/lib/seo";
import "../globals.css";

export const revalidate = 30;
export const metadata: Metadata = {
  metadataBase: new URL(siteOrigin),
  applicationName: appEnv.appName
};

export default async function RootLayout({ children }: { children: React.ReactNode }) {
  const presentation=await websitePresentation();
  return (
    <html lang="en" suppressHydrationWarning>
      <body>
        <SitePresentationProvider value={presentation}><LocaleProvider>{children}<SiteExtensions /></LocaleProvider></SitePresentationProvider>
      </body>
    </html>
  );
}
