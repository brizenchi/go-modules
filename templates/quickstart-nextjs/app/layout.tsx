import type { Metadata } from "next";
import "@fontsource-variable/manrope";
import "@fontsource-variable/newsreader";
import { appEnv } from "@/lib/env";
import { LocaleProvider } from "@/lib/i18n";
import { publicMetadata, siteDescription, siteOrigin } from "@/lib/seo";
import "./globals.css";

export const metadata: Metadata = {
  ...publicMetadata("Free SaaS starter", siteDescription, "/"),
  metadataBase: new URL(siteOrigin),
  applicationName: appEnv.appName
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body>
        <LocaleProvider>{children}</LocaleProvider>
      </body>
    </html>
  );
}
