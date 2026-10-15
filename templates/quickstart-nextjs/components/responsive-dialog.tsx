"use client";

import { useEffect, useId, useRef, type ReactNode } from "react";
import { InterfaceIcon } from "@/components/interface-icon";
import { dialogFocusTarget } from "@/lib/dialog-focus";
import styles from "./responsive-dialog.module.css";

type ResponsiveDialogProps = {
  open: boolean;
  onClose: () => void;
  title: string;
  description?: string;
  closeLabel: string;
  icon?: ReactNode;
  children: ReactNode | ((dismiss: () => void) => ReactNode);
  footer?: ReactNode;
  presentation?: "sheet" | "side";
  dismissible?: boolean;
  size?: "default" | "wide";
};

export function ResponsiveDialog({ open, onClose, title, description, closeLabel, icon, children, footer, presentation = "sheet", dismissible = true, size = "default" }: ResponsiveDialogProps) {
  const dialog = useRef<HTMLDialogElement>(null);
  const titleId = useId();
  const descriptionId = useId();
  const startedOutside = useRef(false);

  useEffect(() => {
    if (!open) return;
    const element = dialog.current;
    if (!element) return;
    const previousFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    const previousOverflow = document.body.style.overflow;
    element.showModal();
    document.body.style.overflow = "hidden";
    return () => {
      element.close();
      document.body.style.overflow = previousOverflow;
      if (previousFocus?.isConnected && !document.querySelector("dialog[open]")) previousFocus.focus({ preventScroll: true });
    };
  }, [open]);

  function closeDialog() {
    dialog.current?.close();
    onClose();
  }
  function dismiss() {
    if (dismissible) closeDialog();
  }

  function outside(element: HTMLDialogElement, x: number, y: number) {
    const bounds = element.getBoundingClientRect();
    return x < bounds.left || x > bounds.right || y < bounds.top || y > bounds.bottom;
  }

  return <dialog ref={dialog} className={`${styles.dialog} ${presentation === "side" ? styles.side : styles.sheet} ${size === "wide" ? styles.wide : ""}`}
    aria-labelledby={titleId} aria-describedby={description ? descriptionId : undefined}
    onCancel={(event) => { event.preventDefault(); dismiss(); }}
    onPointerDown={(event) => { startedOutside.current = event.target === event.currentTarget && outside(event.currentTarget, event.clientX, event.clientY); }}
    onClick={(event) => {
      if (startedOutside.current && event.target === event.currentTarget && outside(event.currentTarget, event.clientX, event.clientY)) dismiss();
      startedOutside.current = false;
    }}
    onKeyDown={(event) => {
      if (event.key !== "Tab") return;
      const elements = Array.from(event.currentTarget.querySelectorAll<HTMLElement>('a[href], button:not(:disabled), input:not(:disabled):not([type="hidden"]), select:not(:disabled), textarea:not(:disabled), [tabindex]:not([tabindex="-1"])'))
        .filter(element => element.tabIndex >= 0 && element.getClientRects().length > 0 && !element.closest("[inert]"));
      const target = dialogFocusTarget(elements.indexOf(document.activeElement as HTMLElement), elements.length, event.shiftKey);
      if (target !== null) { event.preventDefault(); elements[target]?.focus(); }
    }}>
    <div className={styles.handle} aria-hidden="true" />
    <button className={styles.close} type="button" aria-label={closeLabel} disabled={!dismissible} onClick={dismiss}><InterfaceIcon name="close" /></button>
    <div className={styles.header}>
      {icon ? <span className={styles.icon}>{icon}</span> : null}
      <h2 id={titleId}>{title}</h2>
      {description ? <p id={descriptionId}>{description}</p> : null}
    </div>
    <div className={styles.body}>{open ? typeof children === "function" ? children(closeDialog) : children : null}</div>
    {open && footer ? <footer className={styles.footer}>{footer}</footer> : null}
  </dialog>;
}
