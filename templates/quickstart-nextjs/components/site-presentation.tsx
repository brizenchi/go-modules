"use client";
import { createContext, useContext, useEffect, useState } from "react";
import { apiRequest, type WebsiteSettings } from "@/lib/api";
import { SITE_SETTINGS_EVENT } from "@/lib/site-settings";
const SitePresentationContext = createContext<WebsiteSettings | null>(null);
export function SitePresentationProvider({ value, children }: { value: WebsiteSettings | null; children: React.ReactNode }) {
 const [site, setSite] = useState(value);
 useEffect(() => { setSite(value); }, [value]);
 useEffect(() => { const refresh = () => { void apiRequest<WebsiteSettings>("/site/settings").then(next => { if (next.presentation_version === 1) setSite(next); }).catch(() => {}); }; window.addEventListener(SITE_SETTINGS_EVENT, refresh); return () => window.removeEventListener(SITE_SETTINGS_EVENT, refresh); }, []);
 return <SitePresentationContext.Provider value={site}>{children}</SitePresentationContext.Provider>;
}
export function useSitePresentation() { return useContext(SitePresentationContext); }
