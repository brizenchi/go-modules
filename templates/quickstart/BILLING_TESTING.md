# 支付生命周期与验收指南

核对日期：2026-10-12。本文记录已实现的默认规则和上线前需要执行的 Stripe Sandbox 验收。
本次已执行本地自动化测试和 Stripe Test Clock 的真实订阅 API 验收。
浏览器 Checkout → 本站 Webhook → 用户页面的完整链路，以及 credits / Lifetime 的实际购买仍需按清单验收。

## 1. 买完在哪里看

| 商品 | 用户入口 | 当前记录方式 |
| --- | --- | --- |
| Starter / Pro / Premium 月付、年付 | `/billing` 当前订阅、账期、支付方式、账单记录 | `billing_subscriptions` 保存订阅快照；查询非终态订阅时还会向 Stripe 刷新 |
| credits 包 | `/credits` 余额、流水；`/account` 可用积分 | `onCreditsPurchased` 调用积分仓库；按实际购买数量 × 每包积分入账 |
| Lifetime | `/billing` 显示终身权益 | 一次性支付后写入 lifetime / active，无周期订阅 ID，无续费或取消续费 |
| 管理员核对 | `/admin/orders`、`/admin/subscriptions`、`/admin/credits` | 支付记录从 Webhook 事件汇总；不是完整的订单履约系统 |

`/billing` 的“购买记录与会员权益”覆盖订阅、credits 和 Lifetime，来自经过验证的支付事件。
发票列表来自 Stripe Invoice API；新建的一次性 Checkout 已开启 `invoice_creation`。
历史订单不一定有发票，购买记录仍可查询。积分余额与逐笔账本从该页进入 `/credits`。
“已处理”表示事件监听器已成功处理，并不代表所有退款或自定义业务都自动完成。

购买积分不会升级会员；购买订阅或 Lifetime 也不会默认按月赠送积分。
模板已提供后端会员策略与中间件，产品配额和哪些接口收费由宿主接入；见下方接入示例。

## 2. 当前变更规则

来源：共享模块 `modules/billing/app/subscription.go` 的 `resolveChangeMode`，
以及 `modules/billing/adapter/stripe/provider.go`。

| 操作 | 当前处理 |
| --- | --- |
| 同周期升级，例如 Starter 月付 → Pro 月付 | 立即按比例结算；使用 `pending_if_incomplete`，需要付款时付款成功才应用升级 |
| 同周期降级，例如 Premium 年付 → Pro 年付 | 创建或更新 Subscription Schedule，下周期生效 |
| 同档或升档的月付 → 年付 | 立即切换并重置计费周期；任何降档均下周期生效 |
| 年付 → 月付 | 下周期切换 |
| 主动取消续费 | 默认周期末取消；当前周期不立即终止 |
| 3 天后取消 | 保留兼容 API，默认用户页面已移除；自定义产品才使用 |
| 撤销尚未生效的取消 | 清除 `cancel_at_period_end` 和 `cancel_at` |
| 已经取消后重新购买 | 新建 Checkout，不是恢复已结束的旧订阅 |
| 周期订阅用户购买 Lifetime | 当前拒绝重叠购买，必须等周期订阅真正结束 |
| Lifetime 用户重复购买会员 | 当前拒绝；仍可买 credits |

**降档优先延期。** Premium 月付 → Starter 年付现在下周期生效；年付 → 月付也下周期生效。
待付款升级保留旧套餐，页面给出补付款入口；待生效变更展示目标套餐和日期，可以撤销。
已有待付款或预约变更时，新的变更和退订返回 409 / `subscription_change_pending`：
先完成付款或撤销预约，再操作。取消续费后要先恢复续费，才允许变更套餐。

## 3. 到期、退订和退款分别是什么

- 自动续费订阅：账期结束后尝试扣下一期费用，不会因为到了 `period_end` 就自然退订。
- 用户关闭续费：保留本期权益，到取消生效时间终止；此前可恢复续费。
- 扣款失败：进入催收或待补付款流程，最终取消还是 unpaid 由 Stripe 重试设置等决定；
  产品应明确宽限期和权限规则。
- 固定期限会员：如果产品希望“只买一个月、不自动续费”，需要另行实现固定期限权益，
  或在创建周期订阅后显式安排周期末取消；当前没有默认的“不续费月卡”商品。
- Lifetime：没有正常到期；退款或拒付后是否撤销权益，需要实现业务策略。
- 取消续费不会自动退款，也不应顺手清空用户单独购买的 credits。

Stripe 的 [取消生命周期](https://docs.stripe.com/billing/subscriptions/cancel) 和
[订阅 Webhook](https://docs.stripe.com/billing/subscriptions/webhooks) 是联调依据。
应用应处理 Stripe 的最终状态，不能只靠浏览器回跳或一个按时间运行的取消任务。

## 4. 已实现的接口、策略与恢复任务

| 能力 | 入口 |
| --- | --- |
| 当前套餐、待付款、预约套餐、取消生效时间 | `GET /api/v1/stripe/subscription` |
| 撤销预约变更（保留当前订阅） | `POST /api/v1/stripe/subscription/scheduled-change/cancel` |
| 当前用户的全部购买记录 | `GET /api/v1/billing/purchases?page=1&limit=10` |
| 后端计算的实际会员权益 | `GET /api/v1/account/membership` |
| 自动补偿、失败重试、快照刷新 | `internal/feature/billingrecovery/recovery.go`，在 `host_jobs.go` 注册 |

会员策略在 `internal/feature/membership/membership.go` 的 `Evaluate`：

- Starter / Pro / Premium 的 active、trialing、canceling 在有效期内允许使用。
- 到期按服务器时间立即失效，即使取消回调尚未送达。
- past_due / unpaid / incomplete / canceled 和未知状态不授予会员权限；默认无欠费宽限期。
- 缺少周期结束日期时不放行，补偿任务从 Stripe 补齐后恢复。
- Lifetime 的 active 长期有效，对应最高档；购买积分不会改变会员等级。

在 `internal/http/host_routes.go` 为自己的付费功能挂中间件，例如：

```go
g.User.GET("/premium-export", membership.Require(deps.DB, "pro"), exportHandler)
```

只判断前端套餐名称没有访问控制作用。现有 notes 等免费功能没有被强行改成收费。
需要宽限期、额外赠送会员或套餐额度时修改宿主策略和对应测试；邀请回调仍默认空实现。
退款／拒付的自动撤权仍由 `PAYMENT_HOOKS.md` 中的产品回调决定。

补偿任务每分钟运行，单次最多 45 秒，数据库租约 2 分钟，支持多实例。
首次回看最近 24 小时；之后使用固定时间窗口和逐事件游标，窗口间重叠 1 分钟。
每轮最多查询 100 个 provider 事件、轮转重试 20 条未处理事件、刷新 10 个订阅快照。
任务通过 Stripe API 取回原始事件，不允许拿未签名的客户端 JSON 重放；监听器仍需幂等。
有解析/连接错误时保留游标，有已入库的监听器错误时交给重试队列；失败通过 job ERROR 日志暴露。
`billing_recovery_states` 在启动时 AutoMigrate。事件存储和租约表必须持久化。

Stripe [事件 API](https://docs.stripe.com/api/events/list) 仅支持最近 30 天，长期停机、
初次接管 24 小时以前的历史漏单需运维对账，不能宣称会无限回溯。
外部告警通知需要部署平台订阅 `billing-recovery` 错误日志；模板不会主动给第三方发送消息。
Webhook 解析同时支持旧版字段和新版 item 周期 / invoice parent / pricing 字段。
升级仍需用实际端点 API 版本完成 Sandbox 验收。

## 5. 测试分三层

### A. 离线回归：每次提交都跑

在原仓库根目录运行共享模块检查：

```bash
go vet ./modules/billing/...
go test -race -cover ./modules/billing/...
```

在后端目录运行宿主组合与权益测试：

```bash
go vet ./...
go test -race ./...
```

在前端目录运行：

```bash
npm run verify
```

复制后的项目只包含 backend / frontend；共享模块测试在其源码仓库运行。
普通 CI 测试使用模拟传输和本地数据库，不访问真实 Stripe。

现有测试涵盖 Checkout 并发与重试、重复 Webhook、乱序快照、付款前不发权益、
积分数量校验、终身权益保护、取消／恢复、跨计费周期策略，以及积分账本幂等。
它们不能证明你的 Stripe 端点配置、真实 Schedule 行为和产品接口权限已经正确。

新增回归覆盖：新版 Stripe 字段、待支付和预约快照、预约释放、跨周期降级、
变更冲突、会员到期边界、欠费与 Lifetime、购买记录越权、补偿租约和分页幂等。
本地通过不等同于 Stripe 端到端验收。

本次最终模块覆盖率：app 80.5%、Stripe adapter 76.0%、HTTP 52.9%、repo 63.5%、
eventbus 100%、domain 35.3%，装配包 0%。HTTP、repo、domain 等历史包仍低于
仓库 70% 要求，本次没有宣称消除了这些原有覆盖缺口。全量 race / vet 和前端 verify 已通过；
macOS 的 SQLite 链接过程有 LC_DYSYMTAB 警告，未导致测试失败。

### B. 可重复运行的 Stripe 订阅 API 验收

在源码仓库根目录运行（不会随 `go test` 自动联网）：

```bash
go run ./scripts/billing-sandbox -env-file templates/quickstart/.env -run
```

只接受 `sk_test_` 密钥，先检查三个档位的月付 Price 为测试模式，然后创建独立
Test Clock、Customer、Subscription。测试通过或失败都会尝试删除本次 Test Clock；
如报告 cleanup failed，用输出的测试 clock ID 清理。不要在真实模式运行。
文件需配置 `APP_BILLING_STRIPE_SECRET_KEY` 和三个档位的月付 Price；不打印密钥。

2026-10-12 已实测通过：首次订阅、升级预览、付款后升级、预约降级及读取目标、
撤销预约、周期末取消、恢复续费、Test Clock 推进后降级、拒付保留原档位、
补付款生效、Test Clock 到期取消和清理测试对象。
实测修复了预约创建参数冲突、恢复续费参数互斥、降级后 Schedule 导致无法退订三处问题。
脚本测的是支付适配器与真实 Stripe API，不是浏览器或本站 Webhook 的端到端测试。

### C. Stripe Sandbox：走真正的浏览器与 Webhook 链路

1. 独立测试数据库和 Stripe Sandbox；同一环境配置三个级别的月／年 Price、
   一个 credits 一次性 Price、一个 Lifetime 一次性 Price，以及 Customer Portal。
2. 在后台 `/admin/integrations` 核对实际生效配置。数据库覆盖可能优先于环境变量；
   更换配置后按模板要求重启，不能只改 `.env` 就假定切换成功。
3. 本地使用 CLI 转发真实测试事件：

```bash
stripe listen --forward-to localhost:8080/api/v1/stripe/webhook
```

把该监听会话的签名 secret 配给当前后端。Dashboard 端点的签名 secret 和 CLI
监听的 secret 不能混用。不要把密钥或原始支付载荷放进测试报告。
[Stripe CLI 文档](https://docs.stripe.com/cli) 描述了转发方式。

4. 按实现启用并观察 `checkout.session.completed`、异步成功／失败、过期，
   `invoice.paid` / `invoice.payment_failed`，`customer.subscription.created` /
   `updated` / `deleted`；验证退款、拒付时再启用对应事件。
5. 创建独立测试账号：升级、降级、取消、失败恢复、credits、Lifetime 分开测试，
   避免上一个场景留下的 Schedule 干扰下一个场景。
6. 成功卡可用 `4242 4242 4242 4242`；拒付和需要认证的卡从
   [官方测试卡列表](https://docs.stripe.com/testing) 选择。只用于 Stripe 测试环境。

从自己站点创建 Checkout，完成支付后核对本系统用户。`stripe trigger` 的默认
fixture 可用于测试接收器，但不一定有本项目的 user / customer / price 映射，
不能直接当成完整购买验收。

### D. Test Clocks：不用等一个月验证续费和到期

使用 Stripe Sandbox 的 Simulations / Test Clocks 推进订阅时间。
[官方说明](https://docs.stripe.com/billing/testing/test-clocks) 和
[详细操作](https://docs.stripe.com/billing/testing/test-clocks/api-advanced-usage)。

1. 为一个场景创建模拟时钟与测试 Customer，并用已配置 Price 创建订阅。
2. 必须把 Customer / Subscription 映射到本系统测试用户。测试初始化程序可通过
   `modules.Billing.Customers.SaveCustomerID(ctx, userID, "stripe", customerID)`
   绑定后再创建订阅；这是测试程序中的调用，不是公开 HTTP 接口。模板没有现成的
   Test Clock 一键初始化脚本，不要期待任意模拟 Customer 自动出现在你的账号里。
3. 在周期中点测试升级／预约降级，在周期结束后检查续费或取消。
4. 每次推进等待时钟 `ready`，再等待对应 Webhook 处理完成；二者是不同的完成条件。
   发票最终结算可能还需继续推进，不能见到 Invoice 创建就断言扣款成功。
5. 每个断言记录实际生效的 clock time，不用本机时间猜测 Stripe 应处于哪个状态。

时钟只推进 Stripe，不会改变 Go 的 `time.Now()`、本地积分到期或邀请激活期限。
当前“3 天后取消”和恢复资格也使用应用时间：推进时钟后测试这些路径，要让测试用
应用时钟与 Stripe 保持一致，或在推进前设置取消。不能把双方时钟不一致误判成业务结果。
产品权益和积分的时间边界应额外使用可注入 clock 做离线测试。

## 6. 必测矩阵

每行同时验收：**Stripe 状态 → 本地记录 → 用户页面 → 实际受限接口权限**。
“接口返回 200”“页面提示成功”“Webhook 接收 200”都不能独立代表交易履约完成。

| 场景 | 操作 | 关键断言 |
| --- | --- | --- |
| 所有订阅 SKU | 三档 × 月／年分别新购 | Price 与套餐匹配、正确账期、只建一份订阅、功能权限生效 |
| 支付中途退出 | 关闭 Checkout 或返回取消页 | 不提前升级、不加积分；若已扣款则以后续服务器确认结算，不能因回跳标记假定未付款 |
| 同周期升级 | Starter → Pro → Premium | 预览与实际费用核对；待支付保持旧权益，付款成功后升级 |
| 升级拒付／需认证 | 更换测试卡后升级 | 显示待支付／需操作，完成付款后再升级，不虚报成功 |
| 同周期降级 | Premium → Starter | 本期保留 Premium；刷新后仍能看到预约，下一期按新价续费 |
| 预约组合 | 降级后撤销／再升级／取消续费 | Stripe Schedule 与当前订阅一致，无遗留预约复活旧套餐或意外续费 |
| 跨周期 | 月→年、年→月，并同时升降套餐 | 验证上述周期优先规则、计费锚点及退款／抵扣，不只测同级切换 |
| 正常自动续费 | Test Clock 推进一个周期 | 只收一次款、延长权益；不重复赠送邀请奖励或错误增加 credits |
| 取消自动续费 | 周期中取消，推进到结束前／后 | 前可用、后不可用，不再续扣；单独购买的 credits 保留 |
| 撤销取消 | 结束前恢复，再推进 | 清除取消计划，下一期正常续费 |
| 已终止后回购 | 结束后尝试恢复和新购 | 旧订阅不能恢复，新 Checkout 可正常开通，无永久预订锁 |
| 3 天后取消 | 在模拟时间起点设置并推进 | 准确生效时间、最终账单／抵扣符合规则；重复请求不得无意延长取消时间 |
| 续费失败后恢复 | 续费拒付，更新卡并补款 | 催收状态与宽限期正确，恢复后不重复扣权益或赠额 |
| 续费永久失败 | 重试耗尽 | canceled / unpaid 对应产品权限终止，页面提供有效处理入口 |
| credits 购买 | 1、2、100 包；Checkout 中改数量 | 按实际结算数量入账、余额与流水一致；0／负数／101 拒绝 |
| credits 独立性 | 无订阅或取消会员后消费 | 按产品约定仍可使用已买积分；不足余额和并发消费不透支 |
| Lifetime | 新用户购买、刷新、重复购买 | 长期有效、无续费日期；重复会员购买拒绝，仍可买 credits |
| 订阅→Lifetime | 当前订阅未结束／已结束各尝试 | 未结束拒绝重叠购买，结束后允许；老订阅迟到事件不覆盖终身权益 |
| 延迟到账 | 延迟或暂停 Webhook，再恢复 | 显示待确认，恢复后补齐一次；不能仅凭 success URL 发权益 |
| 重复／并发通知 | 同 event 多次；同购买关联不同事件 | 每次购买履约一次；既防重复 event，也核对业务购买标识 |
| 乱序通知 | 新事件后再投递旧事件 | 不复活过期权益、不覆盖新套餐；业务回调也必须处理旧事件 |
| 退款／拒付 | credits、订阅、Lifetime 各测试 | 全额／部分规则明确；已消费积分如何处理；默认空回调不能算已完成撤权 |
| 越权 | 用户 A 访问／修改用户 B 的资源 | 无法读取他人账单、修改订阅或消费他人积分 |
| 版本契约 | 当前端点实际 API 版本的真实样本 | 账期、subscription ID、price、金额和数量解析完整，不默默丢字段 |

## 7. 保存证据与上线标准

每条用例记录以下字段，失败要附 request ID 和安全的错误分类，不附密钥、令牌或完整载荷：

```text
用例编号 / Git commit / Stripe API 版本 / 测试环境
测试用户 / customer ID / subscription 或 checkout session ID / event ID
操作前：套餐、状态、账期、积分余额
执行操作 / 模拟时间 / Stripe 结果
本地快照 / 积分流水 / 前端显示 / 受限接口实际结果
PASS 或 FAIL / 待修问题
```

建议上线门槛：所有 P0 通过；订阅六个 SKU、credits、Lifetime 的真实 Sandbox 购买
都闭环；至少完成一次自动续费、预约降级、取消到期、失败恢复、Webhook 重放和退款
策略验证。没有产品权限断言时，只能说“支付对接通过”，不能说“会员系统验收通过”。
