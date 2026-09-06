"use client";
import Image from "next/image";
import { useCallback, useEffect, useState, type FormEvent } from "react";
import { SiteShell } from "@/components/site-shell";
import { Panel } from "@/components/ui";
import { ConsoleError, ConsoleGate, Pagination, consoleStyles as styles, useConsoleAction, useConsoleResource, useConsoleSession } from "@/components/console-kit";
import { getPrivateImage, listUploadedImages, uploadImage, type UploadedImage } from "@/lib/operations-api";
import { useI18n } from "@/lib/i18n";
export default function FilesPage() {
  const { t } = useI18n();
  const { session, ready } = useConsoleSession();
  const token = session?.token || "";
  const [page, setPage] = useState(1);
  const loader = useCallback((key: string) => listUploadedImages(key, page), [page]);
  const files = useConsoleResource(token, loader);
  const action = useConsoleAction(token);
  const [file, setFile] = useState<File | null>(null);
  const [inputKey, setInputKey] = useState(0);
  const [preview, setPreview] = useState<{ owner: string; url: string; file: UploadedImage } | null>(null);
  useEffect(() => { setFile(null); setPreview(null); setPage(1); setInputKey((value) => value + 1); }, [token]);
  useEffect(() => () => { if (preview) URL.revokeObjectURL(preview.url); }, [preview]);
  function upload(event: FormEvent) {
    event.preventDefault(); if (!file) return;
    void action.run(() => uploadImage(token, file), () => { setFile(null); setInputKey((value) => value + 1); action.setMessage(t({ en: "Image uploaded to your private files.", zh: "图片已上传到你的私有文件。" })); if (page === 1) void files.refresh(); else setPage(1); });
  }
  function view(item: UploadedImage) { void action.run(() => getPrivateImage(token, item.id), (blob) => setPreview({ owner: token, url: URL.createObjectURL(blob), file: item })); }
  const invalid = file && (file.size > 5 * 1024 * 1024 || !["image/jpeg", "image/png", "image/webp", "image/gif"].includes(file.type));
  return <SiteShell eyebrow={t({ en: "Private files", zh: "我的文件" })} title={t({ en: "Your images, in one place.", zh: "上传图片，私有保存。" })} description={t({ en: "Upload images for your work. Files require your account to access and are never exposed as public links.", zh: "上传工作所需的图片，文件仅限当前账号访问。" })}>
    <ConsoleGate session={session} ready={ready}><div className={styles.stack}>
      <ConsoleError error={action.error} /><ConsoleError error={files.error} retry={() => void files.refresh()} />
      {action.message ? <p role="status" className={styles.success}>{action.message}</p> : null}
      <Panel title={t({ en: "Upload image", zh: "上传图片" })} subtitle={t({ en: "PNG, JPEG, WebP or GIF · Up to 5 MB per file.", zh: "支持 PNG、JPEG、WebP、GIF，每个文件最大 5 MB。" })}><form onSubmit={upload} className={styles.stack}><div className="field"><label htmlFor="private-image">{t({ en: "Choose an image", zh: "选择图片" })}</label><input key={inputKey} id="private-image" type="file" accept="image/png,image/jpeg,image/webp,image/gif" onChange={(event) => setFile(event.target.files?.[0] || null)} /></div>{invalid ? <p role="alert">{t({ en: "Choose a supported image up to 5 MB.", zh: "请选择支持的图片格式，大小不超过 5 MB。" })}</p> : null}<button className="button primary" disabled={!file || Boolean(invalid) || action.busy} type="submit">{action.busy ? t({ en: "Working…", zh: "处理中…" }) : t({ en: "Upload to my files", zh: "上传到我的文件" })}</button></form></Panel>
      {preview?.owner === token ? <Panel title={preview.file.filename}><Image unoptimized width={800} height={450} className={styles.preview} src={preview.url} alt={preview.file.filename} /><div className="button-row"><a className="button" href={preview.url} download={preview.file.filename}>{t({ en: "Download image", zh: "下载图片" })}</a><button className="button" type="button" onClick={() => setPreview(null)}>{t({ en: "Close preview", zh: "关闭预览" })}</button></div></Panel> : null}
      <Panel title={t({ en: "Uploaded images", zh: "已上传图片" })}>{files.loading ? <p role="status">{t({ en: "Loading files…", zh: "正在读取文件…" })}</p> : null}{files.data?.items.length ? <div className={styles.tableWrap}><table className="table"><thead><tr><th>{t({ en: "File", zh: "文件" })}</th><th>{t({ en: "Type", zh: "格式" })}</th><th>{t({ en: "Size", zh: "大小" })}</th><th>{t({ en: "Action", zh: "操作" })}</th></tr></thead><tbody>{files.data.items.map((item) => <tr key={item.id}><td>{item.filename}</td><td>{item.content_type}</td><td>{Math.ceil(item.size / 1024)} KB</td><td><button className="button" type="button" disabled={action.busy} onClick={() => view(item)}>{t({ en: "Open privately", zh: "查看图片" })}</button></td></tr>)}</tbody></table></div> : !files.loading && !files.error ? <p className="empty-state">{t({ en: "Your uploaded images will appear here.", zh: "上传的图片会显示在这里。" })}</p> : null}{files.data ? <Pagination result={files.data} loading={files.loading || action.busy} onPage={setPage} /> : null}</Panel>
    </div></ConsoleGate>
  </SiteShell>;
}
