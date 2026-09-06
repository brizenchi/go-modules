# SaaS 运营功能使用指南

这套免费模板包含产品官网、登录注册、Stripe 订阅、邀请推荐，以及可修改的付费业务示例。
演示站可以使用 Stripe 测试模式；账号、邀请关系、笔记、文件和积分记录都由真实后端保存。
`NEXT_PUBLIC_DEMO_MODE` 只控制演示文案，支付是否涉及真实资金由后端 Stripe 密钥和价格环境决定。

## 页面与使用路径

| 页面 | 用途 |
| --- | --- |
| `/admin` | 基础运营概况：注册用户、有效订阅、邀请状态数量 |
| `/admin/users`、`/admin/subscriptions`、`/admin/orders` | 搜索用户、订阅快照与保存的支付记录 |
| `/admin/referrals` | 查询邀请状态；带原因重试已激活邀请的原奖励 |
| `/admin/credits` | 按用户查询流水，发放有到期时间的积分，退回原消费 |
| `/admin/settings` | 修改品牌、介绍、支持邮箱/HTTPS 链接、每次导出积分价格 |
| `/admin/audit` | 查询设置与奖励核对的操作人、原因和结果；积分调整在积分流水中 |
| `/credits` | 个人可用积分、未来 30 天到期积分和完整流水 |
| `/notes` | 免费保存笔记，确认积分价格后生成 Markdown 文件 |
| `/files` | 上传、预览与下载属于当前账号的图片 |
| `/blog`、`/updates`、`/contact` | 双语文章搜索、更新记录与配置后的支持入口 |

后台数量属于**基础运营指标**。DAU、MAU、留存和项目分析仍需根据自己的产品定义活跃行为并接入数据。
支付列表从已校验并保存的支付事件生成，金额缺失时显示未知，不据此推算收入。它不是完整财务报表。

## 配置

1. 在后端设置 `APP_AUTH_ADMIN_EMAILS=owner@example.com`（多个邮箱逗号分隔），用该邮箱登录后访问 `/admin`。身份和权限由后端检查，前端隐藏入口不代替权限校验。
2. 使用 `/admin/settings` 保存公开配置并填写修改原因。导出默认消耗 1 积分，允许 1–1,000,000 的整数。登录、Stripe、Resend、存储密钥继续使用服务端环境变量。
3. 开启注册赠送可设置 `APP_HOST_SIGNUP_CREDITS`。邀请奖励由 `APP_REFERRAL_ACTIVATION_REWARD` 与奖励期限配置决定；管理员核对按钮不会把待激活邀请变成已激活。
4. 本地图片上传配置 `APP_HOST_UPLOADS_ENABLED=true`、`APP_HOST_UPLOADS_PROVIDER=local` 和 `APP_HOST_UPLOADS_DIRECTORY=./var/uploads`。支持 JPEG、PNG、GIF、WebP，每张最多 5 MiB；SVG 不支持。
5. 生产多实例部署使用 `provider=s3`，填写 bucket、region 和需要的 endpoint；R2 使用账户 S3 endpoint 与 `region=auto`。桶必须保持私有。服务端携带凭据读写，前端通过带登录凭据的 API 获取图片。切换存储不会自动搬迁已有文件。
6. 前端 `NEXT_PUBLIC_APP_URL` 用于 canonical 与 sitemap，`NEXT_PUBLIC_APP_NAME` 用于构建时品牌和 SEO。后台品牌设置会更新页面导航；SEO 和文章正文修改后需要重新构建。支持信息未配置时联系页会明确显示暂未开放。

管理员设置与奖励核对请求使用 `Idempotency-Key`，后端 CORS 已允许该请求头。自有网关也需要保留它，并允许配置的前端 origin。

公共文章、更新记录和隐私/条款模板在前端本地内容文件中编辑，见 [内容编辑指南](../quickstart-nextjs/CONTENT.md)。上线前填写真实经营者、支持渠道和实际数据政策。

## 现有数据库升级

升级文件已提供，**没有自动执行**：

1. [20260906_credit_ledger.sql](migrations/20260906_credit_ledger.sql)：保留当前余额、建立积分批次与流水、保留历史支付/邀请去重记录。
2. [20260906_operations.sql](migrations/20260906_operations.sql)：运营设置、审计和上传元数据表。

由维护者审阅并选择维护窗口，停止应用写入并备份后，按上述顺序对 PostgreSQL 显式执行，再发布对应版本。原有启动流程会自动补表，但不会迁移旧余额；未迁移的旧账号仍可查看原余额，积分操作会返回 `503 credit_ledger_migration_required`。不要把启动 AutoMigrate 当成旧余额升级。

新版 `user.Repository.Create` 会为新账号初始化账本。独立扩展用户创建代码时必须保留该入口；直接 `db.Create(&user.User{})` 会跳过积分初始化。

## 验收

- 用户创建笔记，确认价格后导出；余额减少相应积分，个人流水出现消费。原请求失败重试复用同一请求键，不重复扣费。已生成文件在当前页面可重复下载；新请求键代表新一次付费导出。
- 管理员改变导出价格后，使用旧价格确认会返回 `price_changed` 且不扣费；刷新价格后重新确认。
- 管理员按原消费流水退回一次；再次退回或退回发放记录应失败。退款恢复给原消费人；退回积分永久有效。
- 普通用户不能访问任何 `/admin` API；A 账号不能读取 B 账号的笔记或图片。
- 邀请验收使用两个不同账号：通过邀请码注册形成待激活关系，首次符合条件的付费订阅激活并奖励邀请人；后台重试奖励不会重复发放。免费试用、续费、重复回调都不能重复产生奖励。
- Stripe 测试验收与 Resend 实际送达仍使用自己的测试环境，见 [配置与上线指南](../../docs/SETUP_ZH.md)。自动化测试只使用独立 SQLite、临时目录和假服务，不触碰部署数据库。

完整后端契约：[积分与导出](internal/feature/credits/README.md)、[运营与存储](internal/feature/operations/README.md)。
