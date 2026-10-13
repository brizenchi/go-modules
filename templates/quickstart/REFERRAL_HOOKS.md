# 邀请积分奖励

规则写在 `internal/bootstrap/host_hooks.go`，不需要奖励金额配置。

| 阶段 | 函数 | 邀请人 | 被邀请人 |
| --- | --- | --- | --- |
| 有效邀请注册成功，不需要购买 | onReferralRegistered | +20 | +10 |
| 首次符合条件的付费激活（可选） | onReferralActivated | 再 +20 | 再 +10 |

两阶段都完成后双方累计获得 40 / 20。普通注册赠送积分另算。
首次付费通过购买用户 ID 查找已绑定的邀请关系，再奖励双方；无邀请关系不发这笔积分。
购买积分包、免费试用不触发付费邀请奖励，后续续费不重复发放。

## 修改或关闭

只要注册奖励：把 onReferralActivated 中调用 grantReferralCredits 的 return 注释，改为 return nil。
同步将 subscriptions.go 三处 activateReferral 调用传入的奖励统计值改为 0，更新前端活动文案。
注册奖励独立保留。注册阶段仍为 pending，activated 表示付费转化，不代表注册奖励未到账。

## 余额与去重

双方在 `/credits` 查看注册和付费奖励的独立流水。
`/referrals` 的“付费激活奖励”仅统计付费阶段邀请人奖励，不含注册奖励；完整收支看 `/credits`。
注册来源为 referral_signup_inviter / referral_signup_invitee；付费来源保持 referral_inviter / referral_invitee。
每笔 source_id 为邀请 ID。重复注册事件、重复支付事件、管理员重放不会重复发同阶段奖励。
双方分别事务提交；第二笔失败时返回错误，重试只补齐缺失奖励。

不会批量补发历史记录。重放旧邀请事件会按当前规则处理缺少的奖励；已有项目需核对历史政策。
本地 cmd/preview 是隔离演示，不装配这些业务奖励回调；验收使用正常 quickstart 测试服务。

## 手工验收

1. A 登录 `/credits`，记录余额，然后在 `/referrals` 复制邀请链接。
2. 无痕窗口打开链接，B 使用新邮箱注册，不购买任何商品。
3. 双方打开 `/credits` 点“刷新”：A 增加 20；B 有“受邀注册奖励 +10”，注册赠送另算。
4. 重复登录和刷新，不能再次增加注册奖励；邀请状态仍可显示待订阅激活。
5. B 购买 Pro 月付并付款成功，刷新 `/billing`，应显示专业版和有效会员。
6. 双方刷新 `/credits`：A 再增加 20，B 再增加 10；流水标注付费奖励。
7. 后续续费、重复支付回调不能再发这笔邀请奖励。
8. 购买积分包只按实际包数发购买积分，不触发上述付费邀请奖励。

后端验证：`go vet ./...` 和 `go test -race ./...`。
