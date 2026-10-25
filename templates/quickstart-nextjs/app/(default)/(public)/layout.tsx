import { NavigationData } from "@/components/navigation-data";

export const revalidate=30;
export default function PublicLayout({children}:{children:React.ReactNode}){
  return <NavigationData locales={["en"]}>{children}</NavigationData>;
}
