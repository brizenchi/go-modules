import { NavigationIndexProvider } from "./navigation-index";
import { navigationIndex } from "@/lib/public-content";
import type { Locale } from "@/lib/locale";

/** Server wrapper: pages with a fixed language load one index, sign-in pages both. */
export async function NavigationData({ locales, children }: { locales: readonly Locale[]; children: React.ReactNode }) {
  const indexes = await Promise.all(locales.map(locale => navigationIndex(locale)));
  return <NavigationIndexProvider value={Object.fromEntries(locales.map((locale, index) => [locale, indexes[index]]))}>{children}</NavigationIndexProvider>;
}
