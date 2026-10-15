"use client";
import { createContext, useContext } from "react";
import type { Locale } from "@/lib/locale";
const ArticleLanguages = createContext<Locale[] | null>(null);
export function ArticleLanguagesProvider({ locales, children }: { locales: Locale[]; children: React.ReactNode }) { return <ArticleLanguages.Provider value={locales}>{children}</ArticleLanguages.Provider>; }
export function useArticleLanguages() { return useContext(ArticleLanguages); }
