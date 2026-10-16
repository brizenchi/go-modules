"use client";

import { useState } from "react";
import { logoImageURL } from "@/lib/site-branding";
import { useSitePresentation } from "./site-presentation";
import styles from "./brand-mark.module.css";

/** One brand mark for public pages, customer accounts and administration. */
export function BrandMark({ className = "", imageSource }: { className?: string; imageSource?: string }) {
  const site = useSitePresentation();
  const src = imageSource ?? logoImageURL(site?.logo_url);
  const [failedSource, setFailedSource] = useState("");
  const custom = Boolean(src && src !== failedSource);
  return <span className={`${className}${custom ? ` ${styles.custom}` : ""}`} aria-hidden="true">
    {custom ? (
      // Site-managed assets can use any storage/CDN; preserve their original URL.
      // eslint-disable-next-line @next/next/no-img-element
      <img className={styles.image} src={src} alt="" width={48} height={48} onError={() => setFailedSource(src)} />
    ) : <svg viewBox="0 0 28 28" fill="none" stroke="currentColor" strokeWidth="1.5"><path d="M6 8.5 14 4l8 4.5v9L14 22l-8-4.5v-9Z" /><path d="m9.5 10.5 4.5-2.6 4.5 2.6v5L14 18l-4.5-2.5v-5Z" /></svg>}
  </span>;
}
