import { ArticleLanguagePage, articleLanguageMetadataFor, articleLanguageParams } from "@/components/article-language-route";

export const dynamicParams = true;
export const revalidate = 30;
type Props = { params: Promise<{ slug: string[] }> };
export async function generateStaticParams() { return articleLanguageParams("es"); }
export async function generateMetadata({ params }: Props) { return articleLanguageMetadataFor("es", (await params).slug); }
export default async function Page({ params }: Props) { return <ArticleLanguagePage locale="es" slug={(await params).slug} />; }
