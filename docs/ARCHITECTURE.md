# 整体架构

## 目标

这个仓库服务多个独立 SaaS，优化目标依次是：

1. 新项目启动快；
2. 登录、支付、邮件和邀请实现可以复用；
3. 每个 SaaS 可以独立修改用户字段和业务规则；
4. 替换第三方服务商时不需要改共享领域逻辑；
5. 不为了假设中的规模提前引入微服务和消息队列。

当前采用模块化单体。每个 SaaS 单独部署、单独使用数据库，但依赖同一套模块库。

## 三层边界

```text
┌─────────────────────────────────────────────────────┐
│ 具体 SaaS / templates/quickstart                    │
│ User、迁移、配置、服务商选择、事件订阅、业务功能    │
└───────────────────────┬─────────────────────────────┘
                        │ 注入端口、订阅事件
┌───────────────────────▼─────────────────────────────┐
│ modules/auth | billing | email | referral           │
│ 共享业务能力；不知道当前 SaaS 的完整 User 和规则    │
└───────────────────────┬─────────────────────────────┘
                        │ 使用基础设施
┌───────────────────────▼─────────────────────────────┐
│ foundation/config | pgx | slog | tracing | ginx     │
│ 通用技术能力                                         │
└─────────────────────────────────────────────────────┘
```

### foundation

只放与具体业务无关、接口稳定的技术能力。例如数据库连接、日志、追踪、HTTP 响应、
随机数和弹性控制。它不应该导入 `modules` 或模板。

### modules

每个模块内部按下面的方向依赖：

```text
http → app → domain / port
adapter → port / domain
模块入口 New() 负责把传入的端口组装起来
```

模块可以发布事件，但不能在内部直接调用另一个业务模块。例如 auth 不导入 email、
billing 或 referral。邮箱验证码所需的发送能力通过 `emailcode.Mailer` 注入。

### template / 具体 SaaS

这是组合根，也是允许出现跨模块依赖的地方：

- `internal/user`：当前 SaaS 的用户模型和适配器；
- `internal/platform`：选择服务商、创建模块、迁移模块表、挂载路由；
- `internal/bootstrap/subscriptions.go`：连接模块事件；
- `internal/bootstrap/host_hooks.go`：当前 SaaS 的业务回调；
- `internal/feature/*`：产品本身的功能。

## 用户所有权

共享 auth 只定义：

```go
type UserStore interface {
    FindByEmail(...)
    FindOrCreateByEmail(...)
    FindOrCreateFromOAuth(...)
    FindByID(...)
    MarkLogin(...)
}
```

quickstart 在 `internal/user/auth_store.go` 实现它。`User` 增加字段不会改变接口。

默认数据库边界：

| 表 | 所有者 |
| --- | --- |
| `users`、`user_identities` | 当前 SaaS |
| `auth_email_*`、`auth_exchange_codes` | auth 模块 |
| `billing_*` | billing 模块 |
| `referral_codes`、`referrals` | referral 模块 |
| `notes` 和其他产品表 | 当前 SaaS 的 feature |

`user_identities` 独立于 `users`，避免把 `provider`、`subject` 写成只能保存一组的
用户字段。同一个用户可以连接多个登录方式。

### 用户字段变更规则

不要用“以后可能变化”作为共享完整 `User` 的理由。字段按所有权和变化速度处理：

| 字段类型 | 示例 | 放置位置 |
| --- | --- | --- |
| 登录与账号核心 | ID、Email、Role、LastLoginAt | 当前 SaaS 的 `internal/user.User` |
| 当前产品稳定的一对一资料 | Locale、Timezone、OnboardingStep | 当前 SaaS 的 `User`，并做版本化 migration |
| 易变或可重复的业务状态 | 工作区成员、额度流水、偏好、用量 | `internal/feature/*` 的独立表，以 `user_id` 关联 |
| 登录供应商身份 | Google/GitHub subject | 宿主的 `user_identities` |
| 支付、邀请等模块数据 | customer ID、订阅、邀请关系 | 对应模块自己的表 |

判断标准不是“字段是否和用户有关”，而是“谁负责这个字段的规则和生命周期”。大多数
业务数据都和用户有关，但不应该因此全部成为 `users` 表的列。

某个 SaaS 新增稳定字段时，只修改它复制出来的模板和数据库 migration；共享模块升级
不会改这张表。探索期、结构不稳定但又不参与查询和约束的少量数据可以临时使用 JSON，
语义稳定后再迁成明确字段或独立表；不要长期用 JSON 代替关系模型。

## 事件组合

```text
auth.UserSignedUp
  ├─ 读取请求里的 referral_code，调用 referral.AttributeReferral
  ├─ 按当前 SaaS 配置发欢迎邮件
  └─ 按当前 SaaS 规则发注册积分

billing.SubscriptionActivated
  ├─ 尝试激活当前用户的邀请关系
  └─ 调用当前 SaaS 的订阅开通回调

referral.ReferralActivated
  ├─ quickstart 示例把奖励加到宿主 User.Credits
  └─ 调用当前 SaaS 的邀请奖励回调
```

这些关系全部位于模板。某个 SaaS 不需要欢迎邮件，就关闭配置或删除订阅；某个
SaaS 的奖励是优惠券而不是积分，就替换 `onReferralActivated`。

## 服务商替换

- OAuth：实现 `auth/port.IdentityProvider`，在 `buildIdentityProviders` 注册；
- 邮件：实现 `email/port.Sender`，在 `buildEmail` 选择；
- 支付：实现 `billing/port.Provider`，在 `buildBilling` 选择；
- 数据库：实现对应 repository 端口，模块的 app/domain 不变。

替换发生在当前 SaaS 的 `internal/platform`。如果新适配器对多个 SaaS 都有价值，
再把适配器贡献回 `modules/*/adapter`。

## 可靠性和失败处理

当前事件总线是进程内同步执行：

- 监听器按注册顺序执行；
- 错误和 panic 会记录日志，不阻止其他监听器；
- 它适合快速启动和单进程模块化单体；
- 它不保证进程崩溃后的重放。

支付 Webhook 本身由 billing 事件表做幂等。欢迎邮件、外部资源创建等需要强可靠
重试时，应在宿主监听器写入 outbox，再由 Runner 异步消费，而不是让模块互相导入。

## 安全边界

- JWT 和 OAuth state 密钥必须分开生成；
- 关闭 auth 时，用户/管理员路由返回 503，不会退化为匿名访问；
- 关闭模块时不迁移它的表，也不挂它的路由；
- Stripe Webhook 先验签、再幂等处理；
- 完整用户资料不会传给支付或邀请模块。

## 不采用的方案

- 强制共享 `saascore`：组合方式和用户结构会再次被锁死；
- 共享完整 `User`：一个 SaaS 的字段变化会成为所有 SaaS 的升级风险；
- 每个项目复制所有模块：安全修复和支付修复会重复十次；
- 立即拆微服务：当前收益不足以抵消部署和一致性成本。
