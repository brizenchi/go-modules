"use client";

import { cloneElement, useId, type ReactElement, type ReactNode } from "react";
import { PendingLabel } from "./pending-label";

type FieldControl = { id?: string; "aria-describedby"?: string; "aria-invalid"?: boolean; required?: boolean };
export function FormField({ label, description, error, required, children }: {
  label: string; description?: ReactNode; error?: string; required?: boolean; children: ReactElement<FieldControl>;
}) {
  const generated = useId();
  const id = children.props.id || generated;
  const describedBy = [children.props["aria-describedby"], description ? `${id}-help` : "", error ? `${id}-error` : ""].filter(Boolean).join(" ");
  return <div className="form-field">
    <label htmlFor={id}>{label}{required ? <span aria-hidden="true" className="form-required"> *</span> : null}</label>
    {cloneElement(children, { id, required: required || children.props.required, "aria-invalid": error ? true : children.props["aria-invalid"], "aria-describedby": describedBy || undefined })}
    {description ? <div className="form-description" id={`${id}-help`}>{description}</div> : null}
    {error ? <p className="form-error" id={`${id}-error`} role="alert">{error}</p> : null}
  </div>;
}
export function FormActions({ busy, disabled, label, busyLabel, children }: {
  busy: boolean; disabled?: boolean; label: string; busyLabel: string; children?: ReactNode;
}) {
  return <div className="form-actions"><button type="submit" className="button primary" disabled={busy || disabled} aria-busy={busy}>{busy ? <PendingLabel>{busyLabel}</PendingLabel> : label}</button>{children}</div>;
}
