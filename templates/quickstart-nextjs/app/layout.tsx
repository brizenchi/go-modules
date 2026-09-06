import type { Metadata } from "next";
import "@fontsource-variable/manrope";
import "@fontsource-variable/newsreader";
import { appEnv } from "@/lib/env";
import { LocaleProvider } from "@/lib/i18n";
import "./globals.css";

export const metadata: Metadata = {
  title: `${appEnv.appName} — Free SaaS starter`,
  description: "Launch your SaaS faster with a free Next.js and Go template. Authentication, Stripe subscriptions, credits, referrals, Resend email, and a public website are already integrated."
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
