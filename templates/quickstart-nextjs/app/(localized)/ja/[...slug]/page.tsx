import { ArticleLanguagePage, articleLanguageMetadataFor, articleLanguageParams } from "@/components/article-language-route";

export const dynamicParams = true;
export const revalidate = 30;
type Props = { params: Promise<{ slug: string[] }> };
export async function generateStaticParams() { return articleLanguageParams("ja"); }
export async function generateMetadata({ params }: Props) { return articleLanguageMetadataFor("ja", (await params).slug); }
export default async function Page({ params }: Props) { return <ArticleLanguagePage locale="ja" slug={(await params).slug} />; }
