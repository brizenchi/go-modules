import "@fontsource-variable/manrope";
import "@fontsource-variable/newsreader";
import "../../globals.css";
import { ArticleLanguageLayout, articleLanguageMetadata } from "@/components/article-language-route";

export const revalidate = 30;
export const metadata = articleLanguageMetadata;
export default function Layout({ children }: { children: React.ReactNode }) {
  return <ArticleLanguageLayout locale="ja">{children}</ArticleLanguageLayout>;
}
