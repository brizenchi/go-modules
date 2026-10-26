"use client";
import { useEffect, useRef, useState, type FormEvent } from "react";
import type { FooterNavigationGroup, HeaderNavigationItem, NavigationSection, SiteNavigation } from "@/lib/api";
import { defaultNavigation, navigationSections, normalizeNavigation } from "@/content/navigation";
import { addHeaderItem, moveItem, navigationLimits, navigationPayload, navigationProblems, removeItem, unusedFooterSections, unusedHeaderSections, type ArticleOption } from "@/lib/navigation-editor";
import { publishedArticleOptions } from "@/lib/content-api";
import { saveNavigationSettings } from "@/lib/operations-api";
import { SITE_SETTINGS_EVENT } from "@/lib/site-settings";
import { useI18n } from "@/lib/i18n";
import { ConsoleError, newIntentKey, useConsoleAction } from "./console-kit";
import { Select } from "./select";
import { PendingLabel } from "./pending-label";
import styles from "./admin-shell.module.css";

type Copy = { en: string; zh: string };
const menuModes: Array<{ value: HeaderNavigationItem["menu"]; label: Copy }> = [
  { value: "auto", label: { en: "List published pages automatically", zh: "自动列出已发布文章" } },
  { value: "manual", label: { en: "Choose and order pages", zh: "手动选择并排序文章" } },
  { value: "none", label: { en: "No menu", zh: "不显示下拉" } }
];
const kindNames: Record<string, Copy> = {
  features: { en: "Features", zh: "功能介绍" }, "use-cases": { en: "Use cases", zh: "使用场景" }, compare: { en: "Comparisons", zh: "方案比较" },
  docs: { en: "Documentation", zh: "文档" }, blog: { en: "Articles", zh: "文章" }
};
const problemText: Record<ReturnType<typeof navigationProblems>[number], Copy> = {
  header: { en: `Use up to ${navigationLimits.header} different header sections.`, zh: `顶栏最多 ${navigationLimits.header} 个不重复的栏目。` },
  label: { en: `Names have at most ${navigationLimits.label} characters on one line.`, zh: `名称最多 ${navigationLimits.label} 个字符，且不能换行。` },
  menu: { en: `A chosen menu needs 1 to ${navigationLimits.menu} different pages.`, zh: `手动菜单需要选择 1 到 ${navigationLimits.menu} 篇不重复的文章。` },
  "footer-title": { en: "Every footer column needs an English and a Chinese title.", zh: "每个页脚分组都需要中英文标题。" },
  "footer-links": { en: `Use up to ${navigationLimits.footer} footer columns of 1 to ${navigationLimits.links} links, each link once.`, zh: `页脚最多 ${navigationLimits.footer} 组，每组 1 到 ${navigationLimits.links} 个链接，同一链接只出现一次。` }
};

/** Header and footer editor. Destinations are site sections and published pages only. */
export function NavigationSettings({ token, saved, onSaved }: { token: string; saved: SiteNavigation | null | undefined; onSaved: () => void }) {
  const { t, locale } = useI18n();
  const [draft, setDraft] = useState(() => normalizeNavigation(saved));
  useEffect(() => { setDraft(normalizeNavigation(saved)); }, [saved]);
  const [options, setOptions] = useState<ArticleOption[] | null>(null);
  const [optionsFailed, setOptionsFailed] = useState(false);
  useEffect(() => {
    let live = true;
    publishedArticleOptions().then(rows => { if (live) setOptions(rows); }).catch(() => { if (live) setOptionsFailed(true); });
    return () => { live = false; };
  }, []);
  const [reason, setReason] = useState("");
  const action = useConsoleAction(token);
  const intent = useRef({ payload: "", key: "" });
  const problems = navigationProblems(draft);
  const titles = new Map((options || []).map(option => [option.path, option.title]));
  const sectionName = (section: NavigationSection) => t(navigationSections[section].link.label);
  const defaultLabel = (section: NavigationSection) => navigationSections[section].headerLabel || navigationSections[section].link.label;
  const setHeader = (header: HeaderNavigationItem[]) => setDraft(current => ({ ...current, header }));
  const setFooter = (footer: FooterNavigationGroup[]) => setDraft(current => ({ ...current, footer }));
  const updateItem = (index: number, change: Partial<HeaderNavigationItem>) => setHeader(draft.header.map((item, position) => position === index ? { ...item, ...change } : item));
  const updateGroup = (index: number, change: Partial<FooterNavigationGroup>) => setFooter(draft.footer.map((group, position) => position === index ? { ...group, ...change } : group));

  function submit(event: FormEvent) {
    event.preventDefault();
    if (problems.length || Array.from(reason.trim()).length < 3) return;
    const navigation = navigationPayload(draft);
    const serialized = JSON.stringify({ navigation, reason: reason.trim() });
    if (intent.current.payload !== serialized) intent.current = { payload: serialized, key: newIntentKey() };
    void action.run(() => saveNavigationSettings(token, navigation, reason.trim(), intent.current.key), () => {
      setReason(""); intent.current = { payload: "", key: "" };
      window.dispatchEvent(new Event(SITE_SETTINGS_EVENT));
      action.setMessage(t({ en: "Navigation saved. The public site updates after subsequent requests, usually within one minute.", zh: "导航已保存。官网会在后续访问触发缓存刷新，通常一分钟内生效。" }));
      onSaved();
    });
  }

  function moveButtons<T>(list: readonly T[], index: number, name: string, apply: (next: T[]) => void) {
    return <div className={styles.navigationActions}>
      <button type="button" className={styles.navigationButton} disabled={index === 0} onClick={() => apply(moveItem(list, index, -1))} aria-label={t({ en: `Move ${name} up`, zh: `上移：${name}` })}>↑</button>
      <button type="button" className={styles.navigationButton} disabled={index === list.length - 1} onClick={() => apply(moveItem(list, index, 1))} aria-label={t({ en: `Move ${name} down`, zh: `下移：${name}` })}>↓</button>
      <button type="button" className={styles.navigationButton} onClick={() => apply(removeItem(list, index))} aria-label={t({ en: `Remove ${name}`, zh: `移除：${name}` })}>{t({ en: "Remove", zh: "移除" })}</button>
    </div>;
  }

  function manualPages(item: HeaderNavigationItem, index: number) {
    const available = (options || []).filter(option => !item.items.includes(option.path));
    const kinds = [...new Set(available.map(option => option.kind))];
    return <div className={styles.navigationPagesBlock}>
      {item.items.length ? <ol className={styles.navigationPages}>{item.items.map((path, position) => {
        const title = titles.get(path) || path;
        return <li key={path}><span>{title}<small>{path}</small></span>{moveButtons(item.items, position, title, items => updateItem(index, { items }))}</li>;
      })}</ol> : <p className={styles.navigationHint}>{t({ en: "Add at least one page.", zh: "至少添加一篇文章。" })}</p>}
      <div className="field">
        <label htmlFor={`navigation-add-${item.section}`}>{t({ en: "Add a page", zh: "添加文章" })}</label>
        <Select id={`navigation-add-${item.section}`} value="" disabled={!options || item.items.length >= navigationLimits.menu} onChange={event => event.target.value && updateItem(index, { items: [...item.items, event.target.value] })}>
          <option value="">{optionsFailed ? t({ en: "Published pages are unavailable", zh: "暂时无法读取已发布文章" }) : !options ? t({ en: "Loading published pages…", zh: "正在读取已发布文章…" }) : t({ en: "Choose a published page", zh: "选择一篇已发布文章" })}</option>
          {kinds.map(kind => <optgroup key={kind} label={t(kindNames[kind] || { en: kind, zh: kind })}>{available.filter(option => option.kind === kind).map(option => <option key={option.path} value={option.path}>{option.title}</option>)}</optgroup>)}
        </Select>
      </div>
    </div>;
  }

  const unusedHeader = unusedHeaderSections(draft);
  const unusedFooter = unusedFooterSections(draft);
  const canSave = !action.busy && !problems.length && Array.from(reason.trim()).length >= 3;
  return <form className={styles.websiteForm} onSubmit={submit} aria-labelledby="navigation-settings-title">
    <h2 id="navigation-settings-title" className={styles.navigationTitle}>{t({ en: "Navigation & footer", zh: "导航与页脚" })}</h2>
    <ConsoleError error={action.error} />{action.message ? <p role="status" className={styles.dialogNote}>{action.message}</p> : null}
    <fieldset disabled={action.busy} className={styles.websiteSection}>
      <legend>{t({ en: "Header", zh: "顶部导航" })}</legend>
      <p>{t({ en: "Choose the sections in the header, their order and their menus. Menus link only published pages; a section without published pages is hidden automatically. Changing menus does not remove pages from search: the sitemap follows published content.", zh: "选择顶部导航的栏目、顺序和下拉菜单。菜单只链接已发布的文章，没有内容的栏目会自动隐藏。调整导航不会让页面从搜索中消失：sitemap 按已发布内容生成。" })}</p>
      <ol className={styles.navigationList}>{draft.header.map((item, index) => {
        const name = sectionName(item.section);
        const placeholder = defaultLabel(item.section);
        return <li className={styles.navigationRow} key={item.section}>
          <div className={styles.navigationRowHead}><strong>{index + 1}. {name}</strong>{moveButtons(draft.header, index, name, setHeader)}</div>
          <div className={styles.editorMeta}>
            <div className="field"><label htmlFor={`navigation-label-en-${item.section}`}>{t({ en: "Name · EN", zh: "显示名称 · EN" })}</label><input id={`navigation-label-en-${item.section}`} value={item.label.en} placeholder={placeholder.en} maxLength={navigationLimits.label} onChange={event => updateItem(index, { label: { ...item.label, en: event.target.value } })} /></div>
            <div className="field"><label htmlFor={`navigation-label-zh-${item.section}`}>{t({ en: "Name · 中文", zh: "显示名称 · 中文" })}</label><input id={`navigation-label-zh-${item.section}`} value={item.label.zh} placeholder={placeholder.zh} maxLength={navigationLimits.label} onChange={event => updateItem(index, { label: { ...item.label, zh: event.target.value } })} /></div>
          </div>
          {navigationSections[item.section].menu ? <div className="field"><label htmlFor={`navigation-menu-${item.section}`}>{t({ en: "Menu", zh: "下拉菜单" })}</label>
            <Select id={`navigation-menu-${item.section}`} value={item.menu} onChange={event => updateItem(index, { menu: event.target.value as HeaderNavigationItem["menu"] })}>{menuModes.map(mode => <option key={mode.value} value={mode.value}>{t(mode.label)}</option>)}</Select>
          </div> : <p className={styles.navigationHint}>{t({ en: "This section is a single page without a menu.", zh: "这是单个页面，没有下拉菜单。" })}</p>}
          {item.menu === "manual" ? manualPages(item, index) : null}
        </li>;
      })}</ol>
      <div className="field">
        <label htmlFor="navigation-add-section">{t({ en: "Add a section", zh: "添加栏目" })}</label>
        <Select id="navigation-add-section" value="" disabled={draft.header.length >= navigationLimits.header || !unusedHeader.length} onChange={event => event.target.value && setDraft(addHeaderItem(draft, event.target.value as NavigationSection))}>
          <option value="">{draft.header.length >= navigationLimits.header ? t({ en: `Up to ${navigationLimits.header} sections`, zh: `最多 ${navigationLimits.header} 个栏目` }) : t({ en: "Choose a section", zh: "选择一个栏目" })}</option>
          {unusedHeader.map(section => <option key={section} value={section}>{sectionName(section)}</option>)}
        </Select>
      </div>
    </fieldset>
    <fieldset disabled={action.busy} className={styles.websiteSection}>
      <legend>{t({ en: "Footer", zh: "页脚" })}</legend>
      <p>{t({ en: "Footer columns link to sections. Privacy and terms always stay in the bottom row.", zh: "页脚分组链接到各个栏目。隐私政策和服务条款固定在底栏。" })}</p>
      <ol className={styles.navigationList}>{draft.footer.map((group, index) => {
        const name = (locale === "zh" ? group.title.zh : group.title.en) || t({ en: `Column ${index + 1}`, zh: `第 ${index + 1} 组` });
        return <li className={styles.navigationRow} key={index}>
          <div className={styles.navigationRowHead}><strong>{name}</strong>{moveButtons(draft.footer, index, name, setFooter)}</div>
          <div className={styles.editorMeta}>
            <div className="field"><label htmlFor={`footer-title-en-${index}`}>{t({ en: "Title · EN", zh: "标题 · EN" })}</label><input id={`footer-title-en-${index}`} required value={group.title.en} maxLength={navigationLimits.label} onChange={event => updateGroup(index, { title: { ...group.title, en: event.target.value } })} /></div>
            <div className="field"><label htmlFor={`footer-title-zh-${index}`}>{t({ en: "Title · 中文", zh: "标题 · 中文" })}</label><input id={`footer-title-zh-${index}`} required value={group.title.zh} maxLength={navigationLimits.label} onChange={event => updateGroup(index, { title: { ...group.title, zh: event.target.value } })} /></div>
          </div>
          {group.links.length ? <ol className={styles.navigationPages}>{group.links.map((link, position) => <li key={link}><span>{sectionName(link)}</span>{moveButtons(group.links, position, sectionName(link), links => updateGroup(index, { links }))}</li>)}</ol> : <p className={styles.navigationHint}>{t({ en: "Add at least one link.", zh: "至少添加一个链接。" })}</p>}
          <div className="field">
            <label htmlFor={`footer-add-${index}`}>{t({ en: "Add a link", zh: "添加链接" })}</label>
            <Select id={`footer-add-${index}`} value="" disabled={group.links.length >= navigationLimits.links || !unusedFooter.length} onChange={event => event.target.value && updateGroup(index, { links: [...group.links, event.target.value as NavigationSection] })}>
              <option value="">{t({ en: "Choose a section", zh: "选择一个栏目" })}</option>
              {unusedFooter.map(section => <option key={section} value={section}>{sectionName(section)}</option>)}
            </Select>
          </div>
        </li>;
      })}</ol>
      <div><button type="button" className="button" disabled={draft.footer.length >= navigationLimits.footer} onClick={() => setFooter([...draft.footer, { title: { en: "", zh: "" }, links: [] }])}>{t({ en: "Add a footer column", zh: "添加页脚分组" })}</button></div>
    </fieldset>
    <div className={styles.editorSave}>
      {problems.length ? <ul role="alert" className={styles.navigationProblems}>{problems.map(problem => <li key={problem}>{t(problemText[problem])}</li>)}</ul> : null}
      <div className="field"><label htmlFor="navigation-reason">{t({ en: "Change reason", zh: "修改原因" })}</label><input id="navigation-reason" required minLength={3} maxLength={500} value={reason} onChange={event => setReason(event.target.value)} disabled={action.busy} /></div>
      <div className={styles.navigationActions}>
        <button type="submit" className="button primary" disabled={!canSave} aria-busy={action.busy}>{action.busy ? <PendingLabel>{t({ en: "Saving…", zh: "保存中…" })}</PendingLabel> : t({ en: "Save navigation", zh: "保存导航" })}</button>
        <button type="button" className="button" disabled={action.busy} onClick={() => setDraft(defaultNavigation)}>{t({ en: "Restore defaults", zh: "恢复默认" })}</button>
      </div>
    </div>
  </form>;
}
