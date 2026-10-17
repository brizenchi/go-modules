"use client";
import { useEffect, useRef } from "react";
import { createPageReturnRefresh } from "@/lib/page-return";

/** Revalidate after a longer absence, without requesting on every screen switch. */
export function usePageReturn(update: () => void) {
 const latestUpdate = useRef(update);
 useEffect(() => { latestUpdate.current = update; }, [update]);
 useEffect(() => {
  const refresh = createPageReturnRefresh();
  let timer: ReturnType<typeof setTimeout> | undefined;
  const onLeave = () => { clearTimeout(timer); refresh.leave(); };
  const onReturn = () => {
   if (document.visibilityState !== "visible") return;
   clearTimeout(timer);
   timer = setTimeout(() => {
    if (document.visibilityState === "visible" && refresh.shouldRefresh()) latestUpdate.current();
   }, 120);
  };
  const onVisibility = () => document.visibilityState === "visible" ? onReturn() : onLeave();
  if (document.visibilityState !== "visible") onLeave();
  window.addEventListener("blur", onLeave);
  window.addEventListener("focus", onReturn);
  document.addEventListener("visibilitychange", onVisibility);
  return () => {
   clearTimeout(timer);
   window.removeEventListener("blur", onLeave);
   window.removeEventListener("focus", onReturn);
   document.removeEventListener("visibilitychange", onVisibility);
  };
 }, []);
}
