import type { LocalizedText } from "./articles";

export const updates: Array<{ id: string; date: string; label: LocalizedText; title: LocalizedText; items: LocalizedText[]; href: string; link: LocalizedText }> = [
  {
    id: "public-content",
    date: "2026-09-06",
    label: { en: "Website", zh: "网站内容" },
    title: { en: "Guides and product updates, in two languages", zh: "中英文指南和产品更新上线" },
    items: [
      { en: "Browse and search starter guides by title, topic, or article text.", zh: "支持按标题、主题和正文搜索使用指南。" },
      { en: "Read release notes and find privacy, terms, and contact pages from the public site.", zh: "可以从公开网站查看更新记录、隐私说明、使用条款和联系页面。" },
      { en: "Articles include stable links and page-specific sharing information.", zh: "文章具有固定链接和独立的页面分享信息。" }
    ],
    href: "/blog",
    link: { en: "Read the guides", zh: "阅读指南" }
  },
  {
    id: "invitations",
    date: "2026-09-06",
    label: { en: "Invitations", zh: "邀请推荐" },
    title: { en: "A clearer invitation journey", zh: "更清晰的邀请流程" },
    items: [
      { en: "Invitation links carry a code into signup.", zh: "邀请链接可将邀请码带入注册流程。" },
      { en: "The referral center shows invitation statuses, rewards, and activation deadlines.", zh: "推荐中心展示邀请状态、奖励和激活截止时间。" }
    ],
    href: "/blog/understand-invitation-rewards",
    link: { en: "How invitations work", zh: "了解邀请流程" }
  }
];
