"use client";

import { forwardRef, type SelectHTMLAttributes } from "react";

/** Keep native keyboard/option behavior with one shared visual treatment. */
export const Select = forwardRef<HTMLSelectElement, Omit<SelectHTMLAttributes<HTMLSelectElement>, "multiple" | "size">>(
  function Select({ className = "", children, ...props }, ref) {
    return <span className="select-control">
      <select {...props} ref={ref} className={`select-input ${className}`.trim()}>{children}</select>
      <svg viewBox="0 0 20 20" aria-hidden="true" focusable="false"><path d="m6 8 4 4 4-4" /></svg>
    </span>;
  }
);
