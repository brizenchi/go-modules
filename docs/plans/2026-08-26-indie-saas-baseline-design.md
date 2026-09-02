# 独立开发者 SaaS 基础底盘设计

## 目标

为同一个独立开发者维护的约 10 个 SaaS 提供一套最小、可靠、可复制的公共底盘。复制 quickstart 后，只配置数据库、Google、Stripe、Resend 和域名，再修改套餐文案、额度规则和产品功能。

## 取舍

默认只支持 Google/邮箱登录、Stripe Checkout 固定套餐与额度包、Customer Portal、Resend 邮件、PostgreSQL 和结构化 JSON 日志。GitHub、Brevo、邀请模块继续保留为可选能力。暂不引入微服务、消息队列、任意金额 PaymentIntent、复杂权限平台、Prometheus 或工作流引擎。

## 可靠性边界

- 访问令牌和 WebSocket Ticket 必须校验算法、issuer 和 token type，不能互相替代。
- Google 登录只接受 Google 明确标记为已验证的邮箱。
- Stripe Webhook 以 provider event ID 幂等；订阅事件先更新 billing 读模型，再执行宿主监听器；监听器失败时不能把 Webhook 标记为成功。
- 购买额度以 Stripe event ID 写入宿主额度流水，重复投递不会重复加额度。
- 邮箱验证码是持久化、单次使用、限频的；欢迎邮件允许作为非关键业务按宿主配置启用。
- 生产配置拒绝模板 Secret、debug 验证码、log 邮件发送器和不完整的 OAuth/Stripe 配置。

## 复制与发布

根仓库发布统一版本标签，quickstart 消费该标签。初始化脚本复制前后端并替换 Go module、服务名和前端包名。CI 同时验证根模块、workspace 模板，以及指定已发布版本的完全独立模板。

## 日志

保持一条请求一条结构化记录，公共字段固定为 `service`、`project`、`env`、`component`、`operation`、`outcome`、`request_id`、`trace_id`、`span_id`、`user_id`、`route`、`status_code`、`duration_ms` 和 `error_code`。不建设额外日志平台，只保证 10 个项目可以接入同一看板。

