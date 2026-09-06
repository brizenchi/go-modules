# 模块接入指南

quickstart 已经给出了默认组合。只有已有项目不适合复制模板时，才需要直接接入模块。

## 接入 auth

实现当前项目自己的 `auth/port.UserStore`：

```go
type AuthStore struct {
    users *MyUserRepository
}

func (s *AuthStore) FindOrCreateFromOAuth(
    ctx context.Context,
    profile authdomain.OAuthProfile,
) (*authdomain.Identity, error) {
    user, created, err := s.users.FindOrCreateFromOAuth(ctx, profile)
    if err != nil {
        return nil, err
    }
    return &authdomain.Identity{
        UserID:   user.ID,
        Email:    user.Email,
        Username: user.Name,
        Provider: profile.Provider,
        Subject:  profile.Subject,
        IsNew:    created,
    }, nil
}
```

然后注入 JWT、临时 store、Provider 和事件总线：

```go
authModule := auth.New(auth.Deps{
    UserStore:         myAuthStore,
    RoleResolver:      myRoleResolver,
    TokenSigner:       signer,
    WSTicketSigner:    ticketSigner,
    ExchangeCodeStore: authStore,
    EmailCodeIssuer:   issuer,
    EmailCodeVerifier: verifier,
    IdentityProviders: providers,
    Bus:               eventbus.NewInProc(),
})
```

生产使用 GORM 时，认证临时表由 `auth/adapter/gormstore.AutoMigrate` 创建；它不会
创建用户表。

## 接入 billing

billing 不读取固定 users 表。实现最小账户投影：

```go
type AccountLookup struct {
    users *MyUserRepository
}

func (l *AccountLookup) FindBillingAccount(ctx context.Context, userID string) (billingport.Account, error) {
    user, err := l.users.FindByID(ctx, userID)
    if err != nil {
        return billingport.Account{}, err
    }
    return billingport.Account{UserID: user.ID, Email: user.Email}, nil
}
```

然后使用 billing 自己的 GORM 表：

```go
customers := billingrepo.NewCustomerStore(db, accountLookup)
resolver := billingrepo.NewUserResolver(db, accountLookup)

billingModule := billing.New(billing.Deps{
    Provider:     stripeProvider,
    Bus:          billingeventbus.NewInProc(),
    Customers:    customers,
    EventRepo:    billingrepo.NewBillingEventRepo(db),
    Subscriptions: billingrepo.NewSubscriptionRepo(db),
    UserResolver: resolver,
    GetUserID:    currentUserID,
})
```

替换 Stripe 时实现 `billing/port.Provider`，其他代码不需要知道服务商。

## 接入 email

```go
sender, err := resend.New(resend.Config{
    APIKey: "...",
    Sender: emaildomain.Address{Email: "no-reply@example.com"},
})
if err != nil {
    return err
}
emailModule := email.New(sender, nil)
```

email 模块只负责发送。何时发送由宿主订阅 auth/billing/referral 事件决定。

## 接入 referral

```go
referralModule := referral.New(referral.Deps{
    Codes:      gormrepo.NewCodeRepo(db),
    Referrals:  gormrepo.NewReferralRepo(db),
    Generator:  codegen.NewRandom("INV", 10),
    Bus:        referraleventbus.NewInProc(),
    GetUserID:  currentUserID,
    BaseLink:   "https://example.com/invite?ref=",
})
```

两个跨模块动作由宿主显式调用：

1. 注册事件中调用 `AttributeReferral`；
2. 首次合格付费事件中调用 `ActivateReferral`。

奖励入账订阅 `ReferralActivated` 后实现。它可以是积分、优惠券、现金或完全不同的规则。

## 事件监听器

监听器应该满足：

- 可重复执行或带幂等键；
- 不假设事件一定只投递一次；
- Stripe 监听器失败时返回 error：Webhook 不会标记 processed，Stripe 重试时会重新投递；
- 积分、额度等入账另外保存 provider/event 幂等键，不能只依赖 Webhook event 状态；
- 登录后的可选副作用要自行决定是阻断、异步重试还是仅记录，避免邮件故障拖垮登录；
- 需要强可靠时写 outbox，由后台 Runner 消费。

完整可运行实现见：

- `templates/quickstart/internal/platform`
- `templates/quickstart/internal/bootstrap/subscriptions.go`
- `templates/quickstart/internal/bootstrap/host_hooks.go`
