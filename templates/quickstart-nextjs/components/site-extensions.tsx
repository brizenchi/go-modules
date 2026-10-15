"use client";

import Script from "next/script";
import { usePathname } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { appEnv } from "@/lib/env";
import { analyticsConsent, analyticsMeasurementId, analyticsPageURL, crispChatURL } from "@/lib/site-extensions";
import { useI18n } from "@/lib/i18n";
import { ResponsiveDialog } from "./responsive-dialog";
import styles from "./site-extensions.module.css";

declare global {
  interface Window { dataLayer?: unknown[]; gtag?: (...args: unknown[]) => void; }
}
const CONSENT_KEY = "go-modules.analytics-consent";

export function SiteExtensions() {
  const { t } = useI18n();
  const path = usePathname();
  const measurementId = analyticsMeasurementId(appEnv.analyticsMeasurementId);
  const supportURL = crispChatURL(appEnv.crispWebsiteId);
  const [consent, setConsent] = useState<"allowed" | "declined" | null>(null);
  const [consentReady, setConsentReady] = useState(false);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [chatOpen, setChatOpen] = useState(false);
  const [loaded, setLoaded] = useState(false);
  const lastPage = useRef("");
  const pageURL = typeof window === "undefined" ? null : analyticsPageURL(window.location.origin,path);

  useEffect(() => {
    const sync = () => {
      try { setConsent(analyticsConsent(window.localStorage.getItem(CONSENT_KEY))); } catch { setConsent(null); }
      setConsentReady(true);
    };
    sync(); window.addEventListener("storage",sync);
    return () => window.removeEventListener("storage",sync);
  }, []);
  useEffect(() => { setChatOpen(false); }, [path]);
  const active = !!measurementId && consent === "allowed" && !!pageURL;
  const activeRef = useRef(active);
  activeRef.current = active;
  useEffect(() => {
    if (!measurementId) return;
    const target = window as unknown as Record<string,unknown>;
    target[`ga-disable-${measurementId}`] = !active;
    if (!active) { lastPage.current = ""; return; }
    if (loaded && window.gtag && pageURL && lastPage.current !== pageURL) {
      window.gtag("event","page_view",{ page_location:pageURL,page_title:document.title,page_referrer:"",send_to:measurementId });
      lastPage.current = pageURL;
    }
  }, [active,loaded,measurementId,pageURL]);

  function choose(value: "allowed" | "declined") {
    setConsent(value); setSettingsOpen(false);
    try { window.localStorage.setItem(CONSENT_KEY,value); } catch { /* Preference still applies to this view. */ }
    if (value === "declined" && loaded) window.location.reload();
  }
  function initializeAnalytics() {
    if (!activeRef.current) return;
    window.dataLayer ||= [];
    window.gtag ||= function(...args: unknown[]) { if (args.length) window.dataLayer?.push(arguments); };
    window.gtag("consent","default",{analytics_storage:"granted",ad_storage:"denied",ad_user_data:"denied",ad_personalization:"denied"});
    window.gtag("js",new Date());
    window.gtag("config",measurementId,{send_page_view:false,allow_google_signals:false,allow_ad_personalization_signals:false,page_location:pageURL,page_referrer:""});
    setLoaded(true);
  }
  const showAnalyticsChoice = !!measurementId && consentReady && !!pageURL && (!consent || settingsOpen);
  const showChat = !!supportURL && !/^\/(admin|oauth|login)(\/|$)/.test(path);

  return <>
    {active ? <Script id="optional-ga4" src={`https://www.googletagmanager.com/gtag/js?id=${measurementId}`} strategy="afterInteractive" onReady={initializeAnalytics} /> : null}
    {showAnalyticsChoice ? <section className={styles.choice} aria-label={t({en:"Usage analytics preference",zh:"使用统计偏好"})}>
      <strong>{t({en:"Optional usage analytics",zh:"可选使用统计"})}</strong>
      <p>{t({en:"Allow Google Analytics to measure visits to public pages. You can change this choice later.",zh:"允许 Google Analytics 统计公开页面的访问情况。你可以随时修改选择。"})}</p>
      <div><button type="button" className="button" onClick={()=>choose("declined")}>{t({en:"Decline",zh:"不允许"})}</button><button type="button" className="button primary" onClick={()=>choose("allowed")}>{t({en:"Allow analytics",zh:"允许统计"})}</button></div>
    </section> : measurementId && consentReady && pageURL ? <button type="button" className={styles.preferences} onClick={()=>setSettingsOpen(true)}>{t({en:"Analytics preferences",zh:"统计偏好"})}</button> : null}
    {showChat ? <><button className={styles.launcher} type="button" aria-haspopup="dialog" onClick={()=>setChatOpen(true)}>{t({en:"Chat with support",zh:"在线客服"})}</button>
      <ResponsiveDialog open={chatOpen} onClose={()=>setChatOpen(false)} title={t({en:"Support",zh:"在线客服"})} description={t({en:"This chat is provided by Crisp.",zh:"此聊天由 Crisp 提供。"})} closeLabel={t({en:"Close support",zh:"关闭客服"})}>
        <iframe className={styles.chat} src={supportURL} title={t({en:"Support conversation",zh:"客服会话"})} referrerPolicy="no-referrer" sandbox="allow-scripts allow-same-origin allow-forms allow-popups" />
        <a className={styles.fallback} href={supportURL} target="_blank" rel="noopener noreferrer">{t({en:"Open chat in a new window",zh:"在新窗口打开聊天"})} ↗</a>
      </ResponsiveDialog></> : null}
  </>;
}
