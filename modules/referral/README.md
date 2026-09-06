# referral 邀请模块

提供邀请码、邀请归因、激活状态、统计和邀请领域事件。

## 边界

referral 只保存用户 ID 和邀请关系，不读取完整 User，也不决定奖励如何发放。

## 组装

```go
if err := db.AutoMigrate(gormrepo.AutoMigrateModels()...); err != nil {
    return err
}

module := referral.New(referral.Deps{
    Codes:      gormrepo.NewCodeRepo(db),
    Referrals:  gormrepo.NewReferralRepo(db),
    Generator:  codegen.NewRandom("INV", 10),
    Bus:        eventbus.NewInProc(),
    GetUserID:  currentUserID,
    BaseLink:   "https://example.com/invite?ref=",
})
```

## 两个宿主调用点

注册成功并携带邀请码时：

```go
_, err := module.Attribute.AttributeReferral(ctx, newUserID, referralCode)
```

被邀请人满足激活条件时：

```go
_, err := module.Attribute.ActivateReferral(ctx, refereeID, rewardAmount)
```

激活条件可以是首次订阅、完成订单或其他产品事件，不固定在 referral 内。
`NewRandom` 配合数据库唯一索引适合公开生产邀请码；截断用户 ID 的
`NewDeterministic` 只适用于调用方能保证截断后仍唯一的内部场景。

## 事件

```go
module.Subscribe(referralevent.KindReferralRegistered, onRegistered)
module.Subscribe(referralevent.KindReferralActivated, onActivated)
```

`ReferralActivated` 的 `RewardCredits` 只是事件中的记账数值。实际奖励可以是：

- 宿主 User 积分；
- 独立 wallet/credits service；
- 优惠券；
- 现金返利；
- 不发奖励，只做邀请统计。

默认 quickstart 在模板监听器中给邀请人增加 `User.Credits`，每个 SaaS 可以替换。
