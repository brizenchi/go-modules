"use client";

import Link from "next/link";
import { useCallback, useEffect, useRef, useState, type FormEvent } from "react";
import { useI18n } from "@/lib/i18n";
import { draftArticle, listContent, moveContent, readContent, saveContent, publishContent, translateArticle, uploadSiteImage } from "@/lib/content-api";
import { importMarkdown } from "@/lib/markdown";
import type { ContentArticle, ContentArticleState, ContentKind, ContentListFilters, ContentTranslation } from "@/lib/api";
import { articleState, listedLanguages, moveAvailability, noContentFilters, reorderable, withLanguage } from "@/lib/content-filters";
import { canPublish, defaultAlt, validArticlePath, insertImageAtCursor, languageNames, mapLimit, publishOptions, translationTargets, writingLanguages, writingPlan, type WritingLanguage } from "@/lib/article-writing";
import { interfaceLocale, languageTags } from "@/lib/locale";

import { ConsoleError, ConsoleGate, newIntentKey, useConsoleAction, useConsoleResource, useConsoleSession } from "./console-kit";
import { AdminPageHeader } from "./admin-page-header";
import { DataTable } from "./data-table";
import { MarkdownContent } from "./markdown-content";
import { StatusBadge } from "./status-badge";
import { PendingLabel } from "./pending-label";
import styles from "./admin-shell.module.css";
import { ContentImportDialog } from "./content-import-dialog";
import { Select } from "./select";
import { ResponsiveDialog } from "./responsive-dialog";

const emptyTranslation = (): ContentTranslation => ({ title: "", summary: "", body: "", seo_title: "", seo_description: "" });
const newArticle = (): ContentArticle => ({ id: "", kind: "features", path: "/features/", position: 0, version: 0, translations: {}, published: {}, updated_at: "" });
const kinds: ContentKind[] = ["page", "blog", "docs", "features", "use-cases", "compare"];
/** New articles go into the product sections; existing pages, guides and posts stay editable. */
const newKinds: ContentKind[] = ["features", "use-cases", "compare"];
const imageTypes = ["image/png", "image/jpeg", "image/gif", "image/webp"];
const kindLabels = { page: { en: "Pages & introductions", zh: "页面与栏目介绍" }, blog: { en: "Articles", zh: "产品文章" }, docs: { en: "Guides", zh: "使用指南" }, features: { en: "Features", zh: "功能详情" }, "use-cases": { en: "Use cases", zh: "使用场景" }, compare: { en: "Comparisons", zh: "方案比较" } };
const stateBadges: Record<ContentArticleState, { tone: "success" | "warning" | "neutral"; label: { en: string; zh: string } }> = { published: { tone: "success", label: { en: "Published", zh: "已发布" } }, changed: { tone: "warning", label: { en: "Edits unpublished", zh: "修改未发布" } }, draft: { tone: "neutral", label: { en: "Draft", zh: "草稿" } }, missing: { tone: "neutral", label: { en: "Not written", zh: "未撰写" } } };
const statusFilters: Array<{ value: ContentArticleState; label: { en: string; zh: string } }> = [{ value: "published", label: { en: "Published", zh: "已发布" } }, { value: "changed", label: { en: "Published, edits unpublished", zh: "已发布但修改未发布" } }, { value: "draft", label: { en: "Unpublished draft", zh: "有未发布的草稿" } }, { value: "missing", label: { en: "Not written in this language", zh: "缺少该语言" } }];
/** What each state colour means, shown next to the coloured languages. */
function StateLegend({ missing = false }: { missing?: boolean }) {
 const { t } = useI18n(); const states: ContentArticleState[] = missing ? ["published", "changed", "draft", "missing"] : ["published", "changed", "draft"];
 return <span className={styles.stateLegend}>{states.map(state => <span key={state}><i className={styles.stateDot} data-state={state} aria-hidden="true" />{t(stateBadges[state].label)}</span>)}</span>;
}
export function ContentAdminList() {
 const { t, locale } = useI18n(); const { session, ready } = useConsoleSession(); const token = session?.user.role === "admin" ? session.token : "";
 const [page, setPage] = useState(1); const [search, setSearch] = useState(""); const [filters, setFilters] = useState<ContentListFilters>(noContentFilters);
 const loader = useCallback((current: string) => listContent(current, page, filters), [page, filters]); const resource = useConsoleResource(token, loader);
 const filter = (next: ContentListFilters) => { setPage(1); setFilters(next); };
 const filtered = Object.values(filters).some(Boolean);
 const action = useConsoleAction(token); const rows = resource.data?.items || [];
 function move(row: ContentArticle, direction: "up" | "down") {
  void action.run(() => moveContent(token, row.id, direction, t(direction === "up" ? { en: "Move up in its section", zh: "排序上移一位" } : { en: "Move down in its section", zh: "排序下移一位" }), newIntentKey()), () => void resource.refresh());
 }
 return <><AdminPageHeader section="articles" onRefresh={() => void resource.refresh()} busy={resource.loading} actions={<><ContentImportDialog token={token} onImported={() => void resource.refresh()} /><Link className="button primary" href="/admin/articles/new">{t({ en: "New content", zh: "新建内容" })}</Link></>} /><ConsoleGate session={session} ready={ready} admin><ConsoleError error={resource.error} retry={() => void resource.refresh()} /><ConsoleError error={action.error} /><section className={styles.records}>
  <div className={styles.listControls}><form className={styles.toolbar} role="search" onSubmit={event => { event.preventDefault(); filter({ ...filters, query: search.trim() }); }}><div className={styles.search}><div className="field"><label className="sr-only" htmlFor="content-query">{t({ en: "Title or article path", zh: "标题或文章路径" })}</label><input id="content-query" type="search" placeholder={t({ en: "Search title or path", zh: "搜索标题或路径" })} value={search} onChange={event => setSearch(event.target.value)} /></div><button className="button" type="submit">{t({ en: "Search", zh: "搜索" })}</button></div>
   <div className="field"><label className="sr-only" htmlFor="content-kind">{t({ en: "Section", zh: "所属栏目" })}</label><Select id="content-kind" value={filters.kind} onChange={event => filter({ ...filters, kind: event.target.value as ContentListFilters["kind"] })}><option value="">{t({ en: "All sections", zh: "全部栏目" })}</option>{kinds.map(kind => <option key={kind} value={kind}>{t(kindLabels[kind])}</option>)}</Select></div>
   <div className="field"><label className="sr-only" htmlFor="content-language">{t({ en: "Language", zh: "语言" })}</label><Select id="content-language" value={filters.locale} onChange={event => filter(withLanguage(filters, event.target.value as ContentListFilters["locale"]))}><option value="">{t({ en: "All languages", zh: "全部语言" })}</option>{writingLanguages.map(language => <option key={language} value={language}>{t(languageNames[language])}</option>)}</Select></div>
   <div className="field"><label className="sr-only" htmlFor="content-status">{t({ en: "Publication status", zh: "发布状态" })}</label><Select id="content-status" value={filters.status} onChange={event => filter({ ...filters, status: event.target.value as ContentListFilters["status"] })}><option value="">{t({ en: "Any status", zh: "全部状态" })}</option>{statusFilters.filter(item => item.value !== "missing" || filters.locale).map(item => <option key={item.value} value={item.value}>{t(item.label)}</option>)}</Select></div>
   {filtered ? <button className={`button ${styles.resetFilter}`} type="button" onClick={() => { setSearch(""); filter(noContentFilters); }}>{t({ en: "Clear filters", zh: "清除筛选" })}</button> : null}
   <span className={styles.recordCount}>{resource.data ? t({ en: `${resource.data.total} entries`, zh: `共 ${resource.data.total} 条` }) : "—"}</span></form><StateLegend /></div>
  <DataTable caption={t({ en: "Content library", zh: "文章列表" })} columns={[
   { key: "title", heading: t({ en: "Article", zh: "文章" }), cell: (row: ContentArticle) => <span className={styles.cellIdentity}><span>{(filters.locale && row.translations[filters.locale]?.title) || row.translations[locale]?.title || row.translations.en?.title || row.translations.zh?.title || t({ en: "Untitled", zh: "未命名文章" })}</span><small>{row.path}</small></span> },
   { key: "kind", heading: t({ en: "Section", zh: "所属栏目" }), cell: row => t(kindLabels[row.kind]) },
   { key: "languages", heading: t({ en: "Languages", zh: "语言" }), cell: row => <div className={styles.rowActions}>{listedLanguages(row).map(language => { const badge = stateBadges[articleState(row, language)]; return <StatusBadge key={language} tone={badge.tone}>{t(languageNames[language])}<span className="sr-only">{`: ${t(badge.label)}`}</span></StatusBadge>; })}</div> },
   ...(reorderable(filters) ? [{ key: "order", heading: t({ en: "Order", zh: "排序" }), cell: (row: ContentArticle) => { const index = rows.indexOf(row); const can = moveAvailability(rows, index, resource.data?.page ?? 1, resource.data?.total ?? rows.length, resource.data?.limit ?? 20); return row.kind === "page" ? null : <div className={styles.moveButtons}><button className="button" type="button" disabled={action.busy || !can.up} onClick={() => move(row, "up")} aria-label={t({ en: "Move up", zh: "上移" })} title={t({ en: "Move up", zh: "上移" })}>↑</button><button className="button" type="button" disabled={action.busy || !can.down} onClick={() => move(row, "down")} aria-label={t({ en: "Move down", zh: "下移" })} title={t({ en: "Move down", zh: "下移" })}>↓</button></div>; } }] : []),
   { key: "actions", heading: t({ en: "Action", zh: "操作" }), cell: row => <Link className="button" href={`/admin/articles/${row.id}`}>{t({ en: "Edit", zh: "编辑" })}</Link> }
  ]} rows={rows} rowKey={row => row.id} loading={resource.loading} loadingLabel={t({ en: "Loading articles…", zh: "正在读取文章…" })} empty={t(filtered ? { en: "No articles match these filters.", zh: "没有符合筛选条件的文章。" } : { en: "No articles yet.", zh: "暂无文章。" })} pagination={resource.data ? { ...resource.data, onPage: setPage, previousLabel: t({ en: "Previous", zh: "上一页" }), nextLabel: t({ en: "Next", zh: "下一页" }), label: (p, n) => `${p} / ${n}` } : undefined} />
 </section></ConsoleGate></>;
}
export function ContentAdminEditor({ id: routeId }: { id: string }) {
 const { t, locale } = useI18n(); const { session, ready } = useConsoleSession(); const token = session?.user.role === "admin" ? session.token : "";
 const [article, setArticle] = useState<ContentArticle | null>(routeId === "new" ? newArticle() : null); const [language, setLanguage] = useState<WritingLanguage>(locale); const [preview, setPreview] = useState(false); const [fileError, setFileError] = useState(""); const [dirty, setDirty] = useState(false);
 const action = useConsoleAction(token); const intent = useRef({ payload: "", key: "" }); const bodyInput = useRef<HTMLTextAreaElement>(null); const pathInput = useRef<HTMLInputElement>(null);
 const [problem, setProblem] = useState("");
 const [brief, setBrief] = useState({ keyword: "", notes: "" }); const [writeLanguages, setWriteLanguages] = useState<WritingLanguage[]>([locale]); const [progress, setProgress] = useState("");
 const [dialog, setDialog] = useState<"translate" | "publish" | "unpublish" | null>(null); const [targets, setTargets] = useState<WritingLanguage[]>([]);
 // A new article stays in this editor after its first save and takes the saved record's id.
 const id = article?.id || routeId;
 const loader = useCallback((current: string) => readContent(current, routeId), [routeId]); const resource = useConsoleResource(routeId === "new" ? "" : token, loader);
 useEffect(() => { if (resource.data) { setArticle(resource.data); setDirty(false); } }, [resource.data]);
 const order = writingPlan(writeLanguages, language);
 const translation = article?.translations[language] || emptyTranslation();
 const options = article ? publishOptions(article) : []; const publishable = options.filter(canPublish).map(option => option.language);
 const live = writingLanguages.filter(value => article?.published[value]);
 // The audit log records what each action did; editors do not type a reason.
 const auditReason = (description: { en: string; zh: string }) => t(description);
 const names = (values: readonly WritingLanguage[]) => values.map(value => t(languageNames[value])).join(locale === "zh" ? "、" : ", ");
 function updateTranslation(next: Partial<ContentTranslation>) { setDirty(true); setArticle(current => current ? { ...current, translations: { ...current.translations, [language]: { ...(current.translations[language] || emptyTranslation()), ...next } } } : current); }
 function keyFor(payload: unknown) { const value = JSON.stringify(payload); if (value !== intent.current.payload) intent.current = { payload: value, key: newIntentKey() }; return intent.current.key; }
 function persist(current: ContentArticle, translations: ContentArticle["translations"], why: string) {
  const payload = { path: current.path, kind: current.kind, version: current.version, translations, reason: why };
  return saveContent(token, current.id || null, payload, keyFor(payload));
 }
 function saved(next: ContentArticle, message: string) {
  setArticle(next); setDirty(false); intent.current = { payload: "", key: "" }; action.setMessage(message);
  if (routeId === "new") window.history.replaceState(null, "", `/admin/articles/${next.id}`);
 }
 /** Explains what blocks a save and moves focus there; true when the article can be saved. */
 function checkSavable(current: ContentArticle) {
  if (!current.id && !validArticlePath(current.kind, current.path)) { setProblem(t({ en: `Enter the URL path first, for example /${current.kind}/team-spaces: lowercase letters, numbers and hyphens.`, zh: `请先填写文章路径，例如 /${current.kind}/team-spaces：只能用小写字母、数字和连字符。` })); pathInput.current?.focus(); return false; }
  setProblem(""); return true;
 }
 function save(event: FormEvent) {
  event.preventDefault(); if (!article || !checkSavable(article)) return;
  if (article.id && !dirty) { action.setMessage(t({ en: "Everything is already saved.", zh: "没有未保存的修改，已是最新。" })); return; }
  const current = article;
  void action.run(() => persist(current, current.translations, auditReason({ en: "Save draft", zh: "保存草稿" })), next => saved(next, t({ en: "Draft saved. The published article is unchanged.", zh: "草稿已保存，已发布的文章保持当前版本。" })));
 }
 /** Translates into the targets a few at a time; a language that fails does not stop the others. */
 async function translateInto(source: WritingLanguage, from: ContentTranslation, into: readonly WritingLanguage[]) {
  const written: Partial<Record<WritingLanguage, ContentTranslation>> = {}; const failed: WritingLanguage[] = []; let finished = 0;
  setProgress(t({ en: `Translating 0 of ${into.length}…`, zh: `正在翻译 0/${into.length}…` }));
  await mapLimit(into, 3, async target => {
   try {
    const result = await translateArticle(token, source, [target], from);
    const translated = result.translations[target];
    if (!translated) throw new Error("missing translation");
    written[target] = translated;
   } catch { failed.push(target); }
   finished++; setProgress(t({ en: `Translating ${finished} of ${into.length}…`, zh: `正在翻译 ${finished}/${into.length}…` }));
  });
  return { written, failed: writingLanguages.filter(value => failed.includes(value)) };
 }
 function translationMessage(written: WritingLanguage[], failed: WritingLanguage[]) {
  return t({ en: `Saved as drafts: ${names(written)}.`, zh: `已保存为草稿：${names(written)}。` }) + (failed.length ? t({ en: ` ${names(failed)} could not be translated; use Translate with AI to retry.`, zh: `${names(failed)}翻译失败，可点「一键翻译」重试。` }) : "") + t({ en: " Check the text, then choose Publish.", zh: "检查内容后点「发布」。" });
 }
 /** Writes the source language, translates it into the other ticked languages, and saves them all as drafts. */
 function generate() {
  if (!article || article.kind === "page" || !brief.keyword.trim() || !order) return;
  const current = article; const kind = article.kind; const { source, targets: others } = order; const keyword = brief.keyword.trim();
  void action.run(async () => {
   try {
    setProgress(t({ en: `Writing ${languageNames[source].en}…`, zh: `正在撰写${languageNames[source].zh}…` }));
    const draft = await draftArticle(token, { kind, keyword, notes: brief.notes.trim(), locale: source });
    const result = others.length ? await translateInto(source, draft.translation, others) : { written: {}, failed: [] };
    const path = current.id ? current.path : draft.path;
    const translations = { ...current.translations, ...result.written, [source]: draft.translation };
    // The generated text stays in the editor even if saving fails.
    setArticle(value => value ? { ...value, path, translations } : value); setDirty(true); setLanguage(source); setPreview(false);
    setProgress(t({ en: "Saving drafts…", zh: "正在保存草稿…" }));
    const next = await persist({ ...current, path }, translations, auditReason({ en: `AI draft: ${keyword}`, zh: `AI 写作：${keyword}` }));
    return { next, written: writingLanguages.filter(value => value === source || result.written[value]), failed: result.failed };
   } finally { setProgress(""); }
  }, ({ next, written, failed }) => saved(next, translationMessage(written, failed)));
 }
 function openTranslate() { if (!article || !checkSavable(article)) return; setTargets(translationTargets(language).filter(value => articleState(article, value) === "missing")); setDialog("translate"); }
 function translateSelected() {
  if (!article || !targets.length) return;
  const current = article; const source = language; const from = translation; const into = writingLanguages.filter(value => targets.includes(value));
  void action.run(async () => {
   try {
    const result = await translateInto(source, from, into);
    const written = writingLanguages.filter(value => result.written[value]);
    if (!written.length) throw new Error("translation failed");
    setProgress(t({ en: "Saving drafts…", zh: "正在保存草稿…" }));
    const next = await persist(current, { ...current.translations, ...result.written }, auditReason({ en: `Translate into ${written.map(value => languageNames[value].en).join(", ")}`, zh: `翻译为${written.map(value => languageNames[value].zh).join("、")}` }));
    return { next, written, failed: result.failed };
   } finally { setProgress(""); }
  }, ({ next, written, failed }) => { setDialog(null); saved(next, translationMessage(written, failed)); });
 }
 function openPublication(next: "publish" | "unpublish") { if (!article || !checkSavable(article)) return; setTargets(next === "publish" ? publishable : []); setDialog(next); }
 /** Saves unsaved edits, then publishes or unpublishes each chosen language. */
 function changePublication(operation: "publish" | "unpublish") {
  if (!article) return;
  const allowed = operation === "publish" ? publishable : live;
  const chosen = writingLanguages.filter(value => targets.includes(value) && allowed.includes(value)); if (!chosen.length) return;
  const list = (key: "en" | "zh") => chosen.map(value => languageNames[value][key]).join(key === "zh" ? "、" : ", ");
  const why = auditReason(operation === "publish" ? { en: `Publish ${list("en")}`, zh: `发布${list("zh")}` } : { en: `Unpublish ${list("en")}`, zh: `下架${list("zh")}` });
  let current = article; const unsaved = dirty || !article.id;
  void action.run(async () => {
   try {
    if (unsaved) { setProgress(t({ en: "Saving drafts…", zh: "正在保存草稿…" })); current = await persist(current, current.translations, why); setArticle(current); setDirty(false); }
    const batch = newIntentKey();
    for (const [index, value] of chosen.entries()) {
     setProgress(t(operation === "publish" ? { en: `Publishing ${index + 1} of ${chosen.length}…`, zh: `正在发布 ${index + 1}/${chosen.length}…` } : { en: `Unpublishing ${index + 1} of ${chosen.length}…`, zh: `正在下架 ${index + 1}/${chosen.length}…` }));
     current = await publishContent(token, current.id, current.version, value, operation, why, `${batch}-${value}`);
     setArticle(current);
    }
    return current;
   } finally { setProgress(""); }
  }, next => { setDialog(null); saved(next, t(operation === "publish" ? { en: `Published: ${names(chosen)}. Public pages update within about a minute.`, zh: `已发布：${names(chosen)}。公开页面通常一分钟内更新。` } : { en: `Unpublished: ${names(chosen)}. Their pages return 404 within about a minute; drafts are kept.`, zh: `已下架：${names(chosen)}。这些语言的页面通常一分钟内变为 404，草稿保留。` })); });
 }
 /** Uploads an image, inserts it at the cursor, and selects its alt text for the editor to describe. */
 function insertBodyImage(file: File) {
  if (!imageTypes.includes(file.type)) { setFileError(t({ en: "Images must be PNG, JPEG, GIF or WebP.", zh: "图片需为 PNG、JPEG、GIF 或 WebP 格式。" })); return; }
  setFileError("");
  const target = language; const body = translation.body; const cursor = bodyInput.current?.selectionEnd ?? body.length;
  const alt = defaultAlt(file.name) || t({ en: "Describe the image", zh: "图片说明" });
  void action.run(() => uploadSiteImage(token, file), uploaded => {
   const inserted = insertImageAtCursor(body, alt, uploaded.url, cursor);
   setDirty(true); setArticle(current => current ? { ...current, translations: { ...current.translations, [target]: { ...(current.translations[target] || emptyTranslation()), body: inserted.body } } } : current);
   action.setMessage(t({ en: "Image inserted. Describe it in the selected text, then save.", zh: "图片已插入。请把选中的文字改成图片说明，然后保存。" }));
   requestAnimationFrame(() => setTimeout(() => { const input = bodyInput.current; if (input) { input.focus(); input.setSelectionRange(inserted.altStart, inserted.altEnd); } }, 0));
  });
 }
 function droppedImage(files: FileList) { return Array.from(files).find(file => file.type.startsWith("image/")); }
 async function upload(file: File | undefined) {
  if (!file) return; setFileError("");
  if (!/\.md$/i.test(file.name) || file.size > 150000) { setFileError(t({ en: "Choose a Markdown .md file up to 150 KB.", zh: "请选择不超过 150 KB 的 .md 文件。" })); return; }
  try { const imported = importMarkdown(await file.text()); updateTranslation({ body: imported.body, ...(imported.title ? { title: imported.title } : {}) }); } catch { setFileError(t({ en: "The file could not be read.", zh: "无法读取该文件。" })); }
 }
 const busyLabel = (label: string) => action.busy && progress ? <PendingLabel>{progress}</PendingLabel> : label;
 const allTargets = translationTargets(language);
 return <><div className={styles.editorHeader}><Link href="/admin/articles">← {t({ en: "All articles", zh: "返回文章列表" })}</Link><h1>{t(id === "new" ? { en: "New article", zh: "新建文章" } : { en: "Edit article", zh: "编辑文章" })}</h1><p>{t({ en: "Manage homepage, section introductions and articles here. Each language is published separately; drafts stay private.", zh: "在这里管理首页、栏目介绍与文章。每种语言分别发布，草稿变更不会直接展示给访客。" })}</p></div><ConsoleGate session={session} ready={ready} admin>
  <ConsoleError error={resource.error} retry={() => void resource.refresh()} />{dialog ? null : <ConsoleError error={action.error} />}{action.message ? <p role="status" className={styles.dialogNote}>{action.message}</p> : null}
  {!article ? <p role="status">{t({ en: "Loading…", zh: "正在读取…" })}</p> : <><form className={styles.contentEditor} onSubmit={save} noValidate>
   <fieldset disabled={action.busy} className={styles.editorMeta}><div className="field"><label htmlFor="article-kind">{t({ en: "Section", zh: "所属栏目" })}</label><Select id="article-kind" disabled={id !== "new"} value={article.kind} onChange={event => { const kind = event.target.value as ContentKind; setArticle({ ...article, kind, path: `/${kind}/` }); setDirty(true); }}>{(id === "new" ? newKinds : kinds).map(kind => <option key={kind} value={kind}>{t(kindLabels[kind])}</option>)}</Select></div><div className="field"><label htmlFor="article-path">{t({ en: "URL path", zh: "文章路径" })}</label><input id="article-path" ref={pathInput} value={article.path} readOnly={id !== "new"} aria-invalid={id === "new" && !validArticlePath(article.kind, article.path)} aria-describedby={id === "new" ? "article-path-hint" : undefined} onChange={event => { setArticle({ ...article, path: event.target.value.trim().toLowerCase() }); setDirty(true); }} />{id === "new" ? <small id="article-path-hint" className={styles.writingHint}>{t({ en: "Lowercase letters, numbers and hyphens. AI writing fills it in; it cannot change after the first save.", zh: "小写字母、数字和连字符。AI 写作会自动生成；首次保存后不能修改。" })}</small> : null}</div></fieldset>
   {article.kind !== "page" ? <details className={styles.writingPanel} open={id === "new"}><summary>{t({ en: "Write with AI", zh: "AI 写作" })}</summary><fieldset disabled={action.busy} className={styles.editorFields}>
    <p className={styles.writingHint}>{t({ en: "Writes the title, summary, a body with headings and FAQ, search settings and, for a new article, its URL. Links go only to published pages. Every language is saved as a draft; nothing is public until you publish it.", zh: "生成标题、摘要、带小标题与常见问题的正文、搜索展示设置；新文章还会生成 URL。内链只指向已发布的页面。每种语言都保存为草稿，发布之前不会公开。" })}</p>
    <div className="field"><label htmlFor="writing-keyword">{t({ en: "Target keyword", zh: "目标关键词" })}</label><input id="writing-keyword" maxLength={120} value={brief.keyword} onChange={event => setBrief({ ...brief, keyword: event.target.value })} placeholder={t({ en: "For example: team workspaces for SaaS", zh: "例如：SaaS 团队空间" })} /></div>
    <div className="field"><label htmlFor="writing-notes">{t({ en: "Product facts and points to cover", zh: "产品事实与写作要点" })}</label><textarea id="writing-notes" rows={4} maxLength={4000} value={brief.notes} onChange={event => setBrief({ ...brief, notes: event.target.value })} placeholder={t({ en: "The AI states only the product facts written here.", zh: "AI 只会使用这里写明的产品事实。" })} /></div>
    <fieldset className={styles.writeLanguages}><legend>{t({ en: "Languages", zh: "生成语言" })}</legend><div className={styles.translateTargets}>
     <label><input type="checkbox" checked={writeLanguages.length === writingLanguages.length} onChange={event => setWriteLanguages(event.target.checked ? [...writingLanguages] : [])} />{t({ en: "All", zh: "全选" })}</label>
     {writingLanguages.map(value => <label key={value}><input type="checkbox" checked={writeLanguages.includes(value)} onChange={event => setWriteLanguages(current => event.target.checked ? [...current, value] : current.filter(item => item !== value))} />{t(languageNames[value])}{article.translations[value]?.title ? <small>{t({ en: " (replaces the draft)", zh: "（替换现有草稿）" })}</small> : null}</label>)}
    </div></fieldset>
    <div className="button-row"><button type="button" className="button primary" disabled={action.busy || !brief.keyword.trim() || !order} onClick={generate}>{busyLabel(order && order.targets.length ? t({ en: `Write ${order.targets.length + 1} languages`, zh: `生成 ${order.targets.length + 1} 种语言` }) : t({ en: `Write ${languageNames[order?.source ?? language].en}`, zh: `生成${languageNames[order?.source ?? language].zh}版本` }))}</button>
     {order ? <small>{order.targets.length ? t({ en: `Writes ${languageNames[order.source].en} first, then translates it into the other ${order.targets.length}.`, zh: `先写${languageNames[order.source].zh}，再翻译成另外 ${order.targets.length} 种语言。` }) : null}</small> : <small>{t({ en: "Tick at least one language.", zh: "请至少勾选一种语言。" })}</small>}</div>
   </fieldset></details> : null}
   <div className={styles.editorToolbar}><div className={styles.tabs} role="tablist" aria-label={t({ en: "Article language", zh: "文章语言" })}>{writingLanguages.map(value => <button key={value} type="button" role="tab" aria-selected={value === language} disabled={action.busy} onClick={() => setLanguage(value)} lang={languageTags[value]}>{(() => { const state = articleState(article, value); return <>{t(languageNames[value])}<i className={styles.stateDot} data-state={state} aria-hidden="true" title={t(stateBadges[state].label)} /><span className="sr-only">{`: ${t(stateBadges[state].label)}`}</span></>; })()}</button>)}</div><button className="button" type="button" disabled={action.busy || !translation.title.trim() || !translation.body.trim()} onClick={openTranslate}>{t({ en: "Translate with AI", zh: "一键翻译" })}</button><label className={styles.fileImport}>{t({ en: "Import Markdown", zh: "导入 Markdown" })}<input type="file" accept=".md,text/markdown" disabled={action.busy} onChange={event => { void upload(event.target.files?.[0]); event.target.value = ""; }} /></label><button className="button" type="button" onClick={() => setPreview(!preview)}>{t(preview ? { en: "Edit", zh: "继续编辑" } : { en: "Preview", zh: "预览" })}</button></div>
   <StateLegend missing />
   {fileError ? <p role="alert">{fileError}</p> : null}
   {preview ? <article className={styles.editorPreview}><h2>{translation.title || t({ en: "Untitled", zh: "未命名文章" })}</h2><p>{translation.summary}</p><MarkdownContent source={translation.body} locale={interfaceLocale(language)} /></article> : <fieldset disabled={action.busy} className={styles.editorFields}>
    <div className="field"><label htmlFor="article-title">{t({ en: "Title", zh: "标题" })}</label><input id="article-title" maxLength={160} value={translation.title} onChange={event => updateTranslation({ title: event.target.value })} /></div>
    <div className="field"><label htmlFor="article-summary">{t({ en: "Summary", zh: "摘要" })}</label><textarea id="article-summary" rows={2} maxLength={500} value={translation.summary} onChange={event => updateTranslation({ summary: event.target.value })} /></div>
    <div className="field"><div className={styles.bodyLabel}><label htmlFor="article-body">{t({ en: "Body (Markdown)", zh: "正文（Markdown）" })}</label><label className={styles.fileImport}>{t({ en: "Insert image", zh: "插入图片" })}<input type="file" accept={imageTypes.join(",")} disabled={action.busy} onChange={event => { const file = event.target.files?.[0]; if (file) insertBodyImage(file); event.target.value = ""; }} /></label></div>
     <textarea id="article-body" ref={bodyInput} className={styles.markdownInput} rows={22} maxLength={150000} value={translation.body} onChange={event => updateTranslation({ body: event.target.value })} aria-describedby="article-body-hint"
      onPaste={event => { const file = droppedImage(event.clipboardData.files); if (file) { event.preventDefault(); insertBodyImage(file); } }}
      onDragOver={event => { if (Array.from(event.dataTransfer.items).some(item => item.kind === "file")) event.preventDefault(); }}
      onDrop={event => { const file = droppedImage(event.dataTransfer.files); if (file) { event.preventDefault(); insertBodyImage(file); } }} />
     <small id="article-body-hint" className={styles.writingHint}>{t({ en: "Paste or drop an image here, or use Insert image. It goes in at the cursor; then type its description over the selected text.", zh: "可直接粘贴或拖入图片，也可点「插入图片」。图片插在光标处，插入后把选中的文字改成图片说明。" })}</small></div>
    <details><summary>{t({ en: "Search appearance", zh: "搜索展示设置" })}</summary><div className={styles.editorFields}><div className="field"><label htmlFor="article-seo-title">{t({ en: "SEO title (optional)", zh: "SEO 标题（选填）" })}</label><input id="article-seo-title" maxLength={160} value={translation.seo_title} onChange={event => updateTranslation({ seo_title: event.target.value })} /></div><div className="field"><label htmlFor="article-seo-summary">{t({ en: "SEO description (optional)", zh: "SEO 描述（选填）" })}</label><textarea id="article-seo-summary" maxLength={500} value={translation.seo_description} onChange={event => updateTranslation({ seo_description: event.target.value })} /></div></div></details>
   </fieldset>}
   <div className={styles.editorSave}><div className="button-row"><button type="submit" className="button" disabled={action.busy}>{t({ en: "Save draft", zh: "保存草稿" })}</button><button className="button primary" type="button" disabled={action.busy || !publishable.length} onClick={() => openPublication("publish")}>{t({ en: "Publish…", zh: "发布…" })}</button>{live.length ? <button className="button" type="button" disabled={action.busy} onClick={() => openPublication("unpublish")}>{t({ en: "Unpublish…", zh: "下架…" })}</button> : null}{article.id ? <span className={styles.saveState}>{t(dirty ? { en: "Unsaved changes", zh: "有未保存的修改" } : { en: "All changes saved", zh: "已保存" })}</span> : null}</div>{problem ? <p role="alert" className={styles.saveProblem}>{problem}</p> : null}{!publishable.length && article.id ? <p>{t({ en: "Every written language is published and unchanged.", zh: "已写的语言都已发布，没有新的修改。" })}</p> : null}</div>
  </form>
  <ResponsiveDialog open={dialog === "translate"} onClose={() => setDialog(null)} dismissible={!action.busy} title={t({ en: "Translate with AI", zh: "一键翻译" })} closeLabel={t({ en: "Close", zh: "关闭" })} description={t({ en: `Translates the ${languageNames[language].en} version into the ticked languages and saves them as drafts. Links and images are kept.`, zh: `把${languageNames[language].zh}版本翻译成勾选的语言，并保存为草稿。链接和图片会保留。` })}>
   <div className={styles.dialogForm}><ConsoleError error={action.error} />
    <div className={styles.translateTargets}><label><input type="checkbox" disabled={action.busy} checked={targets.length === allTargets.length} onChange={event => setTargets(event.target.checked ? allTargets : [])} />{t({ en: "All", zh: "全选" })}</label>{allTargets.map(value => <label key={value}><input type="checkbox" disabled={action.busy} checked={targets.includes(value)} onChange={event => setTargets(current => event.target.checked ? [...current, value] : current.filter(item => item !== value))} />{t(languageNames[value])}{articleState(article, value) !== "missing" ? <small>{t({ en: " (replaces the draft)", zh: "（替换现有草稿）" })}</small> : null}</label>)}</div>
    <div className="button-row"><button type="button" className="button primary" disabled={action.busy || !targets.length} onClick={translateSelected}>{busyLabel(t({ en: `Translate into ${targets.length} languages`, zh: `翻译 ${targets.length} 种语言` }))}</button></div>
   </div>
  </ResponsiveDialog>
  <ResponsiveDialog open={dialog === "publish" || dialog === "unpublish"} onClose={() => setDialog(null)} dismissible={!action.busy} title={t(dialog === "unpublish" ? { en: "Unpublish", zh: "下架" } : { en: "Publish", zh: "发布" })} closeLabel={t({ en: "Close", zh: "关闭" })} description={t(dialog === "unpublish" ? { en: "Tick the languages to take offline. Their pages return 404; drafts are kept and can be published again.", zh: "勾选要下架的语言。下架后这些语言的页面返回 404，草稿保留，之后可以重新发布。" } : { en: "Tick the languages to publish. Unsaved edits are saved first. Public pages update within about a minute.", zh: "勾选要发布的语言。未保存的修改会先保存。公开页面通常一分钟内更新。" })}>
   {dialog === "publish" || dialog === "unpublish" ? (() => {
    const operation = dialog; const allowed = operation === "publish" ? publishable : live; const count = targets.filter(value => allowed.includes(value)).length;
    return <div className={styles.dialogForm}><ConsoleError error={action.error} />
     <ul className={styles.publishChoices}>{options.filter(option => operation === "publish" || live.includes(option.language)).map(option => { const badge = stateBadges[option.state]; const enabled = allowed.includes(option.language); return <li key={option.language}><label><input type="checkbox" disabled={action.busy || !enabled} checked={enabled && targets.includes(option.language)} onChange={event => setTargets(current => event.target.checked ? [...current, option.language] : current.filter(item => item !== option.language))} /><span>{t(languageNames[option.language])}</span></label><StatusBadge tone={badge.tone}>{t(badge.label)}</StatusBadge>{operation === "publish" && !enabled && option.state !== "missing" ? <small>{t(option.complete ? { en: "No new edits", zh: "没有新修改" } : { en: "Needs a title, summary and body", zh: "缺少标题、摘要或正文" })}</small> : null}</li>; })}</ul>
     <div className="button-row"><button type="button" className={`button ${operation === "publish" ? "primary" : "danger"}`} disabled={action.busy || !count} onClick={() => changePublication(operation)}>{busyLabel(t(operation === "publish" ? { en: `Publish ${count} languages`, zh: `发布 ${count} 种语言` } : { en: `Unpublish ${count} languages`, zh: `下架 ${count} 种语言` }))}</button></div>
    </div>;
   })() : null}
  </ResponsiveDialog></>}
 </ConsoleGate></>;
}
