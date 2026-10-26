import { ArticleLanguagePage, articleLanguageMetadataFor, articleLanguageParams } from "@/components/article-language-route";

export const dynamicParams = true;
export const revalidate = 30;
type Props = { params: Promise<{ slug: string[] }> };
export async function generateStaticParams() { return articleLanguageParams("ko"); }
export async function generateMetadata({ params }: Props) { return articleLanguageMetadataFor("ko", (await params).slug); }
export default async function Page({ params }: Props) { return <ArticleLanguagePage locale="ko" slug={(await params).slug} />; }
