"use client";
import type { PagePresentation } from "@/lib/page-presentation";
import { useI18n } from "@/lib/i18n";
import { SiteShell } from "./site-shell";
import { MarkdownContent } from "./markdown-content";
import { ArticleLanguagesProvider } from "./article-language-context";
import { useLinkablePath } from "./navigation-index";
import type { DiscoveryEntry } from "@/lib/content-discovery";
export function ManagedPage({ page, entries }: { page: PagePresentation; entries: readonly Pick<DiscoveryEntry, "path">[] }) {
 const { locale } = useI18n();
 const linkable = useLinkablePath(entries);
 return <ArticleLanguagesProvider locales={page.locales}><SiteShell pagePath={page.path} eyebrow={page.translation.title} variant="article" title={page.translation.title} description={page.translation.summary}><div className="published-article"><MarkdownContent source={page.translation.body} locale={locale} linkable={linkable} /></div></SiteShell></ArticleLanguagesProvider>;
}
