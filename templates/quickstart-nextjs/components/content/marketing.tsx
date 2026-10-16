"use client";

import {
  aboutPage,
  customerExamples,
  marketingCategories,
  marketingPages,
  resourcesPage,
  type MarketingKind,
  type MarketingPage
} from "@/content/marketing";
import { LocalizedLink } from "@/components/localized-link";
import { SiteShell } from "@/components/site-shell";
import { useI18n } from "@/lib/i18n";
import { getHomeContent } from "@/content/site";
import { appEnv } from "@/lib/env";
import styles from "./marketing.module.css";
import { ProductNavigation, RelatedContent } from "@/components/content-discovery";
import type { DiscoveryEntry } from "@/lib/content-discovery";

type Text = { en: string; zh: string };
type ContentSection = { title: Text; body: Text; bullets?: Text[] };
type ContentLink = { href: string; label: Text };

const labels = {
  overview: { en: "At a glance", zh: "内容摘要" },
  steps: { en: "How it works", zh: "如何使用" },
  limitations: { en: "What to know before you start", zh: "开始前需要了解" },
  faq: { en: "Common questions", zh: "常见问题" },
  related: { en: "Continue exploring", zh: "继续了解" },
  sources: { en: "Sources & further reading", zh: "来源与延伸阅读" }
};

function ContentSections({ sections }: { sections: ContentSection[] }) {
  const { t } = useI18n();
  return sections.map((section, index) => (
    <section className={styles.readingSection} id={`section-${index + 1}`} key={section.title.en}>
      <h2>{t(section.title)}</h2>
      <p>{t(section.body)}</p>
      {section.bullets?.length ? (
        <ul className={styles.bulletList}>
          {section.bullets.map((bullet) => <li key={bullet.en}>{t(bullet)}</li>)}
        </ul>
      ) : null}
    </section>
  ));
}

function RelatedLinks({ links }: { links: ContentLink[] }) {
  const { t } = useI18n();
  if (!links.length) return null;
  return (
    <section className={styles.relatedSection} id="related">
      <h2>{t(labels.related)}</h2>
      <ul className={styles.linkList}>
        {links.map((link) => (
          <li key={link.href}>
            <LocalizedLink href={link.href}>
              <span>{t(link.label)}</span><span aria-hidden="true">↗</span>
            </LocalizedLink>
          </li>
        ))}
      </ul>
    </section>
  );
}

function SourceNotes({ sources, updatedAt }: { sources: ContentLink[]; updatedAt: string }) {
  const { locale, t } = useI18n();
  const date = new Intl.DateTimeFormat(locale === "zh" ? "zh-CN" : "en", {
    year: "numeric", month: "long", day: "numeric", timeZone: "UTC"
  }).format(new Date(`${updatedAt}T00:00:00Z`));
  return (
    <footer className={styles.sourceNotes}>
      <p>{t({ en: "Last reviewed", zh: "最近核对" })} <time dateTime={updatedAt}>{date}</time></p>
      {sources.length ? (
        <>
          <h2>{t(labels.sources)}</h2>
          <ul>
            {sources.map((source) => (
              <li key={source.href}>
                <LocalizedLink href={source.href} rel="noopener noreferrer">{t(source.label)}</LocalizedLink>
              </li>
            ))}
          </ul>
        </>
      ) : null}
    </footer>
  );
}

export function MarketingIndex({ kind }: { kind: MarketingKind }) {
  const { t, locale } = useI18n();
  const category = marketingCategories.find((item) => item.kind === kind);
  if (!category) return null;
  const pages = marketingPages.filter((page) => page.kind === kind);

  return (
    <SiteShell
      variant="compact"
      eyebrow={t({ en: "THE GO MODULES TEMPLATE", zh: "GO MODULES 模板" })}
      title={t(category.title)}
      description={t(category.description)}
      breadcrumbs={[
        { href: "/", label: t({ en: "Home", zh: "首页" }) },
        { label: t(category.title) }
      ]}
    >
      <div className={styles.page}>
        <ProductNavigation currentPath={category.path} />
        <div className={styles.indexIntro}>
          <span className={styles.sectionLabel}>{t({ en: "A practical starting point", zh: "从实际需求出发" })}</span>
          <p>{t(category.intro)}</p>
        </div>
        {kind === "features" ? <div className={styles.capabilities}>{getHomeContent(appEnv.demoMode)[locale].features.map(feature => <article className={styles.capability} key={feature.label}><span>{feature.label} / {feature.detail}</span><h2>{feature.title}</h2><p>{feature.description}</p><LocalizedLink href={feature.href}>{t({ en: "Explore this capability", zh: "了解这项功能" })} <span aria-hidden="true">→</span></LocalizedLink></article>)}</div> : kind === "examples" ? <div className={styles.exampleCards}>{customerExamples.map(item => <section className={styles.exampleCard} key={item.href}><h2>{t(item.label)}</h2><p>{t(item.description)}</p><LocalizedLink className="button" href={item.href}>{t({ en: "Open", zh: "打开" })}<span aria-hidden="true">→</span></LocalizedLink></section>)}</div> : <div className={styles.indexList}>
          {pages.map((page, index) => (
            <article className={styles.indexItem} key={page.path}>
              <span className={styles.itemNumber} aria-hidden="true">{String(index + 1).padStart(2, "0")}</span>
              <div className={styles.itemCopy}>
                <h2><LocalizedLink href={page.path}>{t(page.title)}</LocalizedLink></h2>
                <p>{t(page.description)}</p>
                <LocalizedLink href={page.path} className={styles.textLink}>
                  {t({ en: "Explore the details", zh: "查看详情" })}<span aria-hidden="true">→</span>
                </LocalizedLink>
              </div>
              <dl className={styles.indexFacts}>
                {page.facts.slice(0, 2).map((fact) => (
                  <div key={fact.label.en}><dt>{t(fact.label)}</dt><dd>{t(fact.value)}</dd></div>
                ))}
              </dl>
            </article>
          ))}
        </div>}
        <div className={styles.nextStep}>
          <div>
            <span className={styles.sectionLabel}>{t({ en: "Your next step", zh: "下一步" })}</span>
            <h2>{t({ en: "See how the pieces fit your product.", zh: "看看这些能力如何用于你的产品。" })}</h2>
            <p>{t({ en: "Start with the setup guide, then connect the services your product needs.", zh: "先阅读上手指南，再接入产品需要的服务。" })}</p>
          </div>
          <LocalizedLink className="button primary" href="/docs">{t({ en: "Read the guide", zh: "阅读上手指南" })}<span aria-hidden="true">→</span></LocalizedLink>
        </div>
      </div>
    </SiteShell>
  );
}

export function MarketingDetail({ page, entries }: { page: MarketingPage; entries: DiscoveryEntry[] }) {
  const { t } = useI18n();
  const category = marketingCategories.find((item) => item.kind === page.kind);
  const toc = [
    { id: "overview", label: t(labels.overview) },
    { id: "steps", label: t(labels.steps) },
    ...page.sections.map((section, index) => ({ id: `section-${index + 1}`, label: t(section.title) })),
    { id: "limitations", label: t(labels.limitations) },
    { id: "questions", label: t(labels.faq) }
  ];

  return (
    <SiteShell
      variant="compact"
      eyebrow={category ? t(category.title) : t({ en: "TEMPLATE GUIDE", zh: "模板指南" })}
      title={t(page.title)}
      description={t(page.description)}
      breadcrumbs={[
        { href: "/", label: t({ en: "Home", zh: "首页" }) },
        ...(category ? [{ href: category.path, label: t(category.title) }] : []),
        { label: t(page.title) }
      ]}
      actions={<LocalizedLink className="button primary" href={page.primaryCta.href}>{t(page.primaryCta.label)}</LocalizedLink>}
    >
      <ProductNavigation currentPath={page.path} />
      <div className={`${styles.page} ${styles.detailLayout}`}>
        <div className={styles.readingBody}>
          <section className={styles.summary} id="overview">
            <span className={styles.sectionLabel}>{t(labels.overview)}</span>
            <p>{t(page.intro)}</p>
            <dl className={styles.facts}>
              {page.facts.map((fact) => (
                <div key={fact.label.en}><dt>{t(fact.label)}</dt><dd>{t(fact.value)}</dd></div>
              ))}
            </dl>
          </section>

          <section className={styles.readingSection} id="steps">
            <h2>{t(labels.steps)}</h2>
            <ol className={styles.steps}>
              {page.steps.map((step, index) => (
                <li key={step.title.en}>
                  <span className={styles.stepNumber} aria-hidden="true">{String(index + 1).padStart(2, "0")}</span>
                  <div><h3>{t(step.title)}</h3><p>{t(step.body)}</p></div>
                </li>
              ))}
            </ol>
          </section>

          <ContentSections sections={page.sections} />

          <section className={`${styles.readingSection} ${styles.limitations}`} id="limitations">
            <h2>{t(labels.limitations)}</h2>
            <ul className={styles.bulletList}>
              {page.limitations.map((limitation) => <li key={limitation.en}>{t(limitation)}</li>)}
            </ul>
          </section>

          <section className={styles.readingSection} id="questions">
            <h2>{t(labels.faq)}</h2>
            <div className={styles.faq}>
              {page.faq.map((item) => (
                <details key={item.question.en}>
                  <summary>{t(item.question)}<span className={styles.disclosureIcon} aria-hidden="true">+</span></summary>
                  <p>{t(item.answer)}</p>
                </details>
              ))}
            </div>
          </section>

          <SourceNotes sources={page.sources} updatedAt={page.updatedAt} />
        </div>

        <aside className={styles.sideRail}>
          <nav aria-label={t({ en: "On this page", zh: "本页内容" })}>
            <span className={styles.sectionLabel}>{t({ en: "On this page", zh: "本页内容" })}</span>
            <ol>{toc.map((item) => <li key={item.id}><a href={`#${item.id}`}>{item.label}</a></li>)}</ol>
          </nav>
          <LocalizedLink className={styles.railLink} href="/resources">{t({ en: "All guides", zh: "所有指南" })}<span aria-hidden="true">↗</span></LocalizedLink>
        </aside>
      </div>
      <RelatedContent entries={entries} currentPath={page.path} />
    </SiteShell>
  );
}

export const MarketingPageContent = MarketingDetail;

export function AboutPageContent() {
  const { t } = useI18n();
  return (
    <SiteShell
      variant="compact"
      eyebrow={t({ en: "ABOUT GO MODULES", zh: "关于 GO MODULES" })}
      title={t(aboutPage.title)}
      description={t(aboutPage.description)}
      breadcrumbs={[{ href: "/", label: t({ en: "Home", zh: "首页" }) }, { label: t(aboutPage.title) }]}
    >
      <div className={`${styles.page} ${styles.aboutLayout}`}>
        <div className={styles.readingBody}>
          <div className={styles.aboutIntro}><p>{t(aboutPage.intro)}</p></div>
          <ContentSections sections={aboutPage.sections} />
          <RelatedLinks links={aboutPage.related} />
          <SourceNotes sources={aboutPage.sources} updatedAt={aboutPage.updatedAt} />
        </div>
        <aside className={styles.aboutAside}>
          <span className={styles.sectionLabel}>{t({ en: "Built for your next product", zh: "为下一个产品打好基础" })}</span>
          <p>{t({ en: "A clear starting point. Room to make it your own.", zh: "起点清楚，也留有自由发挥的空间。" })}</p>
          <LocalizedLink href="/docs" className={styles.textLink}>{t({ en: "Explore the guide", zh: "查看使用指南" })}<span aria-hidden="true">→</span></LocalizedLink>
        </aside>
      </div>
    </SiteShell>
  );
}

export function ResourcesPageContent() {
  const { t } = useI18n();
  return (
    <SiteShell
      variant="compact"
      eyebrow={t(resourcesPage.title)}
      title={t(resourcesPage.title)}
      description={t(resourcesPage.description)}
      breadcrumbs={[{ href: "/", label: t({ en: "Home", zh: "首页" }) }, { label: t(resourcesPage.title) }]}
    >
      <div className={styles.page}>
        <div className={styles.indexIntro}>
          <span className={styles.sectionLabel}>{t({ en: "Find your next step", zh: "找到下一步" })}</span>
          <p>{t(resourcesPage.intro)}</p>
        </div>
        <div className={styles.resourceGrid}>
          {marketingCategories.map((category, index) => (
            <section className={styles.resourceGroup} key={category.kind}>
              <span className={styles.itemNumber} aria-hidden="true">{String(index + 1).padStart(2, "0")}</span>
              <h2><LocalizedLink href={category.path}>{t(category.title)}<span aria-hidden="true">↗</span></LocalizedLink></h2>
              <p>{t(category.description)}</p>
              <ul className={styles.resourceLinks}>
                {category.kind === "examples" ? customerExamples.map(item => <li key={item.href}><LocalizedLink href={item.href}>{t(item.label)}<span aria-hidden="true">→</span></LocalizedLink></li>) : marketingPages.filter((page) => page.kind === category.kind).map((page) => (
                  <li key={page.path}><LocalizedLink href={page.path}>{t(page.title)}<span aria-hidden="true">→</span></LocalizedLink></li>
                ))}
              </ul>
            </section>
          ))}
        </div>
        <div className={styles.resourceReading}>
          <ContentSections sections={resourcesPage.sections} />
          <RelatedLinks links={resourcesPage.related} />
          <SourceNotes sources={resourcesPage.sources} updatedAt={resourcesPage.updatedAt} />
        </div>
      </div>
    </SiteShell>
  );
}
