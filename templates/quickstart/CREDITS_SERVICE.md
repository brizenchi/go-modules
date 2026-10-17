# 公共积分服务

所有产品功能、邀请回调和支付回调统一使用 `deps.Credits`。
实现：`internal/credit/service.go`。依赖在启动时通过 `hostapi.Deps.WithServices()` 装配；
已经注入的服务不会被覆盖。没有修改表结构或 HTTP 返回字段。

## 分层

```text
HTTP Handler / 邀请回调 / 支付回调 / 产品 Service
                    ↓
             credit.Service（deps.Credits）
                    ↓
        Store 接口 → user.Repository（事务、锁、账本）
                    ↓
       User / CreditTransaction / CreditLot 等 Model
```

Service 是统一业务入口，限制用户查询范围、区分管理员操作、组织退款。
余额校验、重复操作去重、到期处理和账本更新必须在数据库事务内执行，保留在 Repository，
不能拆成 Service 先查余额、再单独 UPDATE。Model 是数据结构，不负责接收 HTTP 请求。
积分 HTTP handler 只处理身份、请求和响应；笔记导出由 ExportNote 组织业务，
SQL 位于 `internal/feature/credits/export_repository.go`。
原有 Repository 方法和 `credits.New(users)` 保留兼容，新增业务使用 Service。

## 可调用方法

| 方法 | 用途 |
| --- | --- |
| GetCreditSummary | 当前可用余额、近期到期量、最近到期日期；会结算到期积分 |
| ListCreditTransactions | 指定用户分页流水；空用户 ID 被拒绝 |
| GrantCredits | 简单发放；重复的来源和业务编号不重复加分 |
| GrantCreditsWithExpiry | 发放并指定说明、有效期、操作人 |
| FindGrant | 只读查询指定用户的历史发放，可兼容旧事件键；不结算到期、不改余额 |
| ConsumeCredits | 扣分，余额不足拒绝，返回扣分流水 |
| ConsumeCreditsAndDo | 扣分和本地业务写入同一数据库事务 |
| RefundCredits | 按原消费流水全额退回一次，检查归属 |
| RefundByAdmin | 管理员按流水退回，服务从原流水确定用户 |
| ListAdminTransactions | 管理员分页查询，可不指定用户 |

管理员方法只能从已验证管理员权限的入口调用；Service 本身不接受或验证登录令牌。
用户 ID 来自认证身份或可信服务器事件，不使用前端传来的任意用户 ID。

## 查询余额

```go
summary, err := deps.Credits.GetCreditSummary(ctx, userID)
if err != nil {
    return err
}
// summary.Balance 是扣除已到期积分后的可用余额。
```

## 消费 5 积分

```go
entry, err := deps.Credits.ConsumeCredits(ctx, credit.Consumption{
    UserID: userID,
    Source: "report_export",
    SourceID: userID + ":" + operationID,
    Amount: 5,
    Reason: "导出报告",
})
if err != nil {
    return err
}
// entry.ID 为流水编号，entry.BalanceAfter 为该笔消费后的余额。
```

引入 `github.com/brizenchi/quickstart-template/internal/credit`（复制项目后使用项目模块路径）。
`operationID` 是同一个业务操作的稳定编号，重试必须复用，不要每次生成新编号。
使用 `errors.Is(err, credit.ErrInsufficientCredits)` 判断余额不足，其他错误同样使用 `errors.Is`。
来源键在整个账本唯一，因此自定义操作编号通常包含 userID。
扣分数量必须由服务器确定，不能信任浏览器提交的金额。
重复请求返回原流水，不代表可以再次执行昂贵的外部业务。

## 发放积分并记录说明

```go
_, err := deps.Credits.GrantCreditsWithExpiry(ctx, credit.Grant{
    UserID: userID,
    Source: "activity_reward",
    SourceID: userID + ":" + activityID,
    Amount: 20,
    Reason: "活动奖励",
    // ExpiresAt: &expiresAt, // 不设置即无到期日
})
return err
```

后台用户弹窗的「积分变更记录」统一读取这一账本，显示来源、正负数量、变更后余额、原因、操作人和关联记录。邀请奖励、充值和管理员调整不再分散到多个用户标签。

新业务可使用 `subscription_allowance` / `monthly` 表示周期发放，`campaign` / `promotion` 表示活动赠送，并提供稳定的 `SourceID` 防止重复发放；已有来源值和幂等键保持兼容，未知来源直接显示原值。这里提供记录和展示能力，不自动创建月度任务或活动发放规则。

## 业务失败时不扣分

本地数据库操作使用 `ConsumeCreditsAndDo`，回调内仅执行使用传入 tx 的数据库读写。
回调失败，扣分、流水和业务结果一起回滚；重复成功请求不会再次调用回调。
完整例子见 `internal/feature/credits/export.go` 和 `export_repository.go`。

不要在此事务回调里调用 AI、支付或其他外部 API：事务不能撤回外部操作。
外部任务需要独立的持久化任务、重试/补偿设计；当前服务不提供预冻结、结算或部分退款。

## 兼容与验证

邀请注册和付费奖励、购买积分、管理员赠送/退回、用户余额/流水和笔记导出均使用公共服务。
没有修改邀请奖励数额、发放时机或支付规则。
积分账本方法继续在 user.Repository 内兼容旧项目和账户余额投影。
`RefundCredits` 是消费积分退回，不等于 Stripe 现金退款。

购买积分统一经过 `internal/creditpurchase`：已验证的 Webhook 与付款返回确认共用同一个发放方法，以 Checkout Session 为幂等键。`Grant.IdempotencyAliases` 在账户锁内兼容旧版事件 ID 发放，不重写原流水，也不重复入账。来源、金额或归属冲突会拒绝，不自动修改历史余额。

取消续费只停止下一期扣款，当前付费期权益保留至到期，已有积分不退款、不清空。积分自身原有到期规则仍然适用；本次不新增现金退款或退款后自动回收积分的策略。

测试覆盖：余额不足、用户隔离、稳定键重试、管理员退回、取消上下文、扣分与业务写入共同回滚。
原有 Repository、积分 HTTP、邀请/支付组合测试继续验证账本和接口行为。

```bash
go vet ./...
go test -race ./...
```
