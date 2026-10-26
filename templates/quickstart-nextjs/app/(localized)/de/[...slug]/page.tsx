import { ArticleLanguagePage, articleLanguageMetadataFor, articleLanguageParams } from "@/components/article-language-route";

export const dynamicParams = true;
export const revalidate = 30;
type Props = { params: Promise<{ slug: string[] }> };
export async function generateStaticParams() { return articleLanguageParams("de"); }
export async function generateMetadata({ params }: Props) { return articleLanguageMetadataFor("de", (await params).slug); }
export default async function Page({ params }: Props) { return <ArticleLanguagePage locale="de" slug={(await params).slug} />; }
