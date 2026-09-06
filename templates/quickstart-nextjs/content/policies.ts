import type { ArticleSection, LocalizedText } from "./articles";

export type Policy = { title: LocalizedText; summary: LocalizedText; sections: ArticleSection[] };

// Editable starter copy. Supply actual practices and terms before launch.
export const policies: Record<"privacy" | "terms", Policy> = {
  privacy: {
    title: { en: "Privacy, explained plainly.", zh: "把隐私说明写清楚。" },
    summary: { en: "A starting point for explaining what the service stores and how people can ask about their information.", zh: "说明服务保存哪些信息，以及用户如何咨询与自己信息有关的问题。" },
    sections: [
      { id: "operator", title: { en: "Who operates this service", zh: "谁在运营这项服务" }, paragraphs: [{ en: "This is editable starter content. The operator must add their identity, contact details, effective date, and any notices required for the actual service. These details have not been supplied in the template.", zh: "这是可编辑的模板内容。运营者需要补充主体身份、联系方式、生效日期，以及实际服务要求的说明。模板尚未填写这些信息。" }] },
      { id: "information", title: { en: "Information used by the starter", zh: "模板使用的信息" }, paragraphs: [{ en: "Account features use an account identifier, email address, and profile information. Depending on the sign-in method, information may also come from your chosen identity provider. Subscription and invitation features keep records for your plan, payment status, referral relationships, and credits.", zh: "账号功能使用账号标识、邮箱和个人资料。根据登录方式，部分信息可能来自你选择的身份提供方。订阅和邀请功能保存套餐、支付状态、邀请关系和积分记录。" }, { en: "Your browser stores the login session, language preference, and invitation information used during sign-in. An operator who adds analytics, support tools, file uploads, or other services must describe those additions here.", zh: "浏览器保存登录会话、语言偏好和登录流程使用的邀请信息。运营者如果接入统计、客服、文件上传或其他服务，也需要在这里说明新增的数据处理。" }] },
      { id: "providers", title: { en: "Connected service providers", zh: "接入的服务提供方" }, paragraphs: [{ en: "This starter can connect identity providers, Resend for email, and Stripe for payments. The actual providers depend on the operator’s configuration. Replace this paragraph with the providers in use, the information shared with each, and relevant transfer or storage details.", zh: "模板可以接入身份服务、Resend 邮件和 Stripe 支付。实际服务取决于运营者的配置。请将本段替换为正在使用的服务、与各服务共享的信息，以及相关的数据传输和存储说明。" }] },
      { id: "retention", title: { en: "Retention and requests", zh: "保存期限与用户请求" }, paragraphs: [{ en: "Retention periods, deletion procedures, and the handling of access or correction requests must be defined by the operator. The template does not promise a deletion period or response deadline. The contact page shows support channels the operator has configured.", zh: "数据保存期限、删除流程，以及查询或更正请求的处理方式，需要由运营者明确。模板没有承诺具体删除期限或回复时限。联系页面展示运营者已配置的支持渠道。" }] }
    ]
  },
  terms: {
    title: { en: "Know how the service works.", zh: "了解服务的使用规则。" },
    summary: { en: "A place to explain access, subscriptions, credits, and support for your own product.", zh: "在这里说明你的产品如何使用，以及订阅、积分和支持的具体规则。" },
    sections: [
      { id: "service", title: { en: "The service and its operator", zh: "服务内容与运营主体" }, paragraphs: [{ en: "This is editable starter content, not a completed service agreement. Add the operator’s identity, what the service provides, eligibility requirements, an effective date, and the rules appropriate to your deployment.", zh: "这是可编辑的模板内容，尚未构成完整的服务协议。请补充运营主体、提供的服务、使用条件、生效日期，以及适用于实际部署的规则。" }] },
      { id: "account", title: { en: "Accounts and access", zh: "账号与访问" }, paragraphs: [{ en: "Use a sign-in method you control and keep authentication details private. The operator must describe account restrictions, cancellation procedures, and the handling of misuse. Add rules for user content or uploaded files if the product accepts them.", zh: "请使用自己能够管理的登录方式，并妥善保管身份验证信息。运营者需要说明账号限制、注销流程和违规处理方式。如果产品允许用户提交内容或上传文件，还需补充相应规则。" }] },
      { id: "payments", title: { en: "Subscriptions and payments", zh: "订阅与支付" }, paragraphs: [{ en: "The template itself is shared for free. The pricing page demonstrates plans for a product built with it. Prices, billing intervals, renewal terms, cancellation, and refunds must match the operator’s published offer and actual payment configuration.", zh: "模板本身免费分享。价格页演示的是基于模板构建的产品套餐。价格、计费周期、续订、取消和退款规则，需要与公布的方案和实际支付配置一致。" }, { en: "When the deployment uses a payment provider’s test environment, test transactions do not represent a real purchase. A display label alone does not determine whether the backend uses test or live credentials.", zh: "部署使用支付服务的测试环境时，测试交易不代表真实购买。界面的展示标签并不能决定后端使用的是测试凭证还是真实凭证。" }] },
      { id: "credits", title: { en: "Credits and invitations", zh: "积分与邀请" }, paragraphs: [{ en: "Credits are used inside the product. The operator must publish their uses, any expiry rules, and the conditions for invitation rewards. Product credits are distinct from third-party affiliate commissions and should not be presented as a cash balance.", zh: "积分用于产品内部。运营者需要公布积分用途、过期规则和邀请奖励条件。产品积分与第三方联盟佣金不同，不应当被展示为现金余额。" }] },
      { id: "support", title: { en: "Support and changes", zh: "支持与规则调整" }, paragraphs: [{ en: "Use the contact page to find published support channels. Add your actual support scope, change-notice process, dispute handling, and other relevant terms before publishing this as a finished agreement.", zh: "可以通过联系页面查找已公布的支持渠道。作为正式协议发布前，请补充实际支持范围、规则变更通知方式、争议处理和其他相关条款。" }] }
    ]
  }
};
