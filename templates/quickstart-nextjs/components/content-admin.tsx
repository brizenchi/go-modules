"use client";

import Link from "next/link";
import { useCallback, useEffect, useRef, useState, type FormEvent } from "react";
import { useI18n } from "@/lib/i18n";
import { draftArticle, listContent, readContent, saveContent, publishContent, translateArticle, uploadSiteImage } from "@/lib/content-api";
import { importMarkdown } from "@/lib/markdown";
import type { ArticleImageSuggestion, ContentArticle, ContentArticleState, ContentKind, ContentListFilters, ContentTranslation } from "@/lib/api";
import { articleState, listedLanguages, noContentFilters, withLanguage } from "@/lib/content-filters";
import { imageMarkdown, imagePlan, insertImageAt, insertPlannedImage, languageNames, mapLimit, parseImagePlan, translationTargets, writingLanguages, writingPlan, type ImagePlan, type WritingLanguage } from "@/lib/article-writing";
import { interfaceLocale, languageTags, type ContentLocale } from "@/lib/locale";

const tabLabels: Record<ContentLocale, string> = { en: "EN", zh: "中文", ja: "日本語", ko: "한국어", es: "ES", fr: "FR", de: "DE" };
import { ConsoleError, ConsoleGate, newIntentKey, useConsoleAction, useConsoleResource, useConsoleSession } from "./console-kit";
import { AdminPageHeader } from "./admin-page-header";
import { DataTable } from "./data-table";
import { MarkdownContent } from "./markdown-content";
import { StatusBadge } from "./status-badge";
import { PendingLabel } from "./pending-label";
import styles from "./admin-shell.module.css";
import { ContentImportDialog } from "./content-import-dialog";
import { Select } from "./select";

const emptyTranslation = (): ContentTranslation => ({ title: "", summary: "", body: "", seo_title: "", seo_description: "" });
const newArticle = (): ContentArticle => ({ id: "", kind: "features", path: "/features/", position: 0, version: 0, translations: {}, published: {}, updated_at: "" });
const kinds: ContentKind[] = ["page", "blog", "docs", "features", "use-cases", "compare"];
/** New articles go into the product sections; existing pages, guides and posts stay editable. */
const newKinds: ContentKind[] = ["features", "use-cases", "compare"];
const imagePlanKey = (id: string) => `article-image-plan:${id}`;
const kindLabels = { page: { en: "Pages & introductions", zh: "页面与栏目介绍" }, blog: { en: "Articles", zh: "产品文章" }, docs: { en: "Guides", zh: "使用指南" }, features: { en: "Features", zh: "功能详情" }, "use-cases": { en: "Use cases", zh: "使用场景" }, compare: { en: "Comparisons", zh: "方案比较" } };
const stateBadges: Record<ContentArticleState, { tone: "success" | "warning" | "neutral"; label: { en: string; zh: string } }> = { published: { tone: "success", label: { en: "Published", zh: "已发布" } }, changed: { tone: "warning", label: { en: "Edits unpublished", zh: "修改未发布" } }, draft: { tone: "neutral", label: { en: "Draft", zh: "草稿" } }, missing: { tone: "neutral", label: { en: "Not written", zh: "未撰写" } } };
const statusFilters: Array<{ value: ContentArticleState; label: { en: string; zh: string } }> = [{ value: "published", label: { en: "Published", zh: "已发布" } }, { value: "changed", label: { en: "Published, edits unpublished", zh: "已发布但修改未发布" } }, { value: "draft", label: { en: "Unpublished draft", zh: "有未发布的草稿" } }, { value: "missing", label: { en: "Not written in this language", zh: "缺少该语言" } }];
export function ContentAdminList() {
 const { t, locale } = useI18n(); const { session, ready } = useConsoleSession(); const token = session?.user.role === "admin" ? session.token : "";
 const [page, setPage] = useState(1); const [search, setSearch] = useState(""); const [filters, setFilters] = useState<ContentListFilters>(noContentFilters);
 const loader = useCallback((current: string) => listContent(current, page, filters), [page, filters]); const resource = useConsoleResource(token, loader);
 const filter = (next: ContentListFilters) => { setPage(1); setFilters(next); };
 const filtered = Object.values(filters).some(Boolean);
 return <><AdminPageHeader section="articles" onRefresh={() => void resource.refresh()} busy={resource.loading} actions={<><ContentImportDialog token={token} onImported={() => void resource.refresh()} /><Link className="button primary" href="/admin/articles/new">{t({ en: "New content", zh: "新建内容" })}</Link></>} /><ConsoleGate session={session} ready={ready} admin><ConsoleError error={resource.error} retry={() => void resource.refresh()} /><section className={styles.records}>
  <div className={styles.listControls}><form className={styles.toolbar} role="search" onSubmit={event => { event.preventDefault(); filter({ ...filters, query: search.trim() }); }}><div className={styles.search}><div className="field"><label className="sr-only" htmlFor="content-query">{t({ en: "Title or article path", zh: "标题或文章路径" })}</label><input id="content-query" type="search" placeholder={t({ en: "Search title or path", zh: "搜索标题或路径" })} value={search} onChange={event => setSearch(event.target.value)} /></div><button className="button" type="submit">{t({ en: "Search", zh: "搜索" })}</button></div>
   <div className="field"><label className="sr-only" htmlFor="content-kind">{t({ en: "Section", zh: "所属栏目" })}</label><Select id="content-kind" value={filters.kind} onChange={event => filter({ ...filters, kind: event.target.value as ContentListFilters["kind"] })}><option value="">{t({ en: "All sections", zh: "全部栏目" })}</option>{kinds.map(kind => <option key={kind} value={kind}>{t(kindLabels[kind])}</option>)}</Select></div>
   <div className="field"><label className="sr-only" htmlFor="content-language">{t({ en: "Language", zh: "语言" })}</label><Select id="content-language" value={filters.locale} onChange={event => filter(withLanguage(filters, event.target.value as ContentListFilters["locale"]))}><option value="">{t({ en: "All languages", zh: "全部语言" })}</option>{writingLanguages.map(language => <option key={language} value={language}>{t(languageNames[language])}</option>)}</Select></div>
   <div className="field"><label className="sr-only" htmlFor="content-status">{t({ en: "Publication status", zh: "发布状态" })}</label><Select id="content-status" value={filters.status} onChange={event => filter({ ...filters, status: event.target.value as ContentListFilters["status"] })}><option value="">{t({ en: "Any status", zh: "全部状态" })}</option>{statusFilters.filter(item => item.value !== "missing" || filters.locale).map(item => <option key={item.value} value={item.value}>{t(item.label)}</option>)}</Select></div>
   {filtered ? <button className={`button ${styles.resetFilter}`} type="button" onClick={() => { setSearch(""); filter(noContentFilters); }}>{t({ en: "Clear filters", zh: "清除筛选" })}</button> : null}
   <span className={styles.recordCount}>{resource.data ? t({ en: `${resource.data.total} entries`, zh: `共 ${resource.data.total} 条` }) : "—"}</span></form></div>
  <DataTable caption={t({ en: "Content library", zh: "文章列表" })} columns={[
   { key: "title", heading: t({ en: "Article", zh: "文章" }), cell: (row: ContentArticle) => <span className={styles.cellIdentity}><span>{(filters.locale && row.translations[filters.locale]?.title) || row.translations[locale]?.title || row.translations.en?.title || row.translations.zh?.title || t({ en: "Untitled", zh: "未命名文章" })}</span><small>{row.path}</small></span> },
   { key: "kind", heading: t({ en: "Section", zh: "所属栏目" }), cell: row => t(kindLabels[row.kind]) },
   { key: "languages", heading: t({ en: "Publication", zh: "语言与发布状态" }), cell: row => <div className={styles.rowActions}>{listedLanguages(row).map(language => { const badge = stateBadges[articleState(row, language)]; return <StatusBadge key={language} tone={badge.tone}>{tabLabels[language]} · {t(badge.label)}</StatusBadge>; })}</div> },
   { key: "actions", heading: t({ en: "Action", zh: "操作" }), cell: row => <Link className="button" href={`/admin/articles/${row.id}`}>{t({ en: "Edit", zh: "编辑" })}</Link> }
  ]} rows={resource.data?.items || []} rowKey={row => row.id} loading={resource.loading} loadingLabel={t({ en: "Loading articles…", zh: "正在读取文章…" })} empty={t(filtered ? { en: "No articles match these filters.", zh: "没有符合筛选条件的文章。" } : { en: "No articles yet.", zh: "暂无文章。" })} pagination={resource.data ? { ...resource.data, onPage: setPage, previousLabel: t({ en: "Previous", zh: "上一页" }), nextLabel: t({ en: "Next", zh: "下一页" }), label: (p, n) => `${p} / ${n}` } : undefined} />
 </section></ConsoleGate></>;
}
export function ContentAdminEditor({ id: routeId }: { id: string }) {
 const { t, locale } = useI18n(); const { session, ready } = useConsoleSession(); const token = session?.user.role === "admin" ? session.token : "";
 const [article, setArticle] = useState<ContentArticle | null>(routeId === "new" ? newArticle() : null); const [language, setLanguage] = useState<WritingLanguage>(locale); const [preview, setPreview] = useState(false); const [reason, setReason] = useState(""); const [fileError, setFileError] = useState(""); const [dirty, setDirty] = useState(false);
 const action = useConsoleAction(token); const intent = useRef({ payload: "", key: "" });
 const [brief, setBrief] = useState({ keyword: "", notes: "" }); const [plan, setPlan] = useState<ImagePlan | null>(null); const [targets, setTargets] = useState<WritingLanguage[]>([]); const [imageAlt, setImageAlt] = useState(""); const bodyInput = useRef<HTMLTextAreaElement>(null);
 const [writeLanguages, setWriteLanguages] = useState<WritingLanguage[]>([locale]); const [progress, setProgress] = useState("");
 // A new article stays in this editor after its first save and takes the saved record's id.
 const id = article?.id || routeId;
 const loader = useCallback((current: string) => readContent(current, routeId), [routeId]); const resource = useConsoleResource(routeId === "new" ? "" : token, loader);
 useEffect(() => { if (resource.data) { setArticle(resource.data); setDirty(false); } }, [resource.data]);
 // Image suggestions outlive a reload while the article is being finished.
 useEffect(() => { if (id !== "new") { try { setPlan(parseImagePlan(sessionStorage.getItem(imagePlanKey(id)))); } catch { setPlan(null); } } }, [id]);
 function keepPlan(articleId: string, next: ImagePlan | null) {
  setPlan(next);
  try { if (next?.items.length) sessionStorage.setItem(imagePlanKey(articleId), JSON.stringify(next)); else sessionStorage.removeItem(imagePlanKey(articleId)); } catch { /* storage unavailable: suggestions last for this page only */ }
 }
 const order = writingPlan(writeLanguages, language);
 const translation = article?.translations[language] || emptyTranslation(); const validReason = Array.from(reason.trim()).length >= 3 && Array.from(reason.trim()).length <= 500;
 function updateTranslation(next: Partial<ContentTranslation>) { setDirty(true); setArticle(current => current ? { ...current, translations: { ...current.translations, [language]: { ...(current.translations[language] || emptyTranslation()), ...next } } } : current); }
 function keyFor(payload: unknown) { const value = JSON.stringify(payload); if (value !== intent.current.payload) intent.current = { payload: value, key: newIntentKey() }; return intent.current.key; }
 function save(event: FormEvent) {
  event.preventDefault(); if (!article || !validReason) return;
  const payload = { path: article.path, kind: article.kind, position: article.position, version: article.version, translations: article.translations, reason: reason.trim() };
  void action.run(() => saveContent(token, id === "new" ? null : id, payload, keyFor(payload)), saved => { setArticle(saved); setDirty(false); setReason(""); intent.current = { payload: "", key: "" }; action.setMessage(t({ en: "Draft saved. The published article is unchanged.", zh: "草稿已保存，已发布的文章保持当前版本。" })); if (id === "new") window.history.replaceState(null, "", `/admin/articles/${saved.id}`); });
 }
 function publish(operation: "publish" | "unpublish") {
  if (!article || !validReason || dirty || id === "new") return;
  const payload = { id, version: article.version, language, operation, reason: reason.trim() };
  void action.run(() => publishContent(token, id, article.version, language, operation, reason.trim(), keyFor(payload)), saved => { setArticle(saved); setDirty(false); setReason(""); intent.current = { payload: "", key: "" }; action.setMessage(t({ en: "Publication updated. Public pages refresh after subsequent requests, usually within one minute.", zh: "发布状态已更新，公开页面将在后续访问触发缓存刷新，通常一分钟内生效。" })); });
 }
 /** Writes the source language, translates it into the other ticked languages a few at a time,
  * and saves every language that succeeded as a draft. */
 function generate() {
  if (!article || article.kind === "page" || !brief.keyword.trim() || !order) return;
  const current = article; const kind = article.kind; const { source, targets: others } = order; const keyword = brief.keyword.trim();
  const auditReason = reason.trim() || t({ en: `AI draft: ${keyword}`, zh: `AI 写作：${keyword}` });
  void action.run(async () => {
   try {
    setProgress(t({ en: `Writing ${languageNames[source].en}…`, zh: `正在撰写${languageNames[source].zh}…` }));
    const draft = await draftArticle(token, { kind, keyword, notes: brief.notes.trim(), locale: source });
    const written: Partial<Record<WritingLanguage, ContentTranslation>> = { [source]: draft.translation };
    const translatedImages: Partial<Record<WritingLanguage, ArticleImageSuggestion[]>> = {};
    const failed: WritingLanguage[] = []; let finished = 0;
    if (others.length) setProgress(t({ en: `Translating 0 of ${others.length}…`, zh: `正在翻译 0/${others.length}…` }));
    await mapLimit(others, 3, async target => {
     try {
      const result = await translateArticle(token, source, [target], draft.translation, draft.images);
      const translation = result.translations[target];
      if (!translation) throw new Error("missing translation");
      written[target] = translation;
      if (result.images?.[target]) translatedImages[target] = result.images[target];
     } catch { failed.push(target); }
     finished++; setProgress(t({ en: `Translating ${finished} of ${others.length}…`, zh: `正在翻译 ${finished}/${others.length}…` }));
    });
    const path = id === "new" ? draft.path : current.path;
    const translations = { ...current.translations, ...written };
    // The generated text stays in the editor even if saving fails.
    setArticle(value => value ? { ...value, path, translations } : value); setDirty(true); setLanguage(source); setPreview(false);
    setProgress(t({ en: "Saving drafts…", zh: "正在保存草稿…" }));
    const payload = { path, kind: current.kind, position: current.position, version: current.version, translations, reason: auditReason };
    const saved = await saveContent(token, id === "new" ? null : id, payload, keyFor(payload));
    return { saved, written: writingLanguages.filter(value => written[value]), failed: writingLanguages.filter(value => failed.includes(value)), images: imagePlan(source, draft.images, translatedImages) };
   } finally { setProgress(""); }
  }, ({ saved, written, failed, images }) => {
   setArticle(saved); setDirty(false); setReason(""); intent.current = { payload: "", key: "" };
   keepPlan(saved.id, images.items.length ? images : null);
   const names = (values: WritingLanguage[], key: "en" | "zh") => values.map(value => languageNames[value][key]).join(key === "zh" ? "、" : ", ");
   action.setMessage(t({
    en: `Saved as drafts: ${names(written, "en")}.${failed.length ? ` ${names(failed, "en")} could not be translated; use Translate with AI below to retry.` : ""} Check the facts${images.items.length ? ", upload the suggested images" : ""}, then publish each language.`,
    zh: `已保存为草稿：${names(written, "zh")}。${failed.length ? `${names(failed, "zh")}翻译失败，可用下方「一键翻译」重试。` : ""}请核对事实${images.items.length ? "、上传建议的配图" : ""}，再逐个语言发布。`
   }));
   if (id === "new") window.history.replaceState(null, "", `/admin/articles/${saved.id}`);
  });
 }
 /** One upload goes into every language the suggestion was translated into. */
 function addPlannedImage(file: File | undefined, index: number) {
  const item = plan?.items[index];
  if (!file || !plan || !item || !article) return;
  const articleId = article.id || id; const languages = writingLanguages.filter(value => item[value] && article.translations[value]);
  void action.run(() => uploadSiteImage(token, file), uploaded => {
   setArticle(current => {
    if (!current) return current;
    const bodies = Object.fromEntries(writingLanguages.flatMap(value => { const draft = current.translations[value]; return draft ? [[value, draft.body]] : []; }));
    const translations = { ...current.translations };
    for (const [value, body] of Object.entries(insertPlannedImage(bodies, item, plan.source, uploaded.url)) as Array<[WritingLanguage, string]>) translations[value] = { ...(translations[value] || emptyTranslation()), body };
    return { ...current, translations };
   });
   setDirty(true); keepPlan(articleId, { ...plan, items: plan.items.filter((_, position) => position !== index) });
   action.setMessage(t({ en: `Image inserted into ${languages.length} language${languages.length === 1 ? "" : "s"}. Save the draft to keep it.`, zh: `图片已插入 ${languages.length} 种语言，保存草稿后生效。` }));
  });
 }
 function addImage(file: File | undefined, alt: string) {
  if (!file) return;
  const cursor = bodyInput.current?.selectionStart;
  void action.run(() => uploadSiteImage(token, file), uploaded => {
   const markdown = imageMarkdown(alt.trim() || file.name.replace(/\.[^.]+$/, ""), uploaded.url);
   setArticle(current => {
    if (!current) return current;
    const draft = current.translations[language] || emptyTranslation();
    return { ...current, translations: { ...current.translations, [language]: { ...draft, body: insertImageAt(draft.body, markdown, cursor ?? draft.body.length) } } };
   });
   setDirty(true); setImageAlt("");
   action.setMessage(t({ en: "Image inserted. Save the draft to keep it.", zh: "图片已插入，保存草稿后生效。" }));
  });
 }
 function translateAndSave() {
  if (!article || !validReason || !targets.length) return;
  const from = language; const source = article.translations[language] || emptyTranslation(); const chosen = targets;
  void action.run(async () => {
   const result = await translateArticle(token, from, chosen, source);
   const payload = { path: article.path, kind: article.kind, position: article.position, version: article.version, translations: { ...article.translations, ...result.translations }, reason: reason.trim() };
   return saveContent(token, id === "new" ? null : id, payload, keyFor(payload));
  }, saved => {
   setArticle(saved); setDirty(false); setReason(""); setTargets([]); intent.current = { payload: "", key: "" };
   action.setMessage(t({ en: `Translated into ${chosen.map(value => languageNames[value].en).join(", ")} and saved as drafts. Review each language, then publish it.`, zh: `已翻译为${chosen.map(value => languageNames[value].zh).join("、")}并保存为草稿。请逐个检查语言版本后再发布。` }));
   if (id === "new") window.history.replaceState(null, "", `/admin/articles/${saved.id}`);
  });
 }
 async function upload(file: File | undefined) {
  if (!file) return; setFileError("");
  if (!/\.md$/i.test(file.name) || file.size > 150000) { setFileError(t({ en: "Choose a Markdown .md file up to 150 KB.", zh: "请选择不超过 150 KB 的 .md 文件。" })); return; }
  try { const imported = importMarkdown(await file.text()); updateTranslation({ body: imported.body, ...(imported.title ? { title: imported.title } : {}) }); } catch { setFileError(t({ en: "The file could not be read.", zh: "无法读取该文件。" })); }
 }
 return <><div className={styles.editorHeader}><Link href="/admin/articles">← {t({ en: "All articles", zh: "返回文章列表" })}</Link><h1>{t(id === "new" ? { en: "New article", zh: "新建文章" } : { en: "Edit article", zh: "编辑文章" })}</h1><p>{t({ en: "Manage homepage, section introductions and articles here. Each language is published separately; drafts stay private.", zh: "在这里管理首页、栏目介绍与文章。每种语言分别发布，草稿变更不会直接展示给访客。" })}</p></div><ConsoleGate session={session} ready={ready} admin>
  <ConsoleError error={resource.error} retry={() => void resource.refresh()} /><ConsoleError error={action.error} />{action.message ? <p role="status" className={styles.dialogNote}>{action.message}</p> : null}
  {!article ? <p role="status">{t({ en: "Loading…", zh: "正在读取…" })}</p> : <form className={styles.contentEditor} onSubmit={save}>
   <fieldset disabled={action.busy} className={styles.editorMeta}><div className="field"><label htmlFor="article-kind">{t({ en: "Section", zh: "所属栏目" })}</label><Select id="article-kind" disabled={id !== "new"} value={article.kind} onChange={event => { const kind = event.target.value as ContentKind; setArticle({ ...article, kind, path: `/${kind}/` }); setDirty(true); }}>{(id === "new" ? newKinds : kinds).map(kind => <option key={kind} value={kind}>{t(kindLabels[kind])}</option>)}</Select></div><div className="field"><label htmlFor="article-path">{t({ en: "URL path", zh: "文章路径" })}</label><input id="article-path" value={article.path} readOnly={id !== "new"} pattern={article.kind === "page" ? undefined : "/(blog|docs|features|use-cases|compare)/[a-z0-9]+(-[a-z0-9]+)*"} required onChange={event => { setArticle({ ...article, path: event.target.value }); setDirty(true); }} /></div><div className="field"><label htmlFor="article-position">{t({ en: "Sort order", zh: "排序" })}</label><input id="article-position" type="number" min={0} max={100000} value={article.position} onChange={event => { setArticle({ ...article, position: Number(event.target.value) }); setDirty(true); }} /></div></fieldset>
   {article.kind !== "page" ? <details className={styles.writingPanel} open={id === "new"}><summary>{t({ en: "Write with AI", zh: "AI 写作" })}</summary><fieldset disabled={action.busy} className={styles.editorFields}>
    <p className={styles.writingHint}>{t({ en: "Writes the title, summary, a body with headings and FAQ, search settings and, for a new article, its URL. Links go only to published pages. Every language is saved as a draft; nothing is public until you publish it.", zh: "生成标题、摘要、带小标题与常见问题的正文、搜索展示设置；新文章还会生成 URL。内链只指向已发布的页面。每种语言都保存为草稿，发布之前不会公开。" })}</p>
    <div className="field"><label htmlFor="writing-keyword">{t({ en: "Target keyword", zh: "目标关键词" })}</label><input id="writing-keyword" maxLength={120} value={brief.keyword} onChange={event => setBrief({ ...brief, keyword: event.target.value })} placeholder={t({ en: "For example: team workspaces for SaaS", zh: "例如：SaaS 团队空间" })} /></div>
    <div className="field"><label htmlFor="writing-notes">{t({ en: "Product facts and points to cover", zh: "产品事实与写作要点" })}</label><textarea id="writing-notes" rows={4} maxLength={4000} value={brief.notes} onChange={event => setBrief({ ...brief, notes: event.target.value })} placeholder={t({ en: "The AI states only the product facts written here.", zh: "AI 只会使用这里写明的产品事实。" })} /></div>
    <fieldset className={styles.writeLanguages}><legend>{t({ en: "Languages", zh: "生成语言" })}</legend><div className={styles.translateTargets}>
     <label><input type="checkbox" checked={writeLanguages.length === writingLanguages.length} onChange={event => setWriteLanguages(event.target.checked ? [...writingLanguages] : [])} />{t({ en: "All", zh: "全选" })}</label>
     {writingLanguages.map(value => <label key={value}><input type="checkbox" checked={writeLanguages.includes(value)} onChange={event => setWriteLanguages(current => event.target.checked ? [...current, value] : current.filter(item => item !== value))} />{t(languageNames[value])}{article.translations[value]?.title ? <small>{t({ en: " (replaces the draft)", zh: "（替换现有草稿）" })}</small> : null}</label>)}
    </div></fieldset>
    <div className="button-row"><button type="button" className="button primary" disabled={action.busy || !brief.keyword.trim() || !order} onClick={generate}>{action.busy && progress ? <PendingLabel>{progress}</PendingLabel> : order && order.targets.length ? t({ en: `Write ${order.targets.length + 1} languages`, zh: `生成 ${order.targets.length + 1} 种语言` }) : t({ en: `Write ${languageNames[order?.source ?? language].en}`, zh: `生成${languageNames[order?.source ?? language].zh}版本` })}</button>
     {order ? <small>{order.targets.length ? t({ en: `Writes ${languageNames[order.source].en} first, then translates it into the other ${order.targets.length}.`, zh: `先写${languageNames[order.source].zh}，再翻译成另外 ${order.targets.length} 种语言。` }) : null}</small> : <small>{t({ en: "Tick at least one language.", zh: "请至少勾选一种语言。" })}</small>}</div>
   </fieldset></details> : null}
   {plan?.items.length ? <section className={styles.writingImages} aria-labelledby="image-suggestions"><h2 id="image-suggestions">{t({ en: "Suggested images", zh: "配图建议" })}</h2><p className={styles.writingHint}>{t({ en: "Upload each image once: it goes under the same section in every listed language, with alt text in that language.", zh: "每张图只需上传一次，会插入到所列每种语言的对应小节下，并使用该语言的 alt 文本。" })}</p><ul>{plan.items.map((item, index) => { const image = item[language] ?? item[plan.source]; if (!image) return null; return <li key={`${index}-${image.alt}`}><div><strong>{image.alt}</strong><small>{image.section ? t({ en: `Below “${image.section}”. `, zh: `放在「${image.section}」下方。` }) : ""}{image.description}</small><small>{writingLanguages.filter(value => item[value]).map(value => tabLabels[value]).join(" · ")}</small></div><label className={styles.fileImport}>{t({ en: "Upload and insert", zh: "上传并插入" })}<input type="file" accept="image/png,image/jpeg,image/gif,image/webp" disabled={action.busy} onChange={event => { addPlannedImage(event.target.files?.[0], index); event.target.value = ""; }} /></label></li>; })}</ul></section> : null}
   <div className={styles.editorToolbar}><div className={styles.tabs} role="tablist" aria-label={t({ en: "Article language", zh: "文章语言" })}>{writingLanguages.map(value => <button key={value} type="button" role="tab" aria-selected={value === language} disabled={action.busy} onClick={() => setLanguage(value)} lang={languageTags[value]}>{tabLabels[value]}{(() => { const badge = stateBadges[articleState(article, value)]; return <StatusBadge tone={badge.tone}>{t(badge.label)}</StatusBadge>; })()}</button>)}</div><label className={styles.fileImport}>{t({ en: "Import Markdown", zh: "导入 Markdown" })}<input type="file" accept=".md,text/markdown" disabled={action.busy} onChange={event => { void upload(event.target.files?.[0]); event.target.value = ""; }} /></label><button className="button" type="button" onClick={() => setPreview(!preview)}>{t(preview ? { en: "Edit", zh: "继续编辑" } : { en: "Preview", zh: "预览" })}</button></div>
   {fileError ? <p role="alert">{fileError}</p> : null}
   {preview ? <article className={styles.editorPreview}><h2>{translation.title || t({ en: "Untitled", zh: "未命名文章" })}</h2><p>{translation.summary}</p><MarkdownContent source={translation.body} locale={interfaceLocale(language)} /></article> : <fieldset disabled={action.busy} className={styles.editorFields}>
    <div className="field"><label htmlFor="article-title">{t({ en: "Title", zh: "标题" })}</label><input id="article-title" maxLength={160} value={translation.title} onChange={event => updateTranslation({ title: event.target.value })} /></div>
    <div className="field"><label htmlFor="article-summary">{t({ en: "Summary", zh: "摘要" })}</label><textarea id="article-summary" rows={2} maxLength={500} value={translation.summary} onChange={event => updateTranslation({ summary: event.target.value })} /></div>
    <div className={styles.imageInsert}><div className="field"><label htmlFor="image-alt">{t({ en: "Image description (alt text)", zh: "图片说明（alt 文本）" })}</label><input id="image-alt" maxLength={125} value={imageAlt} onChange={event => setImageAlt(event.target.value)} placeholder={t({ en: "What the image shows", zh: "图片展示了什么" })} /></div><label className={styles.fileImport}>{t({ en: "Insert image at cursor", zh: "在光标处插入图片" })}<input type="file" accept="image/png,image/jpeg,image/gif,image/webp" disabled={action.busy || !imageAlt.trim()} onChange={event => { addImage(event.target.files?.[0], imageAlt); event.target.value = ""; }} /></label></div>
    <div className="field"><label htmlFor="article-body">{t({ en: "Body (Markdown)", zh: "正文（Markdown）" })}</label><textarea id="article-body" ref={bodyInput} className={styles.markdownInput} rows={22} maxLength={150000} value={translation.body} onChange={event => updateTranslation({ body: event.target.value })} /></div>
    <details><summary>{t({ en: "Search appearance", zh: "搜索展示设置" })}</summary><div className={styles.editorFields}><div className="field"><label htmlFor="article-seo-title">{t({ en: "SEO title (optional)", zh: "SEO 标题（选填）" })}</label><input id="article-seo-title" maxLength={160} value={translation.seo_title} onChange={event => updateTranslation({ seo_title: event.target.value })} /></div><div className="field"><label htmlFor="article-seo-summary">{t({ en: "SEO description (optional)", zh: "SEO 描述（选填）" })}</label><textarea id="article-seo-summary" maxLength={500} value={translation.seo_description} onChange={event => updateTranslation({ seo_description: event.target.value })} /></div></div></details>
   </fieldset>}
   <fieldset disabled={action.busy} className={styles.translatePanel}><legend>{t({ en: "Translate with AI", zh: "一键翻译" })}</legend>
    <p>{t({ en: `Translates the ${languageNames[language].en} version into the chosen languages and saves everything as drafts. Links and images are kept. Review each language before publishing it.`, zh: `把${languageNames[language].zh}版本翻译成勾选的语言，并一起保存为草稿。链接与图片保持不变。发布前请逐个检查。` })}</p>
    <div className={styles.translateTargets}>{translationTargets(language).map(target => <label key={target}><input type="checkbox" checked={targets.includes(target)} onChange={event => setTargets(current => event.target.checked ? [...current, target] : current.filter(value => value !== target))} />{t(languageNames[target])}{article.translations[target]?.title ? <small>{t({ en: " (replaces the current draft)", zh: "（会替换现有草稿）" })}</small> : null}</label>)}</div>
    <div className="button-row"><button type="button" className="button" disabled={!targets.length || !validReason || !translation.title.trim() || !translation.body.trim()} onClick={translateAndSave}>{t({ en: "Translate and save drafts", zh: "翻译并保存草稿" })}</button>{!validReason ? <small>{t({ en: "Enter a change reason below first.", zh: "请先在下方填写操作原因。" })}</small> : null}</div>
   </fieldset>
   <div className={styles.editorSave}><div className="field"><label htmlFor="article-reason">{t({ en: "Change reason", zh: "操作原因" })}</label><input id="article-reason" minLength={3} maxLength={500} value={reason} required onChange={event => setReason(event.target.value)} disabled={action.busy} placeholder={t({ en: "Recorded in the audit log", zh: "将记录到审计日志" })} /></div><div className="button-row"><button type="submit" className="button" disabled={action.busy || !validReason}>{action.busy ? <PendingLabel>{t({ en: "Saving…", zh: "处理中…" })}</PendingLabel> : t({ en: "Save draft", zh: "保存草稿" })}</button><button className="button primary" type="button" disabled={action.busy || !validReason || dirty || id === "new" || !translation.title.trim() || !translation.summary.trim() || !translation.body.trim()} onClick={() => publish("publish")}>{t({ en: "Publish language", zh: "发布当前语言" })}</button>{article.published[language] ? <button className="button" type="button" disabled={action.busy || !validReason || dirty} onClick={() => publish("unpublish")}>{t({ en: "Unpublish language", zh: "下架当前语言" })}</button> : null}</div>{dirty ? <p>{t({ en: "Save your draft before publishing.", zh: "请先保存草稿，再发布当前语言。" })}</p> : null}</div>
  </form>}
 </ConsoleGate></>;
}
