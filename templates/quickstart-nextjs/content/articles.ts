import type { Locale } from "../lib/locale";

export type LocalizedText = Record<Locale, string>;
export type ArticleSection = { id: string; title: LocalizedText; paragraphs: LocalizedText[] };
export type Article = {
  slug: string;
  publishedAt: string;
  category: LocalizedText;
  title: LocalizedText;
  summary: LocalizedText;
  sections: ArticleSection[];
};

// Local, version-controlled publishing. Add both translations before publishing.
// These are practical starter guides, not customer stories or product claims.
export const articles: Article[] = [
  {
    slug: "from-template-to-your-saas",
    publishedAt: "2026-09-06",
    category: { en: "Getting started", zh: "开始使用" },
    title: { en: "Make the template your product", zh: "把模板变成你的产品" },
    summary: { en: "Connect your services, define one useful workflow, and give your first customers a clear place to start.", zh: "接入自己的服务，做好一个有用的业务流程，让第一批用户知道从哪里开始。" },
    sections: [
      {
        id: "start-with-the-job",
        title: { en: "Start with the job your customer needs done", zh: "先确定用户要完成什么" },
        paragraphs: [
          { en: "Authentication, subscriptions, and invitations provide the foundation. Your product still needs a concrete job: create a report, organize a project, or deliver another result your customer values. Choose one workflow and make the path from signing up to that result easy to follow.", zh: "登录、订阅和邀请提供了基础能力。你的产品还需要一个明确的用途：生成报告、管理项目，或者交付其他有价值的结果。先选一个流程，让用户从注册到拿到结果都能顺利完成。" },
          { en: "Replace the example name, introduction, and plan descriptions together. Tell visitors who the product helps, what they can do, and what the next step is.", zh: "一起替换示例品牌、首页介绍和套餐说明。让访问者清楚知道产品适合谁、能做什么，以及下一步该点哪里。" }
        ]
      },
      {
        id: "connect-your-services",
        title: { en: "Connect the accounts you own", zh: "接入你自己的服务账号" },
        paragraphs: [
          { en: "Configure the frontend and API domains, database, authentication providers, Resend sender, and Stripe catalog for your deployment. Keep private credentials on the backend. The setup guide explains which settings belong on each side.", zh: "为自己的部署配置前端与 API 域名、数据库、登录服务、Resend 发件人和 Stripe 商品。私密凭证放在后端。配置文档会说明前后端各自需要哪些设置。" },
          { en: "Keep payments in the provider's test environment while you check the flow. A test payment label in the UI is only a display setting; the backend credentials determine which payment environment is used.", zh: "验证流程时使用支付服务的测试环境。界面上的测试支付提示只是展示设置，实际使用哪个支付环境由后端凭证决定。" }
        ]
      },
      {
        id: "check-the-whole-journey",
        title: { en: "Check the complete customer journey", zh: "验证完整的用户体验" },
        paragraphs: [
          { en: "Use separate accounts to check signup, the first useful action, a test subscription, and an invitation. Confirm what each account sees when the operation succeeds, is still processing, or fails. Also check the experience on a narrow screen.", zh: "使用不同账号验证注册、第一次使用业务功能、测试订阅和邀请。确认每个账号在成功、处理中和失败时看到的内容，并检查小屏幕上的体验。" },
          { en: "Before sharing the site publicly, replace the starter privacy and terms text with your own policies and add a support channel you actually monitor.", zh: "公开分享网站前，按实际业务完善隐私和条款页面，并配置一个有人处理的支持渠道。" }
        ]
      }
    ]
  },
  {
    slug: "understand-invitation-rewards",
    publishedAt: "2026-09-06",
    category: { en: "Growth", zh: "用户增长" },
    title: { en: "Know where an invitation stands", zh: "看懂每一次邀请的进展" },
    summary: { en: "A shared link, an attributed signup, and a rewarded invitation are three different steps.", zh: "分享链接、注册归因和获得奖励，是邀请流程里的三个不同阶段。" },
    sections: [
      {
        id: "share-your-link",
        title: { en: "Share your personal invitation link", zh: "分享你的专属邀请链接" },
        paragraphs: [
          { en: "Sign in and open the referral center to copy your link. When another person opens it, the invitation page keeps the code for the sign-in flow. Opening a link alone does not mean an invitation has been attributed or rewarded.", zh: "登录后到推荐中心复制专属链接。对方打开链接后，邀请页会保留邀请码供登录流程使用。仅仅打开链接，并不代表已经建立邀请关系或发放奖励。" }
        ]
      },
      {
        id: "read-the-status",
        title: { en: "Follow the status after signup", zh: "注册之后，查看邀请状态" },
        paragraphs: [
          { en: "A pending invitation means the relationship has been recorded and the activation requirement has not yet been met. An activated invitation has met that requirement. An expired invitation passed its activation deadline. The referral center shows the records available to your account.", zh: "“待激活”表示邀请关系已经记录，但尚未满足激活条件；“已激活”表示已经满足条件；“已过期”表示超过了激活期限。推荐中心展示与你账号相关的邀请记录。" },
          { en: "The starter connects activation to a qualifying subscription event. The operator controls the reward policy and activation window. Check the published rules for the deployment you are using before expecting a reward.", zh: "模板通过符合条件的订阅事件激活邀请。奖励规则和激活期限由站点运营者配置，请以正在使用的站点公布的规则为准。" }
        ]
      },
      {
        id: "test-with-two-accounts",
        title: { en: "Test with two separate accounts", zh: "用两个独立账号测试" },
        paragraphs: [
          { en: "Keep the inviter signed in in one browser session. Open the invitation in a separate browser profile and register a new account. Complete the qualifying action in the test payment environment, then refresh the inviter's referral center to check the result.", zh: "在一个浏览器会话中保留邀请人登录状态。在另一个浏览器个人资料中打开邀请链接并注册新账号，使用支付测试环境完成符合条件的操作，再刷新邀请人的推荐中心检查结果。" },
          { en: "Invitation credits are product credits. They are separate from any third-party affiliate commission the template author may receive from a marked service recommendation.", zh: "邀请奖励是产品内积分，与模板作者通过明确标注的第三方服务推荐链接获得的佣金是两回事。" }
        ]
      }
    ]
  },
  {
    slug: "publish-useful-product-content",
    publishedAt: "2026-09-06",
    category: { en: "Product operations", zh: "产品运营" },
    title: { en: "Give your product a place to explain itself", zh: "让用户有地方了解你的产品" },
    summary: { en: "Use guides, release notes, and a reachable support channel to answer the questions around your product.", zh: "用操作指南、更新记录和联系渠道，回答用户在使用产品时遇到的问题。" },
    sections: [
      {
        id: "write-for-a-real-question",
        title: { en: "Answer a real question in each article", zh: "每篇文章回答一个真实问题" },
        paragraphs: [
          { en: "Start with questions your first users ask: how to get a first result, how credits work, or where to find a subscription invoice. Keep the title specific and put the answer near the start.", zh: "从第一批用户的问题写起：如何完成第一次操作、积分如何使用、在哪里找订阅发票。标题要具体，把答案放在靠前的位置。" },
          { en: "The blog stores articles alongside the source code. Each article has a stable URL, a summary, and English and Chinese text. Search matches the article's title, category, summary, and body.", zh: "博客文章与源码一起管理。每篇文章都有固定链接、摘要和中英文正文。搜索会匹配文章标题、分类、摘要和正文。" }
        ]
      },
      {
        id: "publish-honest-updates",
        title: { en: "Describe changes people can use", zh: "更新记录写用户能感知的变化" },
        paragraphs: [
          { en: "A useful release note says what changed and where to use it. Keep unfinished ideas in your internal plan. Link to a guide when a change needs more explanation.", zh: "好的更新记录会告诉用户改了什么、在哪里使用。尚未完成的想法留在内部计划里。需要更多说明时，链接到相关指南。" }
        ]
      },
      {
        id: "make-help-reachable",
        title: { en: "Make help easy to find", zh: "让用户容易找到帮助" },
        paragraphs: [
          { en: "Configure a support email or an HTTPS support page. The contact page displays configured channels and otherwise explains that direct support is not yet available. Do not publish an address or response-time promise you do not maintain.", zh: "配置支持邮箱或 HTTPS 支持页面。联系页会展示已配置的渠道；如果尚未配置，会说明目前没有直接支持入口。不要发布无人维护的邮箱，也不要承诺无法兑现的回复时间。" }
        ]
      }
    ]
  }
];

export function findArticle(slug: string): Article | undefined {
  return articles.find((article) => article.slug === slug);
}

export function searchArticles(query: string, locale: Locale, source: Article[] = articles): Article[] {
  const terms = query.trim().toLocaleLowerCase(locale).split(/\s+/u).filter(Boolean);
  return source.filter((article) => {
    const searchable = [article.title[locale], article.summary[locale], article.category[locale],
      ...article.sections.flatMap((section) => [section.title[locale], ...section.paragraphs.map((paragraph) => paragraph[locale])])
    ].join(" ").toLocaleLowerCase(locale);
    return terms.every((term) => searchable.includes(term));
  });
}

export function formatContentDate(value: string, locale: Locale): string {
  return new Intl.DateTimeFormat(locale === "zh" ? "zh-CN" : "en", {
    year: "numeric", month: "long", day: "numeric", timeZone: "UTC"
  }).format(new Date(`${value}T00:00:00Z`));
}
