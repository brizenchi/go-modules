"use client";

import { useEffect, useState, type ChangeEvent } from "react";
import { useI18n } from "@/lib/i18n";
import { uploadImage } from "@/lib/operations-api";
import { logoFileError, logoImageURL, safeLogoURL } from "@/lib/site-branding";
import { BrandMark } from "./brand-mark";
import { ConsoleError, useConsoleAction } from "./console-kit";
import { PendingLabel } from "./pending-label";
import styles from "./site-logo-field.module.css";

export function SiteLogoField({ token, value, brand, disabled, onChange, onBusyChange }: {
  token: string; value: string; brand: string; disabled: boolean;
  onChange: (value: string) => void; onBusyChange: (busy: boolean) => void;
}) {
  const { t } = useI18n();
  const action = useConsoleAction(token);
  const [preview, setPreview] = useState<{ value: string; url: string } | null>(null);
  const [fileError, setFileError] = useState<"type" | "size" | null>(null);
  useEffect(() => { onBusyChange(action.busy); }, [action.busy, onBusyChange]);
  useEffect(() => () => { if (preview) URL.revokeObjectURL(preview.url); }, [preview]);

  function upload(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    event.target.value = "";
    if (!file) return;
    const invalid = logoFileError(file);
    setFileError(invalid);
    if (invalid) return;
    void action.run(() => uploadImage(token, file), image => {
      const next = `/api/v1/site/logo/${image.id}`;
      setPreview({ value: next, url: URL.createObjectURL(file) });
      onChange(next);
    });
  }

  const invalidURL = Boolean(value.trim() && !safeLogoURL(value));
  return <div className={styles.field}>
    <span className={styles.label}>{t({ en: "Website logo", zh: "网站 Logo" })}</span>
    <div className={styles.row}>
      <div className={styles.preview} aria-label={t({ en: "Logo preview", zh: "Logo 预览" })}>
        <BrandMark className={styles.mark} imageSource={preview?.value === value ? preview.url : logoImageURL(value)} />
        <strong>{brand}</strong>
      </div>
      <div className={styles.actions}>
        <label className={`button ${styles.upload}`}>
          <input id="website-logo-file" type="file" accept="image/png,image/jpeg,image/webp,image/gif" disabled={disabled || action.busy} onChange={upload} aria-label={t({ en: "Upload website logo", zh: "上传网站 Logo" })} />
          {action.busy ? <PendingLabel>{t({ en: "Uploading…", zh: "上传中…" })}</PendingLabel> : t({ en: "Upload logo", zh: "上传 Logo" })}
        </label>
        {value ? <button className="button" type="button" disabled={disabled || action.busy} onClick={() => { onChange(""); setPreview(null); setFileError(null); }}>{t({ en: "Use default", zh: "恢复默认" })}</button> : null}
      </div>
    </div>
    <p className={styles.hint}>{t({ en: "PNG, JPG, WebP or GIF, up to 5 MB. Save settings to apply.", zh: "支持 PNG、JPG、WebP、GIF，最大 5 MB。保存设置后生效。" })}</p>
    {fileError ? <p role="alert" className={styles.error}>{t(fileError === "type" ? { en: "Choose a PNG, JPG, WebP or GIF image.", zh: "请选择 PNG、JPG、WebP 或 GIF 图片。" } : { en: "Choose an image between 1 byte and 5 MB.", zh: "图片不能为空，且不能超过 5 MB。" })}</p> : null}
    <ConsoleError error={action.error} />
    <details className={styles.hosted}><summary>{t({ en: "Use an image URL", zh: "使用图片地址" })}</summary><div className="field">
      <label htmlFor="website-logo-url">{t({ en: "Logo URL", zh: "Logo 地址" })}</label>
      <input id="website-logo-url" type="text" inputMode="url" maxLength={1024} placeholder="https://cdn.example.com/logo.png" value={value} onChange={event => onChange(event.target.value)} disabled={disabled || action.busy} aria-invalid={invalidURL || undefined} aria-describedby="website-logo-help" />
      <small id="website-logo-help">{t({ en: "Use an HTTPS image URL or a public path such as /logo.svg.", zh: "填写 HTTPS 图片地址，或 /logo.svg 这样的公开文件路径。" })}</small>
      {invalidURL ? <span role="alert" className={styles.error}>{t({ en: "Enter a valid HTTPS URL or public file path.", zh: "请填写有效的 HTTPS 地址或公开文件路径。" })}</span> : null}
    </div></details>
  </div>;
}
