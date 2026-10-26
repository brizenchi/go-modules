"use client";
import { useRef, useState } from "react";
import type { ContentImportResult } from "@/lib/api";
import { importContent } from "@/lib/content-api";
import { useI18n } from "@/lib/i18n";
import { ConsoleError, newIntentKey, useConsoleAction } from "./console-kit";
import { ResponsiveDialog } from "./responsive-dialog";
import { PendingLabel } from "./pending-label";
import styles from "./admin-shell.module.css";

export function ContentImportDialog({ token, onImported }: { token: string; onImported: () => void }) {
 const { t } = useI18n(); const action = useConsoleAction(token);
 const [open, setOpen] = useState(false); const [plan, setPlan] = useState<ContentImportResult | null>(null);
 const intent = useRef({ payload: "", key: "" });
 function preview() { setOpen(true); setPlan(null); intent.current = { payload: "", key: "" }; void action.run(() => importContent(token), setPlan); }
 function apply() {
  // The audit log records the action itself; the key makes a repeated click a replay.
  const payload = t({ en: "Import starter content", zh: "导入初始内容" });
  if (intent.current.payload !== payload) intent.current = { payload, key: newIntentKey() };
  void action.run(() => importContent(token, true, payload, intent.current.key), result => { setPlan(result); onImported(); });
 }
 return <><button type="button" className="button" disabled={!token || action.busy} onClick={preview}>{t({ en: "Initialize content", zh: "初始化内容" })}</button>
  <ResponsiveDialog open={open} onClose={() => setOpen(false)} dismissible={!action.busy} title={t({ en: "Review starter content", zh: "预览初始内容" })} closeLabel={t({ en: "Close", zh: "关闭" })}
   description={t({ en: "Create missing pages and articles with EN and Chinese publications. Existing paths are skipped, including drafts and unpublished content.", zh: "为缺失的页面与文章创建中英文发布记录。已有路径，包括草稿和下架内容，都会跳过。" })}>
   <ConsoleError error={action.error} />
   {plan ? <div className={styles.editorFields}><p role="status">{t(plan.applied ? { en: `Created ${plan.created} entries. Skipped ${plan.skipped} existing paths.`, zh: `已创建 ${plan.created} 条，跳过 ${plan.skipped} 条已有路径。` } : { en: `${plan.created} to create · ${plan.skipped} to skip. Preview has not written any data.`, zh: `将创建 ${plan.created} 条，跳过 ${plan.skipped} 条。当前预览未写入数据。` })}</p>
    <ul className={styles.importItems}>{plan.items.map(item => <li key={item.path}><span><strong>{item.title}</strong><code>{item.path}</code></span><small>{t(item.status === "skip" ? { en: "Skip", zh: "跳过" } : plan.applied ? { en: "Created", zh: "已创建" } : { en: "Create", zh: "创建" })}</small></li>)}</ul>
    {!plan.applied && plan.created > 0 ? <><button type="button" className="button primary" disabled={action.busy} onClick={apply}>{action.busy ? <PendingLabel>{t({ en: "Importing…", zh: "正在导入…" })}</PendingLabel> : t({ en: "Import and publish", zh: "导入并发布" })}</button></> : null}
   </div> : <p role="status">{t({ en: "Loading preview…", zh: "正在读取预览…" })}</p>}
  </ResponsiveDialog>
 </>;
}
