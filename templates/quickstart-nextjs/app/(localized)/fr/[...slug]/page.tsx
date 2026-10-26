import { ArticleLanguagePage, articleLanguageMetadataFor, articleLanguageParams } from "@/components/article-language-route";

export const dynamicParams = true;
export const revalidate = 30;
type Props = { params: Promise<{ slug: string[] }> };
export async function generateStaticParams() { return articleLanguageParams("fr"); }
export async function generateMetadata({ params }: Props) { return articleLanguageMetadataFor("fr", (await params).slug); }
export default async function Page({ params }: Props) { return <ArticleLanguagePage locale="fr" slug={(await params).slug} />; }
