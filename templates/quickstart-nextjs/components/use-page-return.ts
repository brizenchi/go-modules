"use client";
import { useEffect } from "react";

/** Refresh on return without a permanent toolbar control. Browser focus and
 * visibility can arrive together, so coalesce them into one read. */
export function usePageReturn(update: () => void) {
 useEffect(() => {
  let timer: ReturnType<typeof setTimeout> | undefined;
  const onReturn = () => {
   if (document.visibilityState !== "visible") return;
   clearTimeout(timer);
   timer = setTimeout(update, 120);
  };
  window.addEventListener("focus", onReturn);
  document.addEventListener("visibilitychange", onReturn);
  return () => {
   clearTimeout(timer);
   window.removeEventListener("focus", onReturn);
   document.removeEventListener("visibilitychange", onReturn);
  };
 }, [update]);
}
