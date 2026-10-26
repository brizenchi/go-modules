"use client";

import Link from "next/link";
import { useCallback, useEffect, useRef, useState, type FormEvent } from "react";
import { useRouter } from "next/navigation";
import { useI18n } from "@/lib/i18n";
import { draftArticle, listContent, readContent, saveContent, publishContent, translateArticle, uploadSiteImage } from "@/lib/content-api";
import { importMarkdown } from "@/lib/markdown";
import type { ArticleImageSuggestion, ContentArticle, ContentKind, ContentTranslation } from "@/lib/api";
import { imageMarkdown, insertImage, insertImageAt, languageNames, translationTargets, writingLanguages, type WritingLanguage } from "@/lib/article-writing";
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
const newArticle = (): ContentArticle => ({ id: "", kind: "blog", path: "/blog/", position: 0, version: 0, translations: {}, published: {}, updated_at: "" });
const kinds: ContentKind[] = ["page", "blog", "docs", "features", "use-cases", "compare"];
const pagePaths = ["/", "/features", "/use-cases", "/compare", "/resources", "/docs", "/blog", "/about", "/updates"];
const kindLabels = { page: { en: "Pages & introductions", zh: "页面与栏目介绍" }, blog: { en: "Articles", zh: "产品文章" }, docs: { en: "Guides", zh: "使用指南" }, features: { en: "Features", zh: "功能详情" }, "use-cases": { en: "Use cases", zh: "使用场景" }, compare: { en: "Comparisons", zh: "方案比较" } };
export function ContentAdminList() {
 const { t, locale } = useI18n(); const { session, ready } = useConsoleSession(); const token = session?.user.role === "admin" ? session.token : "";
 const [page, setPage] = useState(1); const [search, setSearch] = useState(""); const [query, setQuery] = useState("");
 const loader = useCallback((current: string) => listContent(current, page, query), [page, query]); const resource = useConsoleResource(token, loader);
 return <><AdminPageHeader section="articles" onRefresh={() => void resource.refresh()} busy={resource.loading} actions={<><ContentImportDialog token={token} onImported={() => void resource.refresh()} /><Link className="button primary" href="/admin/articles/new">{t({ en: "New content", zh: "新建内容" })}</Link></>} /><ConsoleGate session={session} ready={ready} admin><ConsoleError error={resource.error} retry={() => void resource.refresh()} /><section className={styles.records}>
  <div className={styles.listControls}><form className={styles.toolbar} role="search" onSubmit={event => { event.preventDefault(); setPage(1); setQuery(search.trim()); }}><div className={styles.search}><div className="field"><label className="sr-only" htmlFor="content-query">{t({ en: "Title or article path", zh: "标题或文章路径" })}</label><input id="content-query" type="search" placeholder={t({ en: "Search title or path", zh: "搜索标题或路径" })} value={search} onChange={event => setSearch(event.target.value)} /></div><button className="button" type="submit">{t({ en: "Search", zh: "搜索" })}</button></div><span className={styles.recordCount}>{resource.data ? t({ en: `${resource.data.total} entries`, zh: `共 ${resource.data.total} 条` }) : "—"}</span></form></div>
  <DataTable caption={t({ en: "Content library", zh: "文章列表" })} columns={[
   { key: "title", heading: t({ en: "Article", zh: "文章" }), cell: (row: ContentArticle) => <span className={styles.cellIdentity}><span>{row.translations[locale]?.title || row.translations.en?.title || row.translations.zh?.title || t({ en: "Untitled", zh: "未命名文章" })}</span><small>{row.path}</small></span> },
   { key: "kind", heading: t({ en: "Section", zh: "所属栏目" }), cell: row => t(kindLabels[row.kind]) },
   { key: "languages", heading: t({ en: "Publication", zh: "语言与发布状态" }), cell: row => <div className={styles.rowActions}>{writingLanguages.filter(language => language === "en" || language === "zh" || row.translations[language] || row.published[language]).map(language => <StatusBadge key={language} tone={row.published[language] ? "success" : "neutral"}>{tabLabels[language]} · {t(row.published[language] ? { en: "Published", zh: "已发布" } : { en: "Draft", zh: "草稿" })}</StatusBadge>)}</div> },
   { key: "actions", heading: t({ en: "Action", zh: "操作" }), cell: row => <Link className="button" href={`/admin/articles/${row.id}`}>{t({ en: "Edit", zh: "编辑" })}</Link> }
  ]} rows={resource.data?.items || []} rowKey={row => row.id} loading={resource.loading} loadingLabel={t({ en: "Loading articles…", zh: "正在读取文章…" })} empty={t({ en: "No articles found.", zh: "暂无匹配文章。" })} pagination={resource.data ? { ...resource.data, onPage: setPage, previousLabel: t({ en: "Previous", zh: "上一页" }), nextLabel: t({ en: "Next", zh: "下一页" }), label: (p, n) => `${p} / ${n}` } : undefined} />
 </section></ConsoleGate></>;
}
export function ContentAdminEditor({ id }: { id: string }) {
 const { t, locale } = useI18n(); const router = useRouter(); const { session, ready } = useConsoleSession(); const token = session?.user.role === "admin" ? session.token : "";
 const [article, setArticle] = useState<ContentArticle | null>(id === "new" ? newArticle() : null); const [language, setLanguage] = useState<WritingLanguage>(locale); const [preview, setPreview] = useState(false); const [reason, setReason] = useState(""); const [fileError, setFileError] = useState(""); const [dirty, setDirty] = useState(false);
 const action = useConsoleAction(token); const intent = useRef({ payload: "", key: "" });
 const [brief, setBrief] = useState({ keyword: "", notes: "" }); const [images, setImages] = useState<ArticleImageSuggestion[]>([]); const [targets, setTargets] = useState<WritingLanguage[]>([]); const [imageAlt, setImageAlt] = useState(""); const bodyInput = useRef<HTMLTextAreaElement>(null);
 const loader = useCallback((current: string) => readContent(current, id), [id]); const resource = useConsoleResource(id === "new" ? "" : token, loader);
 useEffect(() => { if (resource.data) { setArticle(resource.data); setDirty(false); } }, [resource.data]);
 const translation = article?.translations[language] || emptyTranslation(); const validReason = Array.from(reason.trim()).length >= 3 && Array.from(reason.trim()).length <= 500;
 function updateTranslation(next: Partial<ContentTranslation>) { setDirty(true); setArticle(current => current ? { ...current, translations: { ...current.translations, [language]: { ...(current.translations[language] || emptyTranslation()), ...next } } } : current); }
 function keyFor(payload: unknown) { const value = JSON.stringify(payload); if (value !== intent.current.payload) intent.current = { payload: value, key: newIntentKey() }; return intent.current.key; }
 function save(event: FormEvent) {
  event.preventDefault(); if (!article || !validReason) return;
  const payload = { path: article.path, kind: article.kind, position: article.position, version: article.version, translations: article.translations, reason: reason.trim() };
  void action.run(() => saveContent(token, id === "new" ? null : id, payload, keyFor(payload)), saved => { setArticle(saved); setDirty(false); setReason(""); intent.current = { payload: "", key: "" }; action.setMessage(t({ en: "Draft saved. The published article is unchanged.", zh: "草稿已保存，已发布的文章保持当前版本。" })); if (id === "new") router.replace(`/admin/articles/${saved.id}`); });
 }
 function publish(operation: "publish" | "unpublish") {
  if (!article || !validReason || dirty || id === "new") return;
  const payload = { id, version: article.version, language, operation, reason: reason.trim() };
  void action.run(() => publishContent(token, id, article.version, language, operation, reason.trim(), keyFor(payload)), saved => { setArticle(saved); setDirty(false); setReason(""); intent.current = { payload: "", key: "" }; action.setMessage(t({ en: "Publication updated. Public pages refresh after subsequent requests, usually within one minute.", zh: "发布状态已更新，公开页面将在后续访问触发缓存刷新，通常一分钟内生效。" })); });
 }
 function generate() {
  if (!article || article.kind === "page" || !brief.keyword.trim()) return;
  const kind = article.kind;
  void action.run(() => draftArticle(token, { kind, keyword: brief.keyword.trim(), notes: brief.notes.trim(), locale: language }), draft => {
   setArticle(current => current ? { ...current, path: id === "new" ? draft.path : current.path, translations: { ...current.translations, [language]: draft.translation } } : current);
   setImages(draft.images); setDirty(true); setPreview(false);
   action.setMessage(t({ en: "Draft written. Check every fact, add the suggested images, then save the draft.", zh: "草稿已生成。请核对每一处事实、按建议添加配图，然后保存草稿。" }));
  });
 }
 function addImage(file: File | undefined, alt: string, section?: string) {
  if (!file) return;
  const cursor = bodyInput.current?.selectionStart;
  void action.run(() => uploadSiteImage(token, file), uploaded => {
   const markdown = imageMarkdown(alt.trim() || file.name.replace(/\.[^.]+$/, ""), uploaded.url);
   setArticle(current => {
    if (!current) return current;
    const draft = current.translations[language] || emptyTranslation();
    const body = section !== undefined ? insertImage(draft.body, markdown, section) : insertImageAt(draft.body, markdown, cursor ?? draft.body.length);
    return { ...current, translations: { ...current.translations, [language]: { ...draft, body } } };
   });
   setDirty(true);
   if (section !== undefined) setImages(current => current.filter(item => item.section !== section || item.alt !== alt)); else setImageAlt("");
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
   if (id === "new") router.replace(`/admin/articles/${saved.id}`);
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
   <fieldset disabled={action.busy} className={styles.editorMeta}><div className="field"><label htmlFor="article-kind">{t({ en: "Section", zh: "所属栏目" })}</label><Select id="article-kind" disabled={id !== "new"} value={article.kind} onChange={event => { const kind = event.target.value as ContentKind; setArticle({ ...article, kind, path: kind === "page" ? "/" : `/${kind}/` }); setDirty(true); }}>{kinds.map(kind => <option key={kind} value={kind}>{t(kindLabels[kind])}</option>)}</Select></div><div className="field"><label htmlFor="article-path">{t({ en: "URL path", zh: "文章路径" })}</label>{article.kind === "page" && id === "new" ? <Select id="article-path" value={article.path} onChange={event => { setArticle({ ...article, path: event.target.value }); setDirty(true); }}>{pagePaths.map(path => <option key={path} value={path}>{path === "/" ? t({ en: "/ · Homepage", zh: "/ · 首页" }) : path}</option>)}</Select> : <input id="article-path" value={article.path} readOnly={id !== "new"} pattern={article.kind === "page" ? undefined : "/(blog|docs|features|use-cases|compare)/[a-z0-9]+(-[a-z0-9]+)*"} required onChange={event => { setArticle({ ...article, path: event.target.value }); setDirty(true); }} />}</div><div className="field"><label htmlFor="article-position">{t({ en: "Sort order", zh: "排序" })}</label><input id="article-position" type="number" min={0} max={100000} value={article.position} onChange={event => { setArticle({ ...article, position: Number(event.target.value) }); setDirty(true); }} /></div></fieldset>
   {article.kind !== "page" ? <details className={styles.writingPanel} open={id === "new"}><summary>{t({ en: "Write with AI", zh: "AI 写作" })}</summary><fieldset disabled={action.busy} className={styles.editorFields}>
    <p className={styles.writingHint}>{t({ en: `Writes the ${languageNames[language].en} version: title, summary, a body with headings and FAQ, search settings and, for a new article, its URL. Links go only to published pages. Nothing is public until you save and publish.`, zh: `生成${languageNames[language].zh}版本：标题、摘要、带小标题与常见问题的正文、搜索展示设置；新文章还会生成 URL。内链只指向已发布的页面。保存并发布之前不会公开。` })}</p>
    <div className="field"><label htmlFor="writing-keyword">{t({ en: "Target keyword", zh: "目标关键词" })}</label><input id="writing-keyword" maxLength={120} value={brief.keyword} onChange={event => setBrief({ ...brief, keyword: event.target.value })} placeholder={t({ en: "For example: team workspaces for SaaS", zh: "例如：SaaS 团队空间" })} /></div>
    <div className="field"><label htmlFor="writing-notes">{t({ en: "Product facts and points to cover", zh: "产品事实与写作要点" })}</label><textarea id="writing-notes" rows={4} maxLength={4000} value={brief.notes} onChange={event => setBrief({ ...brief, notes: event.target.value })} placeholder={t({ en: "The AI states only the product facts written here.", zh: "AI 只会使用这里写明的产品事实。" })} /></div>
    <div className="button-row"><button type="button" className="button" disabled={action.busy || !brief.keyword.trim()} onClick={generate}>{action.busy ? <PendingLabel>{t({ en: "Writing…", zh: "生成中…" })}</PendingLabel> : translation.body.trim() ? t({ en: `Rewrite the ${languageNames[language].en} version`, zh: `重新生成并替换${languageNames[language].zh}版本` }) : t({ en: `Write the ${languageNames[language].en} version`, zh: `生成${languageNames[language].zh}版本` })}</button></div>
   </fieldset></details> : null}
   {images.length ? <section className={styles.writingImages} aria-labelledby="image-suggestions"><h2 id="image-suggestions">{t({ en: "Suggested images", zh: "配图建议" })}</h2><ul>{images.map(image => <li key={`${image.section}-${image.alt}`}><div><strong>{image.alt}</strong><small>{image.section ? t({ en: `Below “${image.section}”. `, zh: `放在「${image.section}」下方。` }) : ""}{image.description}</small></div><label className={styles.fileImport}>{t({ en: "Upload and insert", zh: "上传并插入" })}<input type="file" accept="image/png,image/jpeg,image/gif,image/webp" disabled={action.busy} onChange={event => { addImage(event.target.files?.[0], image.alt, image.section); event.target.value = ""; }} /></label></li>)}</ul></section> : null}
   <div className={styles.editorToolbar}><div className={styles.tabs} role="tablist" aria-label={t({ en: "Article language", zh: "文章语言" })}>{writingLanguages.map(value => <button key={value} type="button" role="tab" aria-selected={value === language} disabled={action.busy} onClick={() => setLanguage(value)} lang={languageTags[value]}>{tabLabels[value]}<StatusBadge tone={article.published[value] ? "success" : "neutral"}>{t(article.published[value] ? { en: "Published", zh: "已发布" } : { en: "Draft", zh: "草稿" })}</StatusBadge></button>)}</div><label className={styles.fileImport}>{t({ en: "Import Markdown", zh: "导入 Markdown" })}<input type="file" accept=".md,text/markdown" disabled={action.busy} onChange={event => { void upload(event.target.files?.[0]); event.target.value = ""; }} /></label><button className="button" type="button" onClick={() => setPreview(!preview)}>{t(preview ? { en: "Edit", zh: "继续编辑" } : { en: "Preview", zh: "预览" })}</button></div>
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
