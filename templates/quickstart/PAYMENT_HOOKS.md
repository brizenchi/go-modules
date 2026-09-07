# Stripe 退款与拒付 hook

业务入口位于 `internal/bootstrap/host_hooks.go`，由 `subscriptions.go` 注册。默认全部返回
`nil`；没有自动退款、扣积分、回收权限、取消订阅或发邮件的 SaaS 策略。

| Stripe 事件 | 模块事件 / 宿主 hook | 载荷含义 |
| --- | --- | --- |
| `refund.created`、`refund.updated`、`refund.failed`、兼容 `charge.refund.updated` | `KindRefundUpdated` / `onRefundUpdated` | 单笔 refund 的 ID、状态、金额、币种、原因、失败原因及 charge/payment intent ID |
| `charge.refunded` | `KindChargeRefunded` / `onChargeRefunded` | charge 原金额、累计 `AmountRefunded`、是否全部退回及 customer/payment intent ID |
| `charge.dispute.created`、`updated`、`closed`、`funds_withdrawn`、`funds_reinstated` | `KindDisputeUpdated` / `onDisputeUpdated` | dispute ID、原事件类型、状态、金额、币种、原因及 charge/payment intent ID |

在 Stripe Dashboard 的现有 webhook endpoint 中按需勾选这些事件。适配器先验证签名，
再将 provider 对象翻译成 typed event；不为获取用户信息额外调用 Stripe API。
`ProviderEventType` 区分 lifecycle 通知，不会把所有通知都当成“退款成功”或“败诉”。
原始字段按 [Stripe 事件文档](https://docs.stripe.com/api/events/types)、
[Refund](https://docs.stripe.com/api/refunds/object) 和
[Dispute](https://docs.stripe.com/api/disputes/object) 定义传递。

## 具体 SaaS 实现时的约定

- 所有金额都是币种最小单位。单笔 refund 的 `Amount` 与 charge 的累计 `AmountRefunded`
  不是同一口径；不能把两类通知相加，也不能把多次更新当作多次退款。
- Envelope 提供 `Provider`、`ProviderEventID`、`OccurredAt`。金额和状态是事件发生时的
  快照，不保证按时间顺序送达；持久化业务状态时需要防止旧事件覆盖新状态。
- 相同事件处理成功后会跳过后续重复投递。hook 返回 error 时事件不会标记完成，
  Stripe 可重试；并发投递或中途崩溃仍可能重复调用，因此每个业务副作用需持久化幂等键，
  如 provider + event ID + action。不同 event ID 描述同一 refund 的更新还需按 refund ID
  维护业务状态，不能只按 event ID 累加扣款。
- refund/dispute 通常没有 customer 或用户 metadata；`Envelope.UserID` 可能为空。
  可依据载荷中的 charge/payment intent ID 在 SaaS 自己的订单表中定位用户。需要用户
  但暂时无法定位时应返回 error 或持久化待处理任务，不能静默丢弃业务动作。
- 如需取消订阅或回收积分，由对应 SaaS 决定部分退款、失败退款、争议中、胜诉/败诉等
  各状态的策略。模板不会改写现有订阅快照和积分余额。

默认 no-op hook 会把有效事件标记为已处理。如果之后才实现业务逻辑，需要自行安排
历史事件回放，不能指望已处理事件再次投递就自动补齐历史副作用。
