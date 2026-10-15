import { managedMetadata } from "@/lib/public-content";
export const revalidate=30;
export async function generateMetadata(){return managedMetadata("/","en");}
export default function PublicLayout({children}:{children:React.ReactNode}){return children;}
