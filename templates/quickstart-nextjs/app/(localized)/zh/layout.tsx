import type { Metadata } from "next";
import "@fontsource-variable/manrope";
import "@fontsource-variable/newsreader";
import { appEnv } from "@/lib/env";
import { LocaleProvider } from "@/lib/i18n";
import { siteOrigin } from "@/lib/seo";
import "../../globals.css";

export const metadata: Metadata = {
  metadataBase: new URL(siteOrigin),
  applicationName: appEnv.appName
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="zh-CN" suppressHydrationWarning>
      <body>
        <LocaleProvider initialLocale="zh">{children}</LocaleProvider>
      </body>
    </html>
  );
}
