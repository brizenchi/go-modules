import { notFound } from "next/navigation";
import { BlogArticle } from "@/components/content/blog";
import { articles, findArticle } from "@/content/articles";
import { publicMetadata } from "@/lib/seo";

export const dynamicParams = false;
export function generateStaticParams() { return articles.map(({ slug }) => ({ slug })); }

type ArticlePageProps = { params: Promise<{ slug: string }> };

export async function generateMetadata({ params }: ArticlePageProps) {
  const { slug } = await params;
  const article = findArticle(slug);
  if (!article) notFound();
  return publicMetadata(article.title.en, article.summary.en, `/blog/${article.slug}`, { publishedAt: article.publishedAt });
}

export default async function ArticlePage({ params }: ArticlePageProps) {
  const { slug } = await params;
  const article = findArticle(slug);
  if (!article) notFound();
  return <BlogArticle article={article} />;
}
