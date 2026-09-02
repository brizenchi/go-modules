# billing 支付模块

提供支付服务商无关的结算、订阅变更、Webhook 幂等处理、订阅快照和支付事件。
当前自带 Stripe 适配器。

## 边界

billing 不导入宿主 User，不要求 `users.plan` 或 Stripe 字段。它只依赖：

- `port.Provider`：Stripe 或其他支付服务商；
- `port.CustomerStore`：支付客户关联；
- `port.AccountLookup`：从宿主读取最小的用户 ID 和邮箱；
- `port.BillingEventRepository`：Webhook 幂等记录；
- `port.SubscriptionRepository`：保存本地订阅快照；
- `port.UserResolver`：把 Webhook 提示映射回宿主用户 ID；
- 事件监听器：应用产品额度、发送邮件、激活邀请等。

## 模块拥有的表

- `billing_events`
- `billing_customers`
- `billing_subscriptions`

```go
if err := billingrepo.AutoMigrate(db); err != nil {
    return err
}
```

## 组装

```go
lookup := myAccountLookup{}

module := billing.New(billing.Deps{
    Provider:     stripeProvider,
    Bus:          eventbus.NewInProc(),
    Customers:    billingrepo.NewCustomerStore(db, lookup),
    EventRepo:    billingrepo.NewBillingEventRepo(db),
    Subscriptions: billingrepo.NewSubscriptionRepo(db),
    UserResolver: billingrepo.NewUserResolver(db, lookup),
    GetUserID:    currentUserID,
})
```

`AccountLookup` 示例：

```go
func (l lookup) FindBillingAccount(ctx context.Context, userID string) (port.Account, error) {
    user, err := l.users.FindByID(ctx, userID)
    if err != nil {
        return port.Account{}, err
    }
    return port.Account{UserID: user.ID, Email: user.Email}, nil
}
```

## 事件

- `subscription.activated`
- `subscription.renewed`
- `subscription.updated`
- `subscription.canceling`
- `subscription.canceled`
- `subscription.reactivated`
- `payment.failed`
- `credits.purchased`

billing 只发布商业事实。某个套餐对应多少额度、要停哪些资源、是否激活邀请，全部由
当前 SaaS 的监听器决定。监听器返回错误时 Webhook 不会标记完成，Stripe 重试会重新
投递；积分和额度入账仍必须使用 provider/event ID 做业务幂等。

## 旧表迁移

`cmd/legacy-billing-migrate` 只用于把旧 `users` 表里的 Stripe 字段迁入 billing 表。
新项目不需要运行。
