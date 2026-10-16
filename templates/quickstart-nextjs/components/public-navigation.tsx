"use client";

import { useEffect, useId, useRef, useState } from "react";
import { LocalizedLink as Link } from "./localized-link";
import { activePublicNavigationHref, publicNavigation, publicNavigationGroups } from "@/content/navigation";
import { disclosureFocusIndex } from "@/lib/disclosure-focus";
import { useI18n } from "@/lib/i18n";
import { InterfaceIcon } from "./interface-icon";
import styles from "./public-navigation.module.css";

export function PublicNavigation({ pathname }: { pathname: string }) {
  const { t, locale } = useI18n();
  const [open, setOpen] = useState("");
  const nav = useRef<HTMLElement>(null);
  const trigger = useRef<HTMLButtonElement | null>(null);
  const id = useId();
  const active = activePublicNavigationHref(pathname);
  useEffect(() => { setOpen(""); }, [pathname, locale]);
  useEffect(() => {
    const outside = (event: PointerEvent) => {
      if (event.target instanceof Node && !nav.current?.contains(event.target)) setOpen("");
    };
    document.addEventListener("pointerdown", outside);
    return () => document.removeEventListener("pointerdown", outside);
  }, []);
  return <nav ref={nav} className={`main-nav ${styles.navigation}`} aria-label={t({ en: "Primary", zh: "主导航" })}
    onBlur={event => { if (!event.currentTarget.contains(event.relatedTarget)) setOpen(""); }}
    onKeyDown={event => { if (event.key === "Escape" && open) { event.preventDefault(); setOpen(""); trigger.current?.focus(); } }}>
    {publicNavigation.map(item => {
      const children = publicNavigationGroups[item.href];
      const panel = `${id}-${item.href.slice(1)}`;
      const link = <Link href={item.href} className={`main-nav-link${active === item.href ? " active" : ""}`}
        aria-current={pathname === item.href ? "page" : active === item.href ? "true" : undefined} onClick={() => setOpen("")}>{t(item.label)}</Link>;
      if (!children) return <span key={item.href}>{link}</span>;
      return <div key={item.href} className={styles.group}>
        {link}
        <button className={styles.toggle} type="button" aria-expanded={open === item.href} aria-controls={panel}
          aria-label={t({ en: `${item.label.en} menu`, zh: `${item.label.zh}菜单` })}
          onClick={event => { trigger.current = event.currentTarget; setOpen(value => value === item.href ? "" : item.href); }}
          onKeyDown={event => {
            if (event.key !== "ArrowDown" && event.key !== "ArrowUp") return;
            event.preventDefault(); trigger.current = event.currentTarget; setOpen(item.href);
            const last = event.key === "ArrowUp";
            requestAnimationFrame(() => {
              const links = document.getElementById(panel)?.querySelectorAll<HTMLAnchorElement>("a");
              links?.[last ? links.length - 1 : 0]?.focus();
            });
          }}><InterfaceIcon name="chevron" /></button>
        <div className={styles.popover} id={panel} hidden={open !== item.href}
          onKeyDown={event => {
            const links = Array.from(event.currentTarget.querySelectorAll<HTMLAnchorElement>("a"));
            const index = disclosureFocusIndex(event.key, links.indexOf(document.activeElement as HTMLAnchorElement), links.length);
            if (index !== null) { event.preventDefault(); links[index]?.focus(); }
          }}>
          {children.map(child => <Link key={child.href} href={child.href} aria-current={pathname === child.href ? "page" : undefined} onClick={() => setOpen("")}>{t(child.label)}<InterfaceIcon name="arrow" /></Link>)}
        </div>
      </div>;
    })}
  </nav>;
}
