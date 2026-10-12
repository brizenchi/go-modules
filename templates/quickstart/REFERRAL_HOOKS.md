# 自定义邀请奖励

**模板默认不发任何邀请奖励。你只需要在业务回调里写规则，不需要奖励金额配置。**
本指南随后端一起复制到新项目；以下路径均相对于后端目录（生成项目的 `backend/`）。

## 1. 找到要改的函数

打开 [internal/bootstrap/host_hooks.go](internal/bootstrap/host_hooks.go)：

| 业务需求 | 修改函数 | 触发时机 |
| --- | --- | --- |
| 通过邀请注册就送会员、积分等 | `onReferralRegistered` | 新用户携有效邀请码注册，邀请关系成功绑定后 |
| 被邀请人付费后再发奖 | `onReferralActivated` | 在激活期限内完成符合条件的付费订阅后 |

两个函数已经由 `subscriptions.go` 注册，默认只有 `return nil`。直接修改函数体，
无需另建 HTTP 接口、再次订阅事件或修改共享 `modules/referral`。

默认流程：

```text
新用户携有效邀请码注册
  → 保存 pending 邀请关系
  → onReferralRegistered（默认不发奖）
  → 在期限内完成符合条件的付费订阅
  → 保存 activated 状态
  → onReferralActivated（默认不发奖）
```

仅打开邀请链接、无效邀请码、老用户登录都不构成新的邀请绑定。
免费试用本身不会激活邀请，单独购买积分包也不会激活邀请。
如果要把激活条件改成“完成注册”或“完成某个业务操作”，修改
[subscriptions.go](internal/bootstrap/subscriptions.go) 的 `activateReferral` 调用位置。
单纯在注册回调发奖不会自动改变 pending 状态。

## 2. 回调里能拿到什么

| 参数 | 用途 |
| --- | --- |
| `ctx` | 传给下游数据库和服务调用，保留取消、超时和追踪信息 |
| `deps.Users` | 宿主用户仓库，包括 `GrantCredits` 等积分方法 |
| `deps.DB` / `deps.Modules` | 宿主数据库和已启用的模块；更复杂业务建议封装成 service |
| `event.Referral.ID` | 邀请记录 ID，可作为稳定的发奖幂等依据 |
| `event.Referral.ReferrerID` | 邀请人，即分享链接的人 |
| `event.Referral.RefereeID` | 被邀请人，即使用邀请码注册的人 |
| `event.Referral.CreatedAt` / `ActivatedAt` | 绑定时间 / 激活时间 |
| `envelope` | 事件类型与事件时间；重放时不要把它当成新的发奖资格 |

## 3. 可直接复制：付费激活后给邀请人 100 积分

在 `host_hooks.go` 的 import 中添加 `"strconv"`，用以下实现替换原来的
`onReferralActivated`。其他类型的 import 已在文件中，无需新增依赖：

```go
func onReferralActivated(
    ctx context.Context,
    deps hostapi.Deps,
    _ referralevent.Envelope,
    event referralevent.ReferralActivated,
) error {
    const credits int64 = 100
    sourceID := strconv.FormatUint(event.Referral.ID, 10)
    if err := deps.Users.GrantCredits(
        ctx, event.Referral.ReferrerID, "referral", sourceID, credits,
    ); err != nil {
        return fmt.Errorf("grant inviter credits: %w", err)
    }
    return nil
}
```

`GrantCredits` 已按 `source + sourceID` 去重；相同奖励重试不会重复加积分。
如果改成注册就送积分，把上述函数体放入 `onReferralRegistered` 即可。
如果给被邀请人发积分，使用 `RefereeID`，并使用另一个 source，例如 `referral_referee`。
同一邀请向两个人发奖不能共用同一个 `source + sourceID`。

邀请页的 `RewardCredits` 是独立的统计字段，默认激活时为 0。
上面的示例会更新实际积分余额和流水，但不会自动更新这个邀请统计字段。
如果产品还需要邀请统计显示 100 积分，在 `subscriptions.go` 的各个 `activateReferral`
调用处传入一致的奖励数量，并在回调中使用已保存的 `event.Referral.RewardCredits` 发奖。
这样以后调整规则时，旧邀请重放仍使用原来的积分数量。

## 4. 你的例子：被邀请人拿会员，邀请人拿积分

如果两个人都是注册成功就拿奖励，统一写在 `onReferralRegistered`；
如果被邀请人注册即拿会员、邀请人等对方付费后才拿积分，分别写在两个回调。

积分方法已经具备。**模板目前没有赠送会员的方法，不能仅写一个调用就获得会员能力。**
先在当前项目实现会员权益，再从回调调用。建议按以下分工：

| 文件 | 需要实现的内容 |
| --- | --- |
| `internal/user/membership_model.go`（新增） | 赠送权益：用户、套餐、起止时间、奖励来源、幂等键 |
| `internal/user/membership_repository.go`（新增） | 例如 `GrantReferralMembership(ctx, userID, referralID, plan, days)`：事务内去重并写入权益 |
| `internal/platform/migrate.go` 或 `internal/bootstrap/host_migrate.go` | 把新增 GORM 模型加入启动迁移 |
| 当前项目的会员权限判断 | 同时识别有效付费订阅和有效赠送权益，并向前端提供真实会员状态 |
| `internal/bootstrap/host_hooks.go` | 调用会员方法和现有积分方法，决定受益人、套餐、天数及触发时机 |

例如“注册后被邀请人获 7 天 Pro、邀请人获 100 积分”的业务顺序如下。
这是实现步骤，`GrantReferralMembership` 是你需要新增的方法，不是现成 API：

```text
onReferralRegistered(ctx, deps, envelope, event):
  调用会员仓库，为 event.Referral.RefereeID 赠送 7 天 Pro
    幂等依据：邀请 ID + referee_membership
    失败：返回错误，停止本次处理
  调用 deps.Users.GrantCredits，为 event.Referral.ReferrerID 发 100 积分
    source = referral，sourceID = 邀请 ID
    失败：返回错误
  返回 nil
```

会员和积分各自都要持久化去重：会员成功、积分失败后重试，会员不能再延长 7 天。
会员去重记录与权益写入应在同一事务里，并用唯一约束处理并发。
首次发放时保存套餐和有效期，重试读取原记录，不能每次用当前时间重新计算到期日。
已有会员是否延长、不同套餐是否升级，由当前项目明确决定；赠送权益应避免被后续
支付订阅同步覆盖。

## 5. 实现后怎么验证

在 [internal/bootstrap/subscriptions_test.go](internal/bootstrap/subscriptions_test.go)
增加当前项目的奖励用例；现有“默认不发奖”断言要随你的规则调整。

- 两个独立账号完成邀请注册：验证双方奖励对象、数量及会员到期时间。
- 同一邀请重复投递、并发投递：每种奖励只成功发放一次。
- 第一种奖励成功、第二种失败，再重试：只补齐缺失奖励。
- 无效邀请码、自邀、已有账号重新绑定：不能获得新的邀请奖励。
- 付费触发的规则：免费试用、零金额续费和超过激活期限时不误发奖。

```bash
go vet ./...
go test -race ./...
```

默认回调是空函数时事件也可能已经处理完毕。以后启用奖励不会自动补发历史邀请；
上线前要决定历史邀请是否适用新规则。管理员的奖励核对只会重放已激活邀请回调，
不会重放注册回调或强行激活邀请。

最后同步修改前端 `/invite`、`/referrals` 的活动文案；如果增加会员 API 字段，
同时更新前端 `lib/api.ts` 和会员状态展示。
