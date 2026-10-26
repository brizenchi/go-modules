"use client";

import Link from "next/link";
import { useCallback, useEffect, useMemo, useRef, useState, type FormEvent } from "react";
import { useRouter } from "next/navigation";
import { useI18n } from "@/lib/i18n";
import { listContent, readContent, saveContent, publishContent } from "@/lib/content-api";
import { createContentTranslation, hasTranslatableContent, isTerminalTranslationStatus, isValidContentPath, otherLocale, readContentTranslation, readContentTranslationSettings, translationCheckState, translationLengthState, translationPhaseLabel } from "@/lib/content-translation";
import { importMarkdown } from "@/lib/markdown";
import type { ContentArticle, ContentKind, ContentLocale, ContentTranslation, ContentTranslationJob, ContentTranslationSettings } from "@/lib/api";
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
   { key: "languages", heading: t({ en: "Publication", zh: "语言与发布状态" }), cell: row => <div className={styles.rowActions}>{(["en", "zh"] as const).map(language => <StatusBadge key={language} tone={row.published[language] ? "success" : "neutral"}>{language === "en" ? "EN" : "中文"} · {t(row.published[language] ? { en: "Published", zh: "已发布" } : { en: "Draft", zh: "草稿" })}</StatusBadge>)}</div> },
   { key: "actions", heading: t({ en: "Action", zh: "操作" }), cell: row => <Link className="button" href={`/admin/articles/${row.id}`}>{t({ en: "Edit", zh: "编辑" })}</Link> }
  ]} rows={resource.data?.items || []} rowKey={row => row.id} loading={resource.loading} loadingLabel={t({ en: "Loading articles…", zh: "正在读取文章…" })} empty={t({ en: "No articles found.", zh: "暂无匹配文章。" })} pagination={resource.data ? { ...resource.data, onPage: setPage, previousLabel: t({ en: "Previous", zh: "上一页" }), nextLabel: t({ en: "Next", zh: "下一页" }), label: (p, n) => `${p} / ${n}` } : undefined} />
 </section></ConsoleGate></>;
}
export function ContentAdminEditor({ id }: { id: string }) {
 const { t, locale } = useI18n(); const router = useRouter(); const { session, ready } = useConsoleSession(); const token = session?.user.role === "admin" ? session.token : "";
 const [article, setArticle] = useState<ContentArticle | null>(id === "new" ? newArticle() : null); const [language, setLanguage] = useState<ContentLocale>(locale); const [preview, setPreview] = useState(false); const [reason, setReason] = useState(""); const [fileError, setFileError] = useState(""); const [dirty, setDirty] = useState(false);
 const [sourceLocale, setSourceLocale] = useState<"" | ContentLocale>(""); const [targetLocale, setTargetLocale] = useState<ContentLocale>(locale === "en" ? "zh" : "en"); const [translationJob, setTranslationJob] = useState<ContentTranslationJob | null>(null); const [jobQueryFailed, setJobQueryFailed] = useState(false); const [translationSettings, setTranslationSettings] = useState<ContentTranslationSettings | null>(null); const [pathError, setPathError] = useState(""); const [reasonError, setReasonError] = useState("");
 const action = useConsoleAction(token); const intent = useRef({ payload: "", key: "" }); const appliedTranslations = useRef(""); const translationIntent = useRef("");
 const loader = useCallback((current: string) => readContent(current, id), [id]); const resource = useConsoleResource(id === "new" ? "" : token, loader);
 useEffect(() => { if (resource.data) { setArticle(resource.data); setDirty(false); setSourceLocale(resource.data.source_locale || ""); appliedTranslations.current = JSON.stringify(resource.data.translations); } }, [resource.data]);
 useEffect(() => { if (!token) return; let cancelled = false; void readContentTranslationSettings(token).then(value => { if (!cancelled) setTranslationSettings(value); }).catch(() => { if (!cancelled) setTranslationSettings(null); }); return () => { cancelled = true; }; }, [token]);
 const activeJobId = article?.active_translation_job_id || "";
 useEffect(() => { if (!token || !activeJobId) return; let cancelled = false; void readContentTranslation(token, activeJobId).then(job => { if (!cancelled) setTranslationJob(job); }).catch(() => {}); return () => { cancelled = true; }; }, [token, activeJobId]);
 const jobActive = translationJob !== null && !isTerminalTranslationStatus(translationJob.status);
 useEffect(() => {
  if (!token || !translationJob || !jobActive) return;
  const timer = setInterval(() => { void (async () => {
   try { const next = await readContentTranslation(token, translationJob.id); setJobQueryFailed(false); setTranslationJob(next);
    if (isTerminalTranslationStatus(next.status)) { translationIntent.current = ""; action.setMessage(""); const fresh = await readContent(token, translationJob.article_id); setArticle(fresh); setDirty(false); setSourceLocale(fresh.source_locale || ""); appliedTranslations.current = JSON.stringify(fresh.translations); }
   } catch { setJobQueryFailed(true); }
  })(); }, 2000);
  return () => clearInterval(timer);
 }, [token, translationJob, jobActive, id]);
 const savedTranslations = useMemo<Partial<Record<ContentLocale, ContentTranslation>>>(() => { try { return JSON.parse(appliedTranslations.current || "{}"); } catch { return {}; } }, [appliedTranslations.current]);
 const draftEdited = useCallback((value: ContentLocale) => JSON.stringify(article?.translations[value] || {}) !== JSON.stringify(savedTranslations[value] || {}), [article, savedTranslations]);
 const translation = article?.translations[language] || emptyTranslation(); const validReason = Array.from(reason.trim()).length >= 3 && Array.from(reason.trim()).length <= 500;
 const sourceDraft = article && sourceLocale ? article.translations[sourceLocale] : undefined;
 const sourceEmpty = sourceLocale !== "" && (!sourceDraft || !hasTranslatableContent(sourceDraft));
 const lengthState = translationSettings && sourceDraft && sourceLocale && !sourceEmpty ? translationLengthState(sourceDraft, sourceLocale, translationSettings) : null;
 const checkMeta = article?.translation_meta?.[language]; const checkState = translationCheckState(checkMeta, draftEdited(language), article?.translation_source_updated?.[language] === true);
 const pathInvalid = article ? !isValidContentPath(article.kind, article.path) : false;
 function updateTranslation(next: Partial<ContentTranslation>) { setDirty(true); setArticle(current => current ? { ...current, translations: { ...current.translations, [language]: { ...(current.translations[language] || emptyTranslation()), ...next } } } : current); }
 function keyFor(payload: unknown) { const value = JSON.stringify(payload); if (value !== intent.current.payload) intent.current = { payload: value, key: newIntentKey() }; return intent.current.key; }
 function save(event: FormEvent) {
  event.preventDefault(); if (!article) return;
  let invalid = false;
  if (pathInvalid) { setPathError(t({ en: "Use a slug like /blog/my-first-post: lowercase letters, digits and hyphens only.", zh: "文章路径需要形如 /blog/my-first-post：仅小写字母、数字和连字符。" })); invalid = true; } else setPathError("");
  if (!validReason) { setReasonError(t({ en: "Describe the change in 3-500 characters; it is recorded in the audit log.", zh: "请填写 3-500 字的操作原因，将记录到审计日志。" })); invalid = true; } else setReasonError("");
  if (invalid) return;
  const payload = { path: article.path, kind: article.kind, position: article.position, version: article.version, translations: article.translations, ...(sourceLocale ? { source_locale: sourceLocale } : {}), reason: reason.trim() };
  void action.run(() => saveContent(token, id === "new" ? null : id, payload, keyFor(payload)), saved => { setArticle(saved); setDirty(false); setReason(""); setSourceLocale(saved.source_locale || sourceLocale); intent.current = { payload: "", key: "" }; appliedTranslations.current = JSON.stringify(saved.translations); action.setMessage(t({ en: "Draft saved. The published article is unchanged.", zh: "草稿已保存，已发布的文章保持当前版本。" })); if (id === "new") router.replace(`/admin/articles/${saved.id}`); });
 }
 function publish(operation: "publish" | "unpublish") {
  if (!article || !validReason || dirty || id === "new") return;
  const payload = { id, version: article.version, language, operation, reason: reason.trim() };
  void action.run(() => publishContent(token, id, article.version, language, operation, reason.trim(), keyFor(payload)), saved => { setArticle(saved); setDirty(false); setReason(""); intent.current = { payload: "", key: "" }; appliedTranslations.current = JSON.stringify(saved.translations); action.setMessage(t({ en: "Publication updated. Public pages refresh after subsequent requests, usually within one minute.", zh: "发布状态已更新，公开页面将在后续访问触发缓存刷新，通常一分钟内生效。" })); });
 }
 function translateAndCheck() {
  if (!article || !sourceLocale || jobActive || sourceEmpty) return;
  if (pathInvalid) { setPathError(t({ en: "Use a slug like /blog/my-first-post: lowercase letters, digits and hyphens only.", zh: "文章路径需要形如 /blog/my-first-post：仅小写字母、数字和连字符。" })); return; }
  setPathError("");
  const target = targetLocale === sourceLocale ? otherLocale(sourceLocale) : targetLocale;
  const note = `AI 翻译：${sourceLocale} → ${target}`;
  // Retries of one submission reuse the intent key until the job reaches a
  // terminal state; a deliberate re-translation starts a new key.
  const intentKey = translationIntent.current || (translationIntent.current = newIntentKey());
  void action.run(async () => {
   let articleId = id;
   let version = article.version;
   const payload = { path: article.path, kind: article.kind, position: article.position, version: article.version, translations: article.translations, source_locale: sourceLocale, reason: note };
   if (id === "new") {
    // A new article is created through the existing save flow first; the
    // translation job needs a saved article id and version.
    const created = await saveContent(token, null, payload, keyFor(payload));
    setArticle(created); setDirty(false); appliedTranslations.current = JSON.stringify(created.translations);
    router.replace(`/admin/articles/${created.id}`);
    articleId = created.id; version = created.version;
   } else if (dirty) {
    const saved = await saveContent(token, id, payload, keyFor(payload));
    setArticle(saved); setDirty(false); appliedTranslations.current = JSON.stringify(saved.translations); version = saved.version;
   }
   return createContentTranslation(token, articleId, { version, source_locale: sourceLocale, target_locale: target, reason: note }, intentKey);
  }, created => { setTranslationJob(created); action.setMessage(t({ en: "Translation task submitted.", zh: "翻译任务已提交。" })); });
 }
 async function upload(file: File | undefined) {
  if (!file) return; setFileError("");
  if (!/\.md$/i.test(file.name) || file.size > 150000) { setFileError(t({ en: "Choose a Markdown .md file up to 150 KB.", zh: "请选择不超过 150 KB 的 .md 文件。" })); return; }
  try { const imported = importMarkdown(await file.text()); updateTranslation({ body: imported.body, ...(imported.title ? { title: imported.title } : {}) }); } catch { setFileError(t({ en: "The file could not be read.", zh: "无法读取该文件。" })); }
 }
 const fieldsBusy = action.busy || jobActive;
 return <><div className={styles.editorHeader}><Link href="/admin/articles">← {t({ en: "All articles", zh: "返回文章列表" })}</Link><h1>{t(id === "new" ? { en: "New article", zh: "新建文章" } : { en: "Edit article", zh: "编辑文章" })}</h1><p>{t({ en: "Manage homepage, section introductions and articles here. Each language is published separately; drafts stay private.", zh: "在这里管理首页、栏目介绍与文章。每种语言分别发布，草稿变更不会直接展示给访客。" })}</p></div><ConsoleGate session={session} ready={ready} admin>
  <ConsoleError error={resource.error} retry={() => void resource.refresh()} /><ConsoleError error={action.error} />{action.message ? <p role="status" className={styles.dialogNote}>{action.message}</p> : null}
   {!article ? <p role="status">{t({ en: "Loading…", zh: "正在读取…" })}</p> : <form className={styles.contentEditor} onSubmit={save} noValidate>
   <fieldset disabled={fieldsBusy} className={styles.editorMeta}><div className="field"><label htmlFor="article-kind">{t({ en: "Section", zh: "所属栏目" })}</label><Select id="article-kind" disabled={id !== "new"} value={article.kind} onChange={event => { const kind = event.target.value as ContentKind; setArticle({ ...article, kind, path: kind === "page" ? "/" : `/${kind}/` }); setDirty(true); }}>{kinds.map(kind => <option key={kind} value={kind}>{t(kindLabels[kind])}</option>)}</Select></div><div className="field"><label htmlFor="article-path">{t({ en: "URL path", zh: "文章路径" })}</label>{article.kind === "page" && id === "new" ? <Select id="article-path" value={article.path} onChange={event => { setArticle({ ...article, path: event.target.value }); setDirty(true); setPathError(""); }}>{pagePaths.map(path => <option key={path} value={path}>{path === "/" ? t({ en: "/ · Homepage", zh: "/ · 首页" }) : path}</option>)}</Select> : <><input id="article-path" value={article.path} readOnly={id !== "new"} aria-invalid={pathError ? true : undefined} onChange={event => { setArticle({ ...article, path: event.target.value }); setDirty(true); setPathError(""); }} />{pathError ? <span className="field-error">{pathError}</span> : null}</>}</div><div className="field"><label htmlFor="article-position">{t({ en: "Sort order", zh: "排序" })}</label><input id="article-position" type="number" min={0} max={100000} value={article.position} onChange={event => { setArticle({ ...article, position: Number(event.target.value) }); setDirty(true); }} /></div><div className="field"><label htmlFor="article-source-locale">{t({ en: "Source language", zh: "原文语言" })}</label><Select id="article-source-locale" value={sourceLocale} onChange={event => { setSourceLocale(event.target.value as "" | ContentLocale); setDirty(true); }}><option value="">{t({ en: "Not confirmed", zh: "未确认" })}</option><option value="en">EN</option><option value="zh">中文</option></Select></div></fieldset>
   <div className={styles.editorToolbar}><div className={styles.tabs} role="tablist" aria-label={t({ en: "Article language", zh: "文章语言" })}>{(["en", "zh"] as const).map(value => <button key={value} type="button" role="tab" aria-selected={value === language} disabled={fieldsBusy} onClick={() => setLanguage(value)}>{value === "en" ? "EN" : "中文"}<StatusBadge tone={article.published[value] ? "success" : "neutral"}>{t(article.published[value] ? { en: "Published", zh: "已发布" } : { en: "Draft", zh: "草稿" })}</StatusBadge></button>)}</div><label className={styles.fileImport}>{t({ en: "Import Markdown", zh: "导入 Markdown" })}<input type="file" accept=".md,text/markdown" disabled={fieldsBusy} onChange={event => { void upload(event.target.files?.[0]); event.target.value = ""; }} /></label><button className="button" type="button" onClick={() => setPreview(!preview)}>{t(preview ? { en: "Edit", zh: "继续编辑" } : { en: "Preview", zh: "预览" })}</button></div>
   {fileError ? <p role="alert">{fileError}</p> : null}
   {translationSettings === null ? null : !translationSettings.enabled ? <p>{t({ en: "AI translation is not enabled on this deployment; editing, preview and publishing are unaffected.", zh: "当前部署未启用 AI 翻译；编辑、预览与发布不受影响。" })}</p> : <div className={styles.translateRow}>
    <div className={styles.translateControls}><label className={styles.translateLabel} htmlFor="translation-target">{t({ en: "Translate to", zh: "翻译为" })}</label><Select id="translation-target" value={targetLocale === sourceLocale ? otherLocale(sourceLocale || "zh") : targetLocale} disabled={fieldsBusy} onChange={event => setTargetLocale(event.target.value as ContentLocale)}>{((["en", "zh"] as const).filter(value => sourceLocale === "" || value !== sourceLocale)).map(value => <option key={value} value={value}>{value === "en" ? "EN" : "中文"}</option>)}</Select><button className="button" type="button" disabled={fieldsBusy || !sourceLocale || sourceEmpty || lengthState?.over === true} onClick={translateAndCheck}>{jobActive ? <PendingLabel>{t(translationPhaseLabel(translationJob?.phase || "queued"))}</PendingLabel> : t({ en: "Translate & check", zh: "翻译并检查" })}</button></div>
    {!sourceLocale ? <small className={styles.recordCount}>{t({ en: "Confirm the source language before translating.", zh: "翻译前请先确认原文语言。" })}</small> : sourceEmpty ? <small className={styles.recordCount}>{t({ en: "The source draft is empty; write it in its language tab first.", zh: "原文草稿是空的，请先在原文语言页签填写内容。" })}</small> : lengthState ? <small className={styles.recordCount}>{t({ en: `Source length ${lengthState.count}/${lengthState.limit} ${sourceLocale === "zh" ? "characters" : "words"}`, zh: `原文长度 ${lengthState.count}/${lengthState.limit}` })}{lengthState.over ? t({ en: " — over the AI translation limit; saving stays available.", zh: " —— 超过 AI 翻译上限；保存不受影响。" }) : ""}</small> : null}
   </div>}
   {translationJob ? <p role="status" className={styles.dialogNote}>{jobActive ? `${t({ en: "AI translation running: ", zh: "AI 翻译进行中：" })}${t(translationPhaseLabel(translationJob.phase))}（${Math.max(0, Math.round((new Date(translationJob.updated_at).getTime() - new Date(translationJob.created_at).getTime()) / 1000))}s）` : translationJob.status === "succeeded" ? (translationJob.check_status === "warning" ? t({ en: "Translation saved with quality warnings. Review the issues below; publishing stays available.", zh: "译文已保存，存在质量提醒。可查看下方问题，仍可手动发布。" }) : t({ en: "Translation saved and no obvious issues were found. Review the draft before publishing.", zh: "译文已保存，检查未发现明显问题。发布前请自行复核。" })) : t({ en: `Translation failed (${translationJob.reason ?? "unknown"}). The previous draft and the published article are unchanged.`, zh: `翻译失败（${translationJob.reason ?? "未知原因"}），原草稿与已发布内容保持不变。` })}</p> : null}
   {jobQueryFailed && jobActive ? <p role="alert">{t({ en: "Could not refresh the task status just now; retrying automatically.", zh: "刚才未能刷新任务状态，正在自动重试。" })}</p> : null}
   {checkState === "stale" ? <p role="status" className={styles.dialogNote}>{t({ en: "This translation was edited after the last AI check, so that check result is no longer current.", zh: "译文在上次 AI 检查后又被修改，该检查结果已失效。" })}</p> : null}
   {checkState === "source-updated" ? <p role="status" className={styles.dialogNote}>{t({ en: "The source content has changed since this translation was checked. Consider translating again; publishing stays available.", zh: "原文在此译文检查后又有更新，可考虑重新翻译；发布不受影响。" })}</p> : null}
   {checkState === "passed" && checkMeta ? <p role="status" className={styles.dialogNote}>{t({ en: "AI check found no obvious issues for this draft.", zh: "AI 检查未发现该草稿的明显问题。" })}</p> : null}
   {checkState === "warning" && checkMeta ? <div role="status" className={styles.dialogNote}><p>{t({ en: "AI check reports issues below. They are advisory; publishing stays available.", zh: "AI 检查发现以下问题。提醒仅供参考，仍可手动发布。" })}</p><ul>{(checkMeta.issues || []).map((issue, index) => <li key={index}>{`${issue.field} · ${issue.message_zh}`}</li>)}</ul></div> : null}
   {preview ? <article className={styles.editorPreview}><h2>{translation.title || t({ en: "Untitled", zh: "未命名文章" })}</h2><p>{translation.summary}</p><MarkdownContent source={translation.body} locale={language} /></article> : <fieldset disabled={fieldsBusy} className={styles.editorFields}>
    <div className="field"><label htmlFor="article-title">{t({ en: "Title", zh: "标题" })}</label><input id="article-title" maxLength={160} value={translation.title} onChange={event => updateTranslation({ title: event.target.value })} /></div>
    <div className="field"><label htmlFor="article-summary">{t({ en: "Summary", zh: "摘要" })}</label><textarea id="article-summary" rows={2} maxLength={500} value={translation.summary} onChange={event => updateTranslation({ summary: event.target.value })} /></div>
    <div className="field"><label htmlFor="article-body">{t({ en: "Body (Markdown)", zh: "正文（Markdown）" })}</label><textarea id="article-body" className={styles.markdownInput} rows={22} maxLength={150000} value={translation.body} onChange={event => updateTranslation({ body: event.target.value })} /></div>
    <details><summary>{t({ en: "Search appearance", zh: "搜索展示设置" })}</summary><div className={styles.editorFields}><div className="field"><label htmlFor="article-seo-title">{t({ en: "SEO title (optional)", zh: "SEO 标题（选填）" })}</label><input id="article-seo-title" maxLength={160} value={translation.seo_title} onChange={event => updateTranslation({ seo_title: event.target.value })} /></div><div className="field"><label htmlFor="article-seo-summary">{t({ en: "SEO description (optional)", zh: "SEO 描述（选填）" })}</label><textarea id="article-seo-summary" maxLength={500} value={translation.seo_description} onChange={event => updateTranslation({ seo_description: event.target.value })} /></div></div></details>
   </fieldset>}
   <div className={styles.editorSave}><div className="field"><label htmlFor="article-reason">{t({ en: "Change reason", zh: "操作原因" })}</label><input id="article-reason" maxLength={500} value={reason} aria-invalid={reasonError ? true : undefined} onChange={event => { setReason(event.target.value); setReasonError(""); }} disabled={fieldsBusy} placeholder={t({ en: "Recorded in the audit log", zh: "将记录到审计日志" })} />{reasonError ? <span className="field-error">{reasonError}</span> : null}</div><div className="button-row"><button type="submit" className="button" disabled={fieldsBusy}>{action.busy ? <PendingLabel>{t({ en: "Saving…", zh: "处理中…" })}</PendingLabel> : t({ en: "Save draft", zh: "保存草稿" })}</button><button className="button primary" type="button" disabled={fieldsBusy || !validReason || dirty || id === "new" || !translation.title.trim() || !translation.summary.trim() || !translation.body.trim()} onClick={() => publish("publish")}>{t({ en: "Publish language", zh: "发布当前语言" })}</button>{article.published[language] ? <button className="button" type="button" disabled={fieldsBusy || !validReason || dirty} onClick={() => publish("unpublish")}>{t({ en: "Unpublish language", zh: "下架当前语言" })}</button> : null}</div>{dirty ? <p>{t({ en: "Save your draft before publishing.", zh: "请先保存草稿，再发布当前语言。" })}</p> : null}{jobActive ? <p>{t({ en: "An AI translation is running for this article; editing is paused until it finishes.", zh: "这篇文章正在进行 AI 翻译，完成后才能继续编辑。" })}</p> : null}</div>
  </form>}
 </ConsoleGate></>;
}
