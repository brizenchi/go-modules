import { ContentAdminEditor } from "@/components/content-admin";
export default async function Page({params}:{params:Promise<{id:string}>}){const{id}=await params;return <ContentAdminEditor key={id} id={id}/>;}
