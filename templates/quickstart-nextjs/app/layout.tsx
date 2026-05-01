import type { Metadata } from "next";
import { appEnv } from "@/lib/env";
import { LocaleProvider } from "@/lib/i18n";
import "./globals.css";

export const metadata: Metadata = {
  title: `${appEnv.appName} | go-modules`,
  description: "Runnable Next.js host frontend for go-modules auth, billing, and referral flows."
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
