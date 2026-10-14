import { notFound } from "next/navigation";
import { GuideArticle } from "@/components/content/guide-article";
import { documentation, findDocumentation } from "@/content/docs";
import { publicMetadata } from "@/lib/seo";

export const dynamicParams = false;

export function generateStaticParams() {
  return documentation.map((guide) => ({ slug: guide.id }));
}

export async function generateMetadata({ params }: { params: Promise<{ slug: string }> }) {
  const { slug } = await params;
  const guide = findDocumentation(slug);
  if (!guide) notFound();
  return publicMetadata(guide.title.en, guide.summary.en, `/docs/${guide.id}`, {});
}

export default async function GuidePage({ params }: { params: Promise<{ slug: string }> }) {
  const { slug } = await params;
  const guide = findDocumentation(slug);
  if (!guide) notFound();
  return <GuideArticle guide={guide} />;
}
