import type { LocalizedText } from "./articles";
import { siteLinks } from "./navigation";

export type { LocalizedText } from "./articles";
export type MarketingKind = "features" | "use-cases" | "compare" | "examples";
export type MarketingLink = { href: string; label: LocalizedText };
export type MarketingSection = { title: LocalizedText; body: LocalizedText; bullets?: LocalizedText[] };
export type MarketingCategory = {
  kind: MarketingKind;
  path: string;
  title: LocalizedText;
  description: LocalizedText;
  intro: LocalizedText;
};
export type MarketingPage = {
  kind: MarketingKind;
  path: string;
  title: LocalizedText;
  description: LocalizedText;
  intro: LocalizedText;
  primaryCta: MarketingLink;
  facts: { label: LocalizedText; value: LocalizedText }[];
  steps: { title: LocalizedText; body: LocalizedText }[];
  sections: MarketingSection[];
  limitations: LocalizedText[];
  faq: { question: LocalizedText; answer: LocalizedText }[];
  related: MarketingLink[];
  sources: MarketingLink[];
  updatedAt: string;
};
export type MarketingUtilityPage = Pick<MarketingPage, "path" | "title" | "description" | "intro" | "sections" | "related" | "sources" | "updatedAt">;

const text = (en: string, zh: string): LocalizedText => ({ en, zh });
const link = (href: string, en: string, zh: string): MarketingLink => ({ href, label: text(en, zh) });
const updatedAt = "2026-10-14";
const repository = "https://github.com/brizenchi/go-modules";
const source = link(`${repository}/tree/main/templates`, "Template source on GitHub", "GitHub 模板源码");
const frontendReadme = link(`${repository}/blob/main/templates/quickstart-nextjs/README.md`, "Frontend setup and included pages", "前端配置与页面说明");
const creditGuide = link(`${repository}/blob/main/templates/quickstart/CREDITS_SERVICE.md`, "Credit service and its boundaries", "积分服务与适用边界");
const exportSource = link(`${repository}/blob/main/templates/quickstart/internal/feature/credits/export.go`, "Note export implementation", "笔记导出实现");

// These pages explain the shipped starter. Keep example workflows distinct from
// capabilities a developer still needs to build for their own product.
export const marketingCategories: MarketingCategory[] = [
  {
    kind: "features", path: "/features",
    title: text("The foundations of your SaaS", "搭建 SaaS 所需的基础能力"),
    description: text("Explore authentication, Stripe subscriptions, and credit accounting in the free Next.js + Go SaaS starter.", "了解免费 Next.js + Go SaaS 模板已有的登录、Stripe 订阅与积分账本。"),
    intro: text("See what is already connected, how customers use it, and which services you need to configure.", "查看哪些流程已经接通、用户如何使用，以及你需要配置哪些服务。")
  },
  {
    kind: "use-cases", path: "/use-cases",
    title: text("Start with the way your product charges", "从你的产品收费方式开始"),
    description: text("Choose a starting workflow for a subscription SaaS, a prepaid credit product, or paid file exports.", "为订阅产品、预付积分产品或付费文件导出选择合适的起步流程。"),
    intro: text("These are implementation starting points. Each explains the existing workflow and the product-specific work that remains.", "这些场景展示可复用的起点，并明确已有流程与仍需开发的业务。")
  },
  {
    kind: "compare", path: "/compare",
    title: text("Choose your starting point", "选择适合你的开发起点"),
    description: text("Compare adapting the Next.js + Go starter with building your SaaS foundations from scratch.", "比较基于 Next.js + Go 模板开发与自行搭建 SaaS 基础能力的适用条件。"),
    intro: text("Choose based on your stack, your workflow, and the responsibilities you are ready to own.", "根据技术栈、业务流程和维护范围做选择。")
  },
  {
    kind: "examples", path: "/examples",
    title: text("Explore the customer account", "体验账户服务"),
    description: text("Try account access, subscription management and referral rewards.", "体验账户中心、订阅与账单，以及推荐奖励。"),
    intro: text("Open one of the three common account services. Sign in when prompted to view your own records.", "选择以下三个通用账户服务，需要时登录后查看自己的记录。")
  }
];

/** Direct product entries; no separate article is needed to try account services. */
export const customerExamples = [
  { ...link("/account", "My account", "账户中心"), description: text("Sign in, view account information and check your credit balance.", "登录、查看账户信息与积分余额。") },
  { ...link("/billing", "Subscription & billing", "订阅与账单"), description: text("Review your subscription, payment methods and billing history.", "查看订阅状态、付款方式与账单记录。") },
  { ...link("/referrals", "Referral rewards", "推荐奖励"), description: text("Share your invitation link and review your referral records.", "分享邀请链接，查看推荐与奖励记录。") }
];

export const marketingPages: MarketingPage[] = [
  {
    kind: "features", path: "/features/authentication", updatedAt,
    title: text("Email and OAuth login for your SaaS", "为 SaaS 接入邮箱与 OAuth 登录"),
    description: text("Start with email-code login, optional Google and GitHub sign-in, a read-only account dialog, and a separate administrator login.", "从邮箱验证码、可选 Google 与 GitHub 登录、只读账户弹窗和独立管理员登录开始。"),
    intro: text("Customers can create an account with an email code and return to their account center. Google and GitHub sign-in are available when you configure those providers.", "用户可通过邮箱验证码创建账号，再进入账户中心。配置对应服务后，还可以使用 Google 和 GitHub 登录。"),
    primaryCta: link("/account", "Try customer login", "体验用户登录"),
    facts: [
      { label: text("Customer access", "用户登录"), value: text("Email code; optional Google / GitHub", "邮箱验证码；可选 Google / GitHub") },
      { label: text("Account", "账户管理"), value: text("Read-only email dialog and sign-out", "只读邮箱弹窗与退出") },
      { label: text("Administrator", "管理员"), value: text("Separate access and permissions", "独立入口与权限") }
    ],
    steps: [
      { title: text("Choose an available method", "选择已启用的登录方式"), body: text("Open the account page. The interface shows the methods enabled by the backend.", "打开账户页，界面会显示后端已启用的登录方式。") },
      { title: text("Complete verification", "完成身份验证"), body: text("Enter your email code or complete the configured provider flow. A new email account is created after verification.", "输入邮箱验证码，或完成已配置的第三方登录流程。新邮箱通过验证后创建账号。") },
      { title: text("Continue your task", "继续刚才的任务"), body: text("Open your account information in a compact dialog to view the read-only sign-in email or sign out. Use the account center for subscriptions, credit history and referrals.", "在简洁的账户弹窗中查看只读登录邮箱或退出登录，从账户中心管理订阅、积分与邀请奖励。") }
    ],
    sections: [
      { title: text("One identity across the customer journey", "用同一个账号完成用户流程"), body: text("The customer account connects subscriptions, credit transactions, purchases, and referrals. It gives your product a place to continue after a visit to a public page.", "账户连接订阅、积分流水、购买和邀请记录，让访客从公开页面进入产品后有明确的操作空间。") },
      { title: text("Customer and administrator access have different jobs", "用户与管理员各自完成自己的任务"), body: text("Customers use their own account center and view a read-only sign-in email in the account dialog. Authorized administrators use a separate console to operate the site; a normal customer login does not grant administrator access.", "用户使用自己的账户中心，在账户弹窗查看只读登录邮箱；获得授权的管理员通过独立后台运营网站。普通用户登录不会获得管理员权限。") }
    ],
    limitations: [
      text("Email delivery and OAuth require your own service configuration. Available login methods depend on the providers enabled for this site.", "邮件发送与 OAuth 需要接入自己的服务配置，可用登录方式以本站启用的服务为准。"),
      text("Team invitations, enterprise SSO, and your product's organization model still need their own implementation.", "团队邀请、企业 SSO 和产品自己的组织模型仍需单独实现。")
    ],
    faq: [
      { question: text("Do customers need a password?", "普通用户需要设置密码吗？"), answer: text("The supplied customer flow uses email codes or an enabled OAuth provider. The administrator password flow is separate.", "模板中的普通用户通过邮箱验证码或已启用的 OAuth 登录；管理员密码登录是独立流程。") },
      { question: text("Does downloading the template enable Google login?", "下载模板后就能直接使用 Google 登录吗？"), answer: text("No. Register your own OAuth application, configure the backend credentials and callback, and enable the provider before offering it to customers.", "需要先注册自己的 OAuth 应用，配置后端凭证与回调地址并启用服务，再向用户开放。") }
    ],
    related: [link("/docs/auth-email", "Configure authentication", "配置登录服务"), link("/features/subscriptions", "Connect login to subscriptions", "接上订阅流程"), link("/examples", "Explore account services", "体验账户服务")],
    sources: [frontendReadme, link("/docs/admin", "Administrator access guide", "管理员访问指南")]
  },
  {
    kind: "features", path: "/features/subscriptions", updatedAt,
    title: text("Stripe subscriptions with customer billing controls", "支持用户自助管理的 Stripe 订阅"),
    description: text("Explore checkout, plan-change previews, cancellation, reactivation, invoices, and the Stripe Billing Portal in the starter.", "了解模板中的结账、套餐变更预览、取消与恢复订阅、账单和 Stripe 自助门户。"),
    intro: text("The starter connects plan selection to Stripe Checkout and a customer billing page. Customers can review their current subscription and see a change preview before confirming a new plan.", "模板将套餐选择、Stripe Checkout 和用户账单页接在一起。用户可以查看当前订阅，并在确认更换套餐前查看变更预览。"),
    primaryCta: link("/pricing", "Explore the sample plans", "查看示例套餐"),
    facts: [
      { label: text("Purchase", "购买"), value: text("Stripe Checkout", "Stripe Checkout") },
      { label: text("Management", "管理"), value: text("Preview, change, cancel, reactivate", "预览、变更、取消与恢复") },
      { label: text("Records", "记录"), value: text("Invoices and purchase history", "账单与购买历史") }
    ],
    steps: [
      { title: text("Select a configured offer", "选择已配置的商品"), body: text("Choose an available plan and billing period. The prices belong to your future SaaS; the template source itself is free.", "选择可用套餐与计费周期。页面价格用于演示未来 SaaS 的收费，模板源码本身免费。") },
      { title: text("Complete checkout", "完成结账"), body: text("Continue to Stripe after signing in. Payment events update the subscription; returning from checkout alone is not proof of payment.", "登录后进入 Stripe，支付事件会更新订阅状态；仅返回网站不代表付款成功。") },
      { title: text("Manage the subscription", "管理订阅"), body: text("Review renewal information, preview a plan change, or use the available cancellation and billing controls.", "查看续费信息、预览套餐变更，或使用可用的取消订阅与账单管理操作。") }
    ],
    sections: [
      { title: text("Show the effect of a plan change", "先说明套餐变更会发生什么"), body: text("A preview reports the amount due now, target plan, and whether the change is immediate or at period end. Pending payments and scheduled changes remain visible on the billing page.", "变更预览说明本次应付金额、目标套餐，以及立即生效还是周期结束后生效。待支付状态和计划中的变更会显示在账单页。") },
      { title: text("Turn a paid plan into your product's entitlement", "把已付费套餐接到产品权益"), body: text("Billing provides subscription and membership state. You decide which real features each plan unlocks and enforce those rules in your own product operations.", "计费流程提供订阅与会员状态。每档套餐开放哪些真实功能，以及如何在业务操作中校验权益，需要由你的产品定义。") }
    ],
    limitations: [
      text("Checkout requires configured Stripe products, credentials, and payment events. Purchases are available only when payment services are enabled.", "结账需要配置 Stripe 商品、凭证与支付事件；启用支付服务后才能购买。"),
      text("Refund and dispute events provide extension hooks. They do not automatically reverse product entitlements or credit purchases.", "退款与拒付事件提供业务扩展接口，不会自动撤销产品权益或积分购买。")
    ],
    faq: [
      { question: text("Are the displayed prices the price of the template?", "页面上的价格是购买模板的费用吗？"), answer: text("No. The source is free. The pricing page demonstrates offers you can configure for your own customers; hosting and service usage have separate costs.", "不是。源码免费，价格页演示你可以向自己的客户出售的套餐；部署与第三方服务费用另计。") },
      { question: text("Where does a customer change or cancel a plan?", "用户在哪里更换或取消套餐？"), answer: text("In the subscriptions and billing page after login. Available actions depend on the current subscription and the configured payment provider.", "登录后进入订阅与账单页。可用操作取决于当前订阅状态和支付服务配置。") }
    ],
    related: [link("/docs/payments", "Configure and test payments", "配置并测试支付"), link("/use-cases/subscription-saas", "Plan a subscription product", "规划订阅产品"), link("/features/credits", "Explore prepaid credits", "了解预付积分")],
    sources: [link("/docs/payments", "Payment setup guide", "支付配置指南"), link(`${repository}/blob/main/templates/quickstart/PAYMENT_HOOKS.md`, "Refund and dispute extension points", "退款与拒付扩展说明")]
  },
  {
    kind: "features", path: "/features/credits", updatedAt,
    title: text("A credit balance with a transaction history", "每笔变化都有记录的积分账本"),
    description: text("Use the starter's credit service for grants, purchases, expiry, consumption and recorded returns in your own product.", "使用模板的积分服务处理发放、购买、到期、消费和退回，再接入自己的产品业务。"),
    intro: text("Credits connect a customer's balance to individual actions. The starter records where credits came from, when they expire, and which operation consumed them.", "积分将用户余额与具体操作连接起来。模板记录积分来源、到期情况，以及每次消费对应的业务操作。"),
    primaryCta: link("/credits", "Open balance and history", "查看积分与明细"),
    facts: [
      { label: text("Customer view", "用户视图"), value: text("Balance, expiry, and transactions", "余额、到期信息与流水") },
      { label: text("Operations", "业务操作"), value: text("Grant, consume, and refund", "发放、消费与退回") },
      { label: text("Integration", "接入方式"), value: text("Product-owned paid actions", "接入产品自己的收费操作") }
    ],
    steps: [
      { title: text("Define the paid action", "定义收费动作"), body: text("Choose the outcome customers buy and set its credit cost on the server. Your product should display that cost before confirmation.", "明确用户购买的结果，在服务端设置积分消耗。你的产品应在用户确认前显示当前价格。") },
      { title: text("Complete the action once", "完成一次业务操作"), body: text("For local database work, the credit service can store the result and charge in one transaction. A retry reuses the same operation identifier.", "针对本地数据库操作，积分服务可在同一事务中保存结果并扣分；重试需要复用同一个操作编号。") },
      { title: text("Explain the balance change", "说明余额为什么变化"), body: text("The customer checks the transaction history. Authorized administrators can review entries and issue a recorded full refund of a consumption.", "用户可以查看流水；获得授权的管理员可核对记录，并按消费流水执行有记录的全额积分退回。") }
    ],
    sections: [
      { title: text("A history customers can inspect", "用户能够核对的消费记录"), body: text("A remaining balance alone does not explain what happened. Grants, purchases, spending, expiry, and refunds give customers and operators a shared record to investigate.", "只有余额不足以解释发生了什么。发放、购买、消费、到期和退回记录，让用户与运营人员可以对照同一份账本排查问题。") },
      { title: text("Credit refunds and cash refunds are different", "积分退回与现金退款分别处理"), body: text("Returning credits restores a recorded consumption. It does not refund a Stripe charge. Decide how your product handles a cash refund, revoked access, and already-spent credits.", "积分退回用于返还一笔已记录的积分消费，不会退还 Stripe 付款。现金退款、权益撤回和已使用积分如何处理，需要由你的产品制定规则。") }
    ],
    limitations: [
      text("This is a prepaid credit ledger, not a complete metered billing system with usage aggregation and invoices in arrears.", "这是预付积分账本，没有完整的用量汇总与后付费出账系统。"),
      text("External AI or other API calls need a separate task and compensation design. Credit reservation, settlement, and partial refunds are not included.", "外部 AI 等 API 调用需要独立的任务与补偿设计；当前不提供积分预冻结、结算或部分退款。")
    ],
    faq: [
      { question: text("Does retrying a paid export charge again?", "付费导出重试会再次扣分吗？"), answer: text("Retrying the same successful operation with the same identifier returns its saved result. A new operation is a new purchase; your product must preserve the identifier when retrying.", "使用同一操作编号重试已成功的操作，会返回保存的结果。新操作属于新的消费；产品应在重试时保留原编号。") },
      { question: text("Can credits expire?", "积分可以设置有效期吗？"), answer: text("The service supports grants with an expiry date, and the customer credit page shows expiry information. Publish the expiry rules you choose for your product.", "服务支持发放带到期时间的积分，用户积分页会显示到期信息。上线时应公开自己产品采用的有效期规则。") }
    ],
    related: [link("/examples", "Explore account services", "体验账户服务"), link("/use-cases/usage-based-products", "Design a credit-based product", "设计积分产品"), link("/docs/payments", "Configure credit purchases", "配置积分购买")],
    sources: [creditGuide, exportSource]
  },
  {
    kind: "use-cases", path: "/use-cases/subscription-saas", updatedAt,
    title: text("Build a SaaS with recurring subscriptions", "搭建按周期订阅的 SaaS"),
    description: text("Connect signup, plan selection, subscription state, and customer self-service while adding the core feature your SaaS sells.", "连接注册、套餐选择、订阅状态与用户自助管理，并接入你真正要出售的核心业务。"),
    intro: text("Use a subscription when customers pay for continued access to a service. The starter supplies the account and billing journey; your product supplies the recurring value.", "当用户为持续使用服务付费时，可以采用订阅模式。模板提供账户与计费流程，你的产品负责交付值得持续付费的功能。"),
    primaryCta: link("/docs/overview", "Start with the template guide", "从模板指南开始"),
    facts: [
      { label: text("Payment pattern", "收费方式"), value: text("Recurring access", "按周期获得使用权限") },
      { label: text("Included journey", "已有流程"), value: text("Signup → checkout → billing", "注册 → 结账 → 账单管理") },
      { label: text("Your work", "你的业务"), value: text("Core feature and plan entitlements", "核心功能与套餐权益") }
    ],
    steps: [
      { title: text("Name the outcome", "定义用户要获得的结果"), body: text("Describe the task your customer repeats and what each plan includes. Replace the starter's sample offers with your own.", "写清用户会重复完成什么任务、每档套餐包含什么，并替换模板中的示例商品。") },
      { title: text("Connect billing and access", "连接计费与使用权限"), body: text("Configure Stripe, then check membership and resource ownership in the product operations that require paid access.", "配置 Stripe，在需要付费的业务操作中校验会员权益和资源归属。") },
      { title: text("Test the full lifecycle", "验证完整订阅周期"), body: text("Test successful and failed payments, plan changes, cancellation, and reactivation before offering the service to customers.", "向客户开放前，验证付款成功与失败、套餐变更、取消和恢复订阅。") }
    ],
    sections: [
      { title: text("Example: a paid workspace feature", "场景示例：工作台中的会员功能"), body: text("A customer signs in, selects a plan, and gains access to a feature you implement. They later change plans or cancel from billing. This is a proposed product flow, not an additional feature already shipped in the template.", "用户登录、选择套餐后，获得你实现的某项工作台功能。之后可在账单页更换或取消套餐。这是业务接入示例，不代表模板已内置这项额外功能。") },
      { title: text("Keep purchasing and daily work connected", "让购买流程与日常使用衔接"), body: text("Public feature and pricing pages explain the offer. The workspace completes the task. A visible billing entry lets existing customers manage their plan without going through the public sales journey again.", "公开功能页与价格页说明购买内容，工作台负责完成任务。清楚的账单入口让已有用户直接管理套餐，无需重新经过介绍和购买流程。") }
    ],
    limitations: [text("The template does not decide your pricing, included seats, usage limits, or feature access policy.", "模板不会替你决定定价、席位数、用量限制或功能权限策略。"), text("Payment configuration and your core product must be ready before this becomes a service you can sell.", "支付配置与核心业务准备完成后，才能成为面向客户出售的产品。")],
    faq: [
      { question: text("Can a subscription also use credits?", "订阅可以同时使用积分吗？"), answer: text("Both capabilities are available. You must define whether a plan grants credits, how often they are granted, and what happens when a plan changes; do not imply such rules until you implement them.", "两种能力都可使用。套餐是否赠送积分、多久发放一次、套餐变更时如何处理，需要你定义并实现后再向用户说明。") },
      { question: text("Is the example billing page already a complete product?", "示例账单页面就是完整产品了吗？"), answer: text("It is a working billing foundation. Customers still need a useful service, clear entitlements, support, and the policies you will operate.", "它是可接入的计费基础。用户仍需要有用的服务、明确权益、支持渠道和实际执行的规则。") }
    ],
    related: [link("/features/subscriptions", "Review subscription capabilities", "查看订阅能力"), link("/pricing", "Inspect the sample offers", "查看示例商品"), link("/docs/launch", "Review launch requirements", "查看上线准备")],
    sources: [link("/docs/payments", "Payment guide", "支付指南"), link("/docs/launch", "Launch guide", "上线指南")]
  },
  {
    kind: "use-cases", path: "/use-cases/usage-based-products", updatedAt,
    title: text("Build a product that charges prepaid credits per action", "搭建按操作消耗预付积分的产品"),
    description: text("Use a credit balance for individually priced actions, with clear confirmation, transaction records, and defined retry behavior.", "为单次收费操作接入积分余额、价格确认、流水记录和清楚的重试规则。"),
    intro: text("Prepaid credits fit products where a customer buys a balance and spends it on individual results. Connect your own paid action to the shared credit service.", "用户先获得积分，再按次购买具体结果，适合预付积分模式。可以将自己的收费操作接入共享积分服务。"),
    primaryCta: link("/examples", "Explore account services", "体验账户服务"),
    facts: [
      { label: text("Billing model", "计费模型"), value: text("Prepaid credits per operation", "预付积分，按操作消费") },
      { label: text("Customer confirmation", "用户确认"), value: text("Cost before the action", "操作前显示积分价格") },
      { label: text("Included example", "已有示例"), value: text("A local Markdown export", "本地 Markdown 导出") }
    ],
    steps: [
      { title: text("Price a meaningful unit", "为明确的业务单位定价"), body: text("Choose what counts as one paid operation, such as one export. Calculate costs on the server and explain the result before confirmation.", "明确什么算一次付费操作，例如一次导出。由服务端决定消耗，在确认前说明用户将获得什么。") },
      { title: text("Define success and retry", "定义成功与重试"), body: text("Store a stable operation identifier and the result. Repeating that operation should return the same result instead of treating a retry as a new purchase.", "保存稳定的操作编号与结果。同一次操作重试应返回原结果，避免把重试当成新的购买。") },
      { title: text("Expose the record", "提供消费记录"), body: text("Show the remaining balance and a ledger entry so customers can connect a delivered result to its credit cost.", "显示剩余余额和积分流水，让用户把实际收到的结果与本次消耗对应起来。") }
    ],
    sections: [
      { title: text("Prepaid credits are different from monthly usage invoices", "预付积分与月末按量出账有所区别"), body: text("The starter accounts for a balance that customers spend. It does not aggregate metered usage and issue an invoice after the billing period. Choose that distinction deliberately when designing your offer.", "模板管理的是用户可消费的积分余额，没有在计费周期结束后汇总用量并出账的流程。设计收费方式时，需要先确定采用哪种模式。") },
      { title: text("For AI jobs, add a real task lifecycle", "接入 AI 任务时，补齐任务流程"), body: text("An external generation request cannot be rolled back by a database transaction. Add durable job state, provider retries, delivery tracking, and a compensation policy before charging for those jobs.", "数据库事务无法撤回外部生成请求。对这类任务收费前，需要补齐持久化任务状态、服务商重试、结果交付与补偿规则。") }
    ],
    limitations: [text("The shipped service does not include credit reservation, later settlement, partial refunds, or a job queue for external providers.", "已有服务不包含积分预冻结、事后结算、部分退款或外部服务任务队列。"), text("A credit purchase still needs configured payment services. The isolated preview can demonstrate spending, but not checkout.", "购买积分需要已配置的支付服务；隔离预览可以体验消费流程，无法完成支付结账。")],
    faq: [
      { question: text("Is an AI generation service included?", "模板已经接入 AI 生成服务了吗？"), answer: text("A server-side model adapter is included. It makes stateless text requests, without a default chat UI, conversation storage or charging policy. Your product defines its conversations, tasks and credit usage.", "模板包含服务端模型适配器，支持无状态文本请求；默认不提供聊天界面、对话存储或收费规则。对话、任务与积分消费方式由具体产品定义。") },
      { question: text("What happens when the balance is insufficient?", "余额不足时会发生什么？"), answer: text("The credit service rejects the consumption. Your product should give the customer a clear way to review their balance or purchase more credits when checkout is available.", "积分服务会拒绝消费。你的产品应给用户清楚的余额查看入口，并在支付可用时提供购买积分的路径。") }
    ],
    related: [link("/features/credits", "Understand the credit service", "了解积分服务"), link("/use-cases/paid-exports", "Apply this to file exports", "用于文件导出"), link("/docs/payments", "Connect credit checkout", "连接积分购买")],
    sources: [creditGuide, exportSource]
  },
  {
    kind: "use-cases", path: "/use-cases/paid-exports", updatedAt: "2026-10-15",
    title: text("Build exports that charge credits", "开发按积分计费的导出功能"),
    description: text("Connect your own generated files to authentication, credit accounting and recorded consumption.", "将自己的文件生成业务接入认证、积分记账与消费记录。"),
    intro: text("The template supplies shared account and credit services. You build the export screen, generate the file and define what a successful paid result means.", "模板提供通用账户与积分服务。导出界面、文件生成，以及什么情况算完成付费交付，由自己的产品实现。"),
    primaryCta: link("/docs/features", "Read the integration guide", "阅读接入指南"),
    facts: [
      { label: text("Reusable", "可复用能力"), value: text("Authentication and credit ledger", "用户认证与积分账本") },
      { label: text("Product-owned", "产品自行实现"), value: text("Export interface and generator", "导出界面与生成器") },
      { label: text("Customer records", "用户记录"), value: text("Credit history", "积分明细") }
    ],
    steps: [
      { title: text("Define the output", "定义交付结果"), body: text("Choose the file format, generation rules and server-side credit cost for your own product.", "为自己的产品选择文件格式、生成规则与服务端积分价格。") },
      { title: text("Review before charging", "扣费前确认"), body: text("Show the selected result and cost before the customer confirms. Check resource ownership on the server.", "在用户确认前显示交付结果与费用，并在服务端检查资源归属。") },
      { title: text("Deliver and record", "交付并记录"), body: text("Return the generated file and record its credit consumption. Preserve an operation identifier when retrying.", "返回生成文件并记录积分消费，重试时复用同一操作编号。") }
    ],
    sections: [
      { title: text("Use the shared accounting boundary", "复用记账边界"), body: text("For local database work, store the result and credit consumption in the same transaction. Backend export code is an implementation reference, without a default customer interface.", "本地数据库操作可在同一事务中保存结果并记录积分消费。后端导出代码保留为实现参考，不提供默认用户界面。") },
      { title: text("Own external generation", "实现自己的生成流程"), body: text("External generation needs its own task lifecycle and compensation rules. The template does not supply a report generator or ready-made export product.", "外部生成服务需要独立的任务流程与补偿规则。模板不提供报告生成器或现成的导出产品。") }
    ],
    limitations: [text("Export screens, file formats and delivery rules belong to your product.", "导出界面、文件格式与交付规则由产品实现。"), text("File access requires server-side authorization and ownership checks.", "文件访问需要服务端认证与资源归属检查。")],
    faq: [
      { question: text("Does the starter generate PDF or Word files?", "模板会生成 PDF 或 Word 文件吗？"), answer: text("No. Implement and connect a generator for the formats your product needs.", "不会，需要为产品所需格式实现并接入生成器。") },
      { question: text("Where is the cost defined?", "在哪里定义费用？"), answer: text("Your business service defines the cost and confirms it on the server before recording consumption.", "由业务服务定义费用，并在服务端确认后记录消费。") }
    ],
    related: [link("/features/credits", "Review credit accounting", "了解积分记账"), link("/docs/launch", "Prepare your own product", "准备自己的产品")],
    sources: [exportSource, creditGuide]
  },
  {
    kind: "compare", path: "/compare/starter-vs-from-scratch", updatedAt,
    title: text("Use the starter or build from scratch?", "使用模板，还是从头搭建？"),
    description: text("Compare the included Next.js + Go foundations, customization work, and maintenance responsibilities before choosing how to build.", "从已有基础能力、定制工作与维护责任三个方面，比较使用 Next.js + Go 模板和从头开发。"),
    intro: text("The starter is a practical fit when you want Next.js, Go, and its account and billing patterns. Building from scratch gives you a clean starting point when those choices do not fit your product.", "当 Next.js、Go 与模板已有的账户计费方式符合需求时，可以从模板开始。如果这些选择不适合业务，从头搭建能让你按自己的约束设计。"),
    primaryCta: link("/docs/features", "Inspect what is included", "查看已包含的能力"),
    facts: [
      { label: text("Source price", "源码价格"), value: text("Starter: free to obtain", "模板：免费获取") },
      { label: text("Technology", "技术选择"), value: text("Starter: Next.js + Go", "模板：Next.js + Go") },
      { label: text("Ownership", "维护责任"), value: text("Both: you operate your product", "两种方式都由你运营产品") }
    ],
    steps: [
      { title: text("Compare your real requirements", "先对照真实需求"), body: text("List the authentication, billing, data, and operational workflows you need. Match them to the supplied examples rather than to a feature count.", "列出所需的登录、计费、数据与运营流程，再与模板示例逐项对照。") },
      { title: text("Run one complete journey", "跑通一个完整流程"), body: text("Try account creation, subscription management and referral rewards. Read the source for the business rules you expect to change.", "体验注册、订阅管理与推荐奖励，阅读预计需要修改的业务规则源码。") },
      { title: text("Choose what you can maintain", "选择能够持续维护的方案"), body: text("Consider familiarity with the stack, custom integrations, upgrades, and test coverage. There is no universal time or cost saving for every team.", "考虑技术栈熟悉度、定制集成、升级与测试覆盖。不同团队的开发时间和成本没有统一答案。") }
    ],
    sections: [
      { title: text("Authentication and billing", "登录与计费"), body: text("With the starter, adapt connected email login, account pages, Stripe billing, and credit examples. From scratch, you select and implement those flows yourself. In either case, configure your service accounts and verify their behavior.", "使用模板时，可以调整已接通的邮箱登录、账户页、Stripe 计费和积分示例；从头搭建时，需要自行选择并实现这些流程。两种方式都需要配置服务账号并验收行为。") },
      { title: text("Product model and design", "产品模型与界面"), body: text("The starter provides a public site, customer workspace, and administrator console. You replace its content and add the core feature. From scratch, you control those boundaries from the first design decision and build their foundations too.", "模板提供公开网站、用户工作台与管理后台，你需要替换内容并增加核心业务。从头搭建时，可以从最初就决定这些边界，同时也需要实现它们的基础。") },
      { title: text("Ongoing costs and responsibility", "长期费用与责任"), body: text("Free source does not include hosting, a managed database, email delivery, payment fees, or an operations team. Both approaches leave service operation, dependency updates, customer support, and product rules with you.", "免费源码不包含托管、数据库、邮件发送、支付手续费或运营团队。两种方式都需要你负责服务运营、依赖更新、客户支持与业务规则。") }
    ],
    limitations: [text("This comparison describes this repository and the choice to build your own foundations. It is not a benchmark of other starter kits.", "本页比较的是当前仓库与自行搭建基础能力两种方式，不是对其他模板的测试排名。"), text("If you require another stack, enterprise identity, or a different billing model, assess the adaptation work before choosing the starter.", "如果需要其他技术栈、企业身份系统或不同计费模型，应先评估改造工作再选择。")],
    faq: [
      { question: text("Does the starter include hosting?", "模板包含托管服务吗？"), answer: text("No. You obtain the source and deploy your own frontend, API, and database with providers you choose.", "不包含。你获得源码，再自行选择服务商部署前端、API 与数据库。") },
      { question: text("Can I replace the example business?", "可以替换示例业务吗？"), answer: text("Yes. Notes and exports demonstrate how to connect a user action to ownership and charging. Your own business feature belongs in the product template.", "可以。笔记与导出演示了如何连接用户操作、资源归属和收费，你可以在产品模板中实现自己的业务。") }
    ],
    related: [link("/features", "Browse included foundations", "浏览基础能力"), link("/examples", "Explore account services", "体验账户服务"), link("/about", "Understand the project", "了解项目定位")],
    sources: [source, frontendReadme, link("/docs/launch", "What remains before launch", "上线前还需要什么")]
  },

];

export const aboutPage: MarketingUtilityPage = {
  path: "/about", updatedAt,
  title: text("A free starting point for your Next.js + Go SaaS", "为 Next.js + Go SaaS 提供免费的开发起点"),
  description: text("Learn what the go-modules starter includes, where its source lives, and what you own when you build a product with it.", "了解 go-modules 模板包含什么、源码在哪里，以及用它开发产品时由你负责哪些部分。"),
  intro: text("This website demonstrates the SaaS templates in the go-modules repository. You can obtain the source for free, run it yourself, and build your own product on the supplied workflows.", "这个网站演示 go-modules 仓库中的 SaaS 模板。你可以免费获取源码，自行运行，并基于已有流程开发自己的产品。"),
  sections: [
    { title: text("What is included", "模板已经提供什么"), body: text("A Next.js public website and account center connect to a Go API. Shared capabilities cover login, Stripe subscriptions, credits, referrals, AI integration and administrator operations. Product-specific screens and workflows are supplied by your own application.", "Next.js 公开网站与账户中心连接 Go API，公共能力包括登录、Stripe 订阅、积分、邀请、AI 接入与管理后台。具体业务的界面与流程由你的产品接入。") },
    { title: text("What you build and operate", "哪些部分由你开发与运营"), body: text("You add the core service, define customer entitlements, configure your providers, and operate the deployment. Sample prices describe a possible customer offer; they are not a charge for the starter source.", "你需要添加核心业务、定义客户权益、配置服务商并负责部署运营。示例价格展示面向客户的收费方式，不是获取模板源码的费用。") },
    { title: text("Where to verify details", "在哪里核对实现"), body: text("The public source repository and its setup documentation are the reference for these pages. Check the implementation before adopting a workflow, and use the project's GitHub issue tracker for reproducible code problems.", "这些介绍以公开源码仓库和配置文档为依据。采用某个流程前可以查看实现，能够复现的代码问题可以提交到项目的 GitHub issue。") }
  ],
  related: [link("/docs/overview", "Read the starting guide", "阅读入门指南"), link("/compare/starter-vs-from-scratch", "Assess whether it fits", "判断是否适合你"), link("/contact", "Find contact information", "查看联系信息")],
  sources: [source, link(`${repository}/issues`, "Project issue tracker", "项目问题跟踪")]
};

export const resourcesPage: MarketingUtilityPage = {
  path: "/resources", updatedAt,
  title: siteLinks.resources.label,
  description: text("Find starter capabilities, business patterns, a working example, configuration guides, and source references in one place.", "集中查找模板能力、业务场景、可用示例、配置指南和源码依据。"),
  intro: text("Choose the task you are working on. Start with the product journey, then open the matching setup guide when you are ready to connect your own services.", "从正在处理的任务出发，先理解产品流程，准备接入自己的服务时再打开对应配置指南。"),
  sections: [
    { title: text("Understand the starting point", "了解开发起点"), body: text("Browse the authentication, subscription, and credit pages to see what is already connected. The comparison explains when adapting the template is a reasonable fit.", "通过登录、订阅与积分页面了解已有能力，再通过比较页判断模板是否适合自己的业务。") },
    { title: text("Choose and try a customer workflow", "选择并体验用户流程"), body: text("Open the account center to try sign-in, subscription management, credit history and referral rewards.", "进入账户中心，体验登录、订阅管理、积分明细和推荐奖励。") },
    { title: text("Connect your services and prepare to launch", "接入服务并准备上线"), body: text("Use the setup guides for domains, authentication, payments, and deployment preparation. The source repository contains the implementation and more detailed developer references.", "使用配置指南连接域名、登录与支付，再完成上线准备。源码仓库提供具体实现与更详细的开发参考。") }
  ],
  related: [
    link("/features", "Explore capabilities", "浏览基础能力"),
    link("/use-cases", "Choose a business pattern", "选择业务场景"),
    link("/examples", "Try a working example", "体验可用示例"),
    link("/compare", "Compare starting points", "比较开发方式"),
    link("/docs", "Open setup guides", "打开配置指南"),
    link("/blog", "Read practical articles", "阅读实践文章"),
    link("/updates", "Read project updates", "查看项目更新")
  ],
  sources: [source, frontendReadme]
};

export const marketingRoutes: Pick<MarketingPage, "path" | "title" | "description">[] = [
  ...marketingCategories,
  ...marketingPages,
  aboutPage,
  resourcesPage
].map(({ path, title, description }) => ({ path, title, description }));

export function getMarketingPage(path: string): MarketingPage | undefined {
  return marketingPages.find((page) => page.path === path);
}

export const findMarketingPage = getMarketingPage;

export function getMarketingCategory(kind: MarketingKind): MarketingCategory | undefined {
  return marketingCategories.find((category) => category.kind === kind);
}
