# 文档入口

如果你要认真理解和使用这个项目，按下面顺序阅读。

## 第一次阅读

1. [整体架构](./ARCHITECTURE.md)
   - 为什么没有共享 `User` 和强制 `saascore`
   - `foundation`、`modules`、`template` 的边界
   - 注册、支付、邮件、邀请如何通过事件组合

2. [quickstart 开发指南](../templates/quickstart/README.md)
   - 启动链路
   - 文件放在哪里
   - 如何修改用户字段
   - 如何新增业务功能和监听器

3. [配置标准](./CONFIG_STANDARD.md)
   - YAML、`.env`、环境变量优先级
   - 模块和服务商如何启停
   - GitHub-only、无邮件、无支付等组合

## 开始部署时

4. [第三方服务配置](./SETUP_ZH.md)
   - PostgreSQL、JWT、Resend/Brevo
   - Google/GitHub OAuth
   - Stripe Webhook、生产环境变量、发布与启动验收

5. [模块接入指南](./INTEGRATION.md)
   - 在其他项目中直接使用单个模块
   - 实现 `UserStore` / `AccountLookup`
   - 订阅领域事件

## 维护架构时

- [ADR-0001：宿主拥有组合和用户模型](./adr/0001-template-owned-composition-and-user-schema.md)
- [版本管理](../VERSIONING.md)
- [本次重构实施计划](./plans/2026-08-08-template-owned-composition-refactor.md)

旧的 `SAASCORE_GUIDE.md` 不再适用：组合代码已经进入 quickstart，代码入口是
`templates/quickstart/internal/platform` 和 `internal/bootstrap/subscriptions.go`。
