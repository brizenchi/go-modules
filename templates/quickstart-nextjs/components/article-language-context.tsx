"use client";
import { createContext, useContext } from "react";
import type { ContentLocale } from "@/lib/locale";
const ArticleLanguages = createContext<ContentLocale[] | null>(null);
export function ArticleLanguagesProvider({ locales, children }: { locales: ContentLocale[]; children: React.ReactNode }) { return <ArticleLanguages.Provider value={locales}>{children}</ArticleLanguages.Provider>; }
export function useArticleLanguages() { return useContext(ArticleLanguages); }
