import type { ArticleSection } from "./articles";

export const documentation: ArticleSection[] = [
  { id: "overview", title: { en: "Start with the template", zh: "从模板开始" }, paragraphs: [
    { en: "The frontend combines a public website with a signed-in account center. The Go backend provides authentication, billing, and referral modules. Explore settings, subscriptions, and referrals directly, then adapt the product name, content, and business features to your own service.", zh: "前端包含公开网站和登录后的账户中心，Go 后端提供登录、计费和邀请模块。你可以直接体验设置、订阅与推荐流程，再根据自己的服务修改品牌、文案和业务功能。" },
    { en: "Start from the .env.example files in the frontend and backend template directories. Install frontend dependencies and follow the backend README for database and service setup. Keep private credentials in backend configuration.", zh: "从前后端模板目录中的 .env.example 开始配置。安装前端依赖，并按后端 README 完成数据库和服务设置。私密凭证保存在后端配置中。" }
  ] },
  { id: "domains", title: { en: "Connect the frontend and API", zh: "连接前端与 API" }, paragraphs: [
    { en: "Set NEXT_PUBLIC_APP_URL to your frontend origin and NEXT_PUBLIC_API_BASE_URL to the API base, including /api/v1. The backend must allow the frontend origin through CORS. OAuth callback URLs and payment return URLs need to match the deployed domains.", zh: "NEXT_PUBLIC_APP_URL 填前端地址，NEXT_PUBLIC_API_BASE_URL 填包含 /api/v1 的 API 基础地址。后端 CORS 需要允许前端域名。OAuth 回调和支付返回地址也要与部署域名保持一致。" },
    { en: "Public routes include the overview, pricing, documentation, blog, updates, and contact pages. Settings, subscriptions, and referrals are available directly from the account center after signing in.", zh: "公开路由包括总览、价格、文档、博客、更新记录和联系页面。登录后可直接从账户中心访问设置、订阅与推荐。" }
  ] },
  { id: "auth-email", title: { en: "Set up sign-in and Resend", zh: "配置登录和 Resend" }, paragraphs: [
    { en: "Configure the Resend API key and verified sender on the backend, together with your authentication providers. The frontend reads available sign-in methods from the API. Optional NEXT_PUBLIC_AUTH_EMAIL_ENABLED and NEXT_PUBLIC_AUTH_OAUTH_PROVIDERS values control frontend presentation.", zh: "在后端配置 Resend API key、已验证的发件人，以及准备开放的登录服务。前端从 API 读取可用登录方式；可选的 NEXT_PUBLIC_AUTH_EMAIL_ENABLED 和 NEXT_PUBLIC_AUTH_OAUTH_PROVIDERS 用于控制前端展示。" },
    { en: "Test a complete sign-in and sign-out with an address you control. Confirm that the session opens account settings and that a failed or expired login produces a useful message.", zh: "使用自己管理的邮箱验证完整的登录与退出流程。确认登录后会直接打开账户设置，失败或过期时也能看到明确提示。" }
  ] },
  { id: "payments", title: { en: "Configure plans and test payments", zh: "配置套餐并测试支付" }, paragraphs: [
    { en: "The template is free. Example plans belong to the product you build. Configure Stripe products, prices, backend credentials, and the signed webhook endpoint together. Public pricing copy must agree with the catalog; the subscription page starts Checkout and reads subscription and invoice records from the backend.", zh: "模板免费，示例套餐用于你基于模板构建的产品。一起配置 Stripe 商品、价格、后端凭证和签名 webhook 地址。公开价格说明应与商品配置一致，订阅页面负责发起 Checkout 并从后端读取订阅与发票。" },
    { en: "NEXT_PUBLIC_DEMO_MODE only controls the demo notice. Keep backend payment credentials in test mode while testing; changing the frontend flag does not switch the payment environment. Use the subscription page to complete the demo payment flow.", zh: "NEXT_PUBLIC_DEMO_MODE 只控制演示提示。测试时应使用后端支付测试凭证，修改前端标记不会切换支付环境。演示支付流程可直接在订阅页面完成。" }
  ] },
  { id: "invitations", title: { en: "Verify the invitation journey", zh: "验证邀请流程" }, paragraphs: [
    { en: "Copy an invitation link from the referral center, open it in a separate browser profile, and create a new account. Complete the qualifying subscription action in the test environment, then check the inviter’s records for the status and reward.", zh: "从推荐中心复制邀请链接，在独立的浏览器个人资料中打开并创建新账号。使用测试环境完成符合条件的订阅操作，再检查邀请人的状态和奖励记录。" },
    { en: "Publish the reward rules and activation window for your own service. Product invitation credits are separate from affiliate fees for third-party services recommended by the template author.", zh: "为自己的服务公布邀请奖励规则和激活期限。产品内邀请积分与模板作者推荐第三方服务获得的联盟佣金是不同机制。" }
  ] },
  { id: "content", title: { en: "Publish content and support details", zh: "完善内容和支持信息" }, paragraphs: [
    { en: "Edit content/articles.ts for bilingual blog posts, content/updates.ts for release notes, and content/policies.ts for your privacy and terms pages. CONTENT.md explains publishing and metadata settings. Configure a real support email or HTTPS help page in the site settings.", zh: "在 content/articles.ts 编辑中英文博客，在 content/updates.ts 编辑更新记录，在 content/policies.ts 完善隐私与条款。CONTENT.md 说明了发布流程和页面元信息配置。请在站点设置中配置真实的支持邮箱或 HTTPS 帮助页面。" },
    { en: "English is the default rendered language. The language switch changes the on-page copy at the same URL. Public pages include canonical links and sharing metadata; account and admin routes are excluded from the sitemap.", zh: "页面默认渲染英文。语言切换会在同一 URL 下更换页面文案。公开页面配置了规范链接和分享信息，账号及管理路由不会出现在站点地图中。" }
  ] }
];
