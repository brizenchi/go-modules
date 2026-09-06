"use client";
import Link from "next/link";
import { useEffect, useRef, useState, type FormEvent } from "react";
import { SiteShell } from "@/components/site-shell";
import { Panel } from "@/components/ui";
import { ConsoleError, ConsoleGate, consoleStyles as styles, downloadText, newIntentKey, useConsoleAction, useConsoleResource, useConsoleSession } from "@/components/console-kit";
import { createNote, exportNote, getCreditBalance, listNotes, type NoteExport } from "@/lib/operations-api";
import { getPublicSiteSettings, SITE_SETTINGS_EVENT } from "@/lib/site-settings";
import { useI18n } from "@/lib/i18n";
export default function NotesPage() {
  const { t } = useI18n();
  const { session, ready } = useConsoleSession();
  const token = session?.token || "";
  const notes = useConsoleResource(token, listNotes);
  const balance = useConsoleResource(token, getCreditBalance);
  const action = useConsoleAction(token);
  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");
  const [cost, setCost] = useState<number | null>(null);
  const [costError, setCostError] = useState<unknown>(null);
  const [priceRefresh, setPriceRefresh] = useState(0);
  const [exports, setExports] = useState<Record<number, NoteExport>>({});
  const [selected, setSelected] = useState<number | null>(null);
  const keys = useRef<Record<number, string>>({});
  useEffect(() => { setTitle(""); setBody(""); setExports({}); setSelected(null); keys.current = {}; }, [token]);
  useEffect(() => {
    const controller = new AbortController();
    let disposed = false;
    const timer = window.setTimeout(() => controller.abort(), 8000);
    void getPublicSiteSettings(controller.signal).then((settings) => { if (!disposed) { setCost(settings.export_credit_cost); setCostError(null); } }).catch((error) => { if (!disposed) { setCost(null); setCostError(error); } }).finally(() => window.clearTimeout(timer));
    const sync = () => setPriceRefresh((value) => value + 1);
    window.addEventListener(SITE_SETTINGS_EVENT, sync);
    return () => { disposed = true; controller.abort(); window.clearTimeout(timer); window.removeEventListener(SITE_SETTINGS_EVENT, sync); };
  }, [priceRefresh]);
  function save(event: FormEvent) {
    event.preventDefault();
    void action.run(() => createNote(token, title.trim(), body), () => { setTitle(""); setBody(""); action.setMessage(t({ en: "Note saved. You can export it when you're ready.", zh: "笔记已保存，可以随时导出。" })); void notes.refresh(); });
  }
  function confirmExport(event: FormEvent) {
    event.preventDefault(); if (selected === null || cost === null) return;
    const id = selected;
    const key = keys.current[id] || (keys.current[id] = newIntentKey());
    void action.run(() => exportNote(token, id, key, cost), (result) => { setExports((current) => ({ ...current, [id]: result })); setSelected(null); action.setMessage(t({ en: `Export ready. Remaining balance: ${result.balance} credits. Download it below.`, zh: `导出完成，剩余 ${result.balance} 积分。点击下方按钮下载。` })); void balance.refresh(); });
  }
  return <SiteShell eyebrow={t({ en: "Notes", zh: "我的笔记" })} title={t({ en: "Write freely. Export when ready.", zh: "记录想法，按需导出。" })} description={t({ en: "Create and save notes for free. Export a Markdown file using credits, with every purchase and use visible in your credit history.", zh: "免费创建与保存笔记，使用积分导出 Markdown 文件。获得和消费的积分都能在明细中查看。" })} actions={<Link className="button" href="/credits">{t({ en: "Balance and credit history", zh: "查看积分与明细" })}</Link>}>
    <ConsoleGate session={session} ready={ready}><div className={styles.stack}>
      <ConsoleError error={action.error} retry={() => { setPriceRefresh((value) => value + 1); void balance.refresh(); }} /><ConsoleError error={notes.error} retry={() => void notes.refresh()} /><ConsoleError error={costError} retry={() => setPriceRefresh((value) => value + 1)} />
      {action.message ? <p role="status" className={styles.success}>{action.message}</p> : null}
      <div className={styles.editor}>
        <Panel title={t({ en: "New note", zh: "新建笔记" })}><form onSubmit={save} className={styles.stack}><div className="field"><label htmlFor="note-title">{t({ en: "Title", zh: "标题" })}</label><input id="note-title" required maxLength={200} value={title} onChange={(event) => setTitle(event.target.value)} /></div><div className="field"><label htmlFor="note-body">{t({ en: "Content", zh: "内容" })}</label><textarea id="note-body" rows={10} maxLength={50000} value={body} onChange={(event) => setBody(event.target.value)} /></div><button className="button primary" disabled={action.busy || !title.trim()} type="submit">{t({ en: "Save note · Free", zh: "保存笔记 · 免费" })}</button></form></Panel>
        <Panel title={t({ en: "Export with credits", zh: "用积分导出" })}><div className={styles.stack}><p>{cost !== null ? t({ en: `Each new export costs ${cost} credits. A retry of the same export does not charge again.`, zh: `每次新导出消耗 ${cost} 积分，同一次导出重试不会重复扣费。` }) : t({ en: "Export pricing is currently unavailable.", zh: "暂时无法读取导出价格。" })}</p><p>{t({ en: "Available balance", zh: "可用积分" })}: <strong>{balance.data?.balance ?? "—"}</strong></p><ConsoleError error={balance.error} retry={() => void balance.refresh()} /><Link className="button" href="/billing">{t({ en: "Get more credits", zh: "购买积分" })}</Link><p className="muted">{t({ en: "Only you can read or export your notes. A ready export can be downloaded again on this page without another charge.", zh: "只有你能读取或导出自己的笔记。在当前页面重复下载已生成文件无需再次扣费。" })}</p></div></Panel>
      </div>
      {selected !== null ? <Panel title={t({ en: "Confirm this export", zh: "确认导出" })}><form onSubmit={confirmExport}><p>{t({ en: `Use ${cost} credits to export this note as Markdown?`, zh: `本次将使用 ${cost} 积分，将笔记导出为 Markdown 文件。` })}</p><div className="button-row"><button className="button primary" type="submit" disabled={action.busy || cost === null}>{t({ en: "Confirm and generate", zh: "确认并生成文件" })}</button><button className="button" type="button" disabled={action.busy} onClick={() => setSelected(null)}>{t({ en: "Cancel", zh: "取消" })}</button></div></form></Panel> : null}
      <Panel title={t({ en: "Saved notes", zh: "已保存的笔记" })} subtitle={t({ en: "Your 50 most recent notes.", zh: "显示最近保存的 50 条笔记。" })}>{notes.loading ? <p role="status">{t({ en: "Loading notes…", zh: "正在加载笔记…" })}</p> : null}{notes.data?.map((note) => <article className={styles.note} key={note.id}><h3>{note.title}</h3><p>{note.body.slice(0, 240)}{note.body.length > 240 ? "…" : ""}</p>{exports[note.id] ? <button className="button primary" type="button" onClick={() => downloadText(exports[note.id].filename, exports[note.id].content)}>{t({ en: "Download Markdown", zh: "下载 Markdown" })}</button> : <button className="button" type="button" disabled={action.busy || cost === null || (!keys.current[note.id] && (!balance.data || balance.data.balance < cost))} onClick={() => { setSelected(note.id); }}>{keys.current[note.id] ? t({ en: "Retry this export", zh: "重试本次导出" }) : t({ en: "Export with credits", zh: "使用积分导出" })}</button>}</article>)}{notes.data?.length === 0 ? <p className="empty-state">{t({ en: "Save your first note to get started.", zh: "保存第一条笔记，开始体验。" })}</p> : null}</Panel>
    </div></ConsoleGate>
  </SiteShell>;
}
