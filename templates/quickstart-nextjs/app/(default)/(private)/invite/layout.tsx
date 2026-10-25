import { NavigationData } from "@/components/navigation-data";

/** The public header follows the visitor's saved language here, so both indexes are loaded. */
export default function Layout({ children }: { children: React.ReactNode }) {
  return <NavigationData locales={["en", "zh"]}>{children}</NavigationData>;
}
