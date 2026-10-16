"use client";
import type { PagePresentation } from "@/lib/page-presentation";
import { useI18n } from "@/lib/i18n";
import { SiteShell } from "./site-shell";
import { MarkdownContent } from "./markdown-content";
import { ArticleLanguagesProvider } from "./article-language-context";
export function ManagedPage({ page }: { page: PagePresentation }) {
 const { locale } = useI18n();
 return <ArticleLanguagesProvider locales={page.locales}><SiteShell pagePath={page.path} eyebrow={page.translation.title} variant="article" title={page.translation.title} description={page.translation.summary}><div className="published-article"><MarkdownContent source={page.translation.body} locale={locale} /></div></SiteShell></ArticleLanguagesProvider>;
}
