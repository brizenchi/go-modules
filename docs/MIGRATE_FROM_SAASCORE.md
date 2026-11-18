# 从 saascore 升级到新版模板

适用于从 go-modules `d6a572f`（2026-08-07）或更早版本复制出来、仍依赖
`stacks/saascore` 和 `modules/user` 的产品。按本文操作后，产品改用新版模板的结构，
用户、付费、积分、邀请数据全部保留。

本文的每一步都用模拟旧库完整演练过：[`scripts/saascore-rehearsal/run.sh`](../scripts/saascore-rehearsal/run.sh)
（见[第 4 节](#4-演练)）。动生产库之前，建议先用生产库的脱敏副本按本文走一遍。

| 你要做的 | 什么时候做 | 停机 |
| --- | --- | --- |
| [1. 代码迁移](#1-代码迁移) | 提前在分支上完成 | 不需要 |
| [2. 前端修改](#2-前端修改) | 和后端同时上线 | 不需要 |
| [3. 数据迁移](#3-数据迁移) | 维护窗口 | 需要，通常几分钟 |

## 0. 先看结论：哪些东西变了

2026-09-02 起（ADR-0001，`docs/adr/0001-template-owned-composition-and-user-schema.md`），
共享库不再提供 `stacks/saascore` 和共享的 `modules/user`。用户表、模块组合和业务回调
都放进模板，复制后归产品所有。

### 数据去哪了

旧 `users` 表的列不会被删除（AutoMigrate 只加不删），但新代码不再读写下面这些列：

| 旧 `users` 列 | 新位置 | 怎么迁 |
| --- | --- | --- |
| `id`、`email`、`role`、`username`、`avatar_url`、`email_verified*`、`last_login_at` | 原地保留 | 不用迁 |
| `provider`、`provider_subject` | `user_identities` | [`user_identities.sql`](migrate-from-saascore/user_identities.sql)（第 5 步） |
| `stripe_customer_id` | `billing_customers` | `legacy-billing-migrate`（第 4 步） |
| `stripe_subscription_id`、`stripe_price_id`、`stripe_product_id`、`billing_status`、`billing_period_start/end`、`cancel_effective_at` | `billing_subscriptions` | `legacy-billing-migrate`（第 4 步） |
| `plan` | 由 `billing_subscriptions` 推出；手动给的会员放 `membership_overrides` | 第 4 步 + 第 7 步 |
| `credits` | 积分账本 `user_credit_transactions` / `user_credit_lots`；`users.credits` 继续作为余额缓存 | [`credit_ledger.sql`](migrate-from-saascore/credit_ledger.sql)（第 6 步） |

`auth_*`、`referral_codes`、`referrals`、`billing_events` 表结构兼容，数据不用动。
新代码启动时只会新增表和列，不改已有数据，也不会在已有表上加唯一约束（已用模拟旧库验证）。

### 上线后用户会感受到的变化

1. **所有人需要重新登录一次。** 新版 token 多了必填的 `token_type`，saascore 签发的旧
   token 一律返回 401。JWT 密钥不用换，前端收到 401 跳登录页即可。
2. **付款失败（`payment_failed`、`past_due` 等）立即失去会员权限。** saascore 在付款失败期间
   保留 `users.plan`；新模板的 `membership.Evaluate` 默认没有宽限期。要保留宽限期，
   改 `internal/feature/membership/membership.go` 的 `Evaluate`。
3. **手动给的会员要重新登记。** 只改过 `users.plan`、没有 Stripe 记录的用户，迁移后变成 free，
   需要在第 7 步写入 `membership_overrides`。
4. **账单周期已过但没收到续费事件的订阅**会被视为过期。第 8 步的核对会列出这些用户。

## 1. 代码迁移

### 1.1 新建项目骨架

用新版模板生成一个新目录，再把业务代码搬进去，比在旧项目里逐个文件改更快：

```bash
cd go-modules            # 已检出 v0.3.0
make init-quickstart DEST=../myapp-next MODULE=github.com/me/myapp APP=myapp VERSION=v0.3.0
```

后端在 `../myapp-next/backend`，配套前端在 `../myapp-next/frontend`。`go.mod` 里的
`github.com/brizenchi/go-modules` 已固定到 `v0.3.0`；旧项目里的临时提交号和 `replace`
不要带过来。

### 1.2 目录对照

| 旧（`d6a572f` 模板） | 新 |
| --- | --- |
| `internal/bootstrap/saascore.go`（拼 `saascore.Config`） | `internal/platform/config.go`、`internal/platform/*_provider.go` |
| `internal/bootstrap/host_hooks.go`（`saascore.HostHooks`） | `internal/bootstrap/host_hooks.go`（同名函数，签名改为 `(ctx, deps hostapi.Deps, envelope, event)`） |
| `internal/bootstrap/referral.go`（`PolicyHooks.ResolveReferralReward`） | `host_hooks.go` 的 `onReferralRegistered` / `onReferralActivated`；激活时机在 `subscriptions.go` 的 `activateReferral` |
| `internal/http/host_routes.go` | `internal/http/host_routes.go`（YOURS），按 `Public` / `User` / `Admin` 分组 |
| 自己的业务包 | `internal/feature/<名字>/`，参考 `internal/feature/note` |
| 直接用 `modules/user/adapter/gormrepo` 读写用户 | `internal/user`（模板复制，产品拥有） |
| 直接改 `users.credits` | `deps.Credits`（`internal/credit`），见 `templates/quickstart/CREDITS_SERVICE.md` |
| 读 `users.plan` 判断会员 | `membership.Load` / `membership.Require` |

### 1.3 哪些文件可以改

- **保持和模板一致**（标注 `TEMPLATE-OWNED`，方便以后合并模板更新）：`cmd/quickstart/main.go`、
  `internal/bootstrap/app.go`、`runner.go`、`internal/hostapi/hostapi.go`、`internal/http/router.go`、
  `internal/http/middleware/router.go`。`internal/platform`、`internal/user`、`internal/credit` 也尽量不改。
- **产品自己写**（标注 `YOURS` 或本来就是扩展位置）：`internal/bootstrap/host_hooks.go`、
  `host_jobs.go`、`host_migrate.go`、`internal/http/host_routes.go`、`internal/hostcfg`、
  `internal/feature/*`，以及 `internal/feature/membership` 的 `Evaluate`。
- 要给用户加数据：可以在 `internal/user/model.go` 加字段；想和模板保持一致的，用独立表
  通过 `user_id` 关联。

业务代码通过 `hostapi.Deps` 拿依赖（数据库、积分、模块、配置、HTTP 客户端），不要用包级
全局变量。新表加到 `host_migrate.go` 的 `hostModels`，后台任务加到 `host_jobs.go`。

> **上线前必须改掉**：任何直接 `UPDATE users SET credits = …` 的代码。迁移后余额以积分账本为准，
> 直接改列会让账本和余额对不上，后续积分操作会报 `credit ledger balance mismatch`。

### 1.4 配置

旧配置项都还在，含义不变；`referral.activation_reward` 被删掉，奖励金额改在 `host_hooks.go` 里写。
特别注意：

- `auth.user_jwt_secret` 保持原值。`server.name` 会写进 token 的 `iss`，改不改都可以（反正要重新登录）。
- `auth.admin_emails` 和 `users.role = 'admin'` 仍然决定谁是管理员；另有可选的
  `auth.admin_email` / `auth.admin_password` 用于 `/admin` 密码登录。
- 新增项见 `templates/quickstart/deploy/config.yaml.example`。

`APP_ENV=production` 时，新模板启动前会做更严格的检查，旧配置常见的不通过项：

| 检查 | 要求 |
| --- | --- |
| `db.ssl_mode` | `require`、`verify-ca` 或 `verify-full` |
| `auth.user_jwt_secret` | 至少 32 个随机字符，不能是模板占位值 |
| `http.allowed_origins` | 前端 HTTPS origin，不能是 `*`；留空时从 `auth.frontend_redirect` 推出 |
| `auth.oauth_cookie_secure` | 必须为 true（回调地址是 HTTPS 时省略即可） |
| `email.provider` | 开启邮箱登录时必须是 `resend` 或 `brevo` |
| Stripe | `sk_live_` / `sk_test_` / `rk_*` 密钥、`whsec_` 签名密钥、至少一个价格 ID |

### 1.5 本地验证

```bash
go build ./... && go test ./...
```

再用一份生产库的脱敏副本启动新代码，确认能起来。新代码启动时会自动建新表，所以不要
直接连生产库。

## 2. 前端修改

接口路径都没变，旧接口全部保留。需要改的是：

1. **收到 401 时清掉本地 token 并跳登录页。** 上线后所有旧 token 都会 401。
2. **Google/GitHub 登录多了一次性校验。** 发起登录时先生成 verifier，跳转
   `GET /api/v1/auth/:provider/authorize?redirect=1&challenge=<base64url(SHA-256(verifier))>`；回调后调用
   `POST /api/v1/auth/exchange-token` 时带上 `oauth_verifier`。旧前端不带这两个参数会收到 400。
   可以直接参照 `templates/quickstart-nextjs` 的实现。
3. **退出登录会吊销 token。** `POST /api/v1/auth/logout` 之后，这个 token 不能再用。
4. **用新接口读账户信息**（类型见 `templates/quickstart-nextjs/lib/api.ts`）：

| 用途 | 接口 |
| --- | --- |
| 开启了哪些登录方式、是否开启付费 | `GET /api/v1/capabilities` |
| 当前用户资料 | `GET` / `PATCH /api/v1/account/profile` |
| 会员状态（套餐、是否有效、到期时间） | `GET /api/v1/account/membership` |
| 积分余额和流水 | `GET /api/v1/credits`、`GET /api/v1/credits/transactions` |
| 购买记录、积分包支付结果 | `GET /api/v1/billing/purchases`、`GET /api/v1/billing/credit-checkouts/:id` |
| 取消已预约的套餐变更 | `POST /api/v1/stripe/subscription/scheduled-change/cancel` |
| 管理后台 | `POST /api/v1/auth/admin/login`、`/api/v1/admin/*` |

完整的新旧接口对照见[第 5 节](#5-接口对照)。

## 3. 数据迁移

### 准备

- 一台能连生产库的机器，装好 `psql` 和 Go。
- 检出 go-modules 的发布版本，下面的 SQL 文件和迁移工具都从这里运行：
  ```bash
  git clone --depth 1 --branch v0.3.0 https://github.com/brizenchi/go-modules
  cd go-modules
  export DATABASE_URL='postgres://…'     # 生产库；不要写进文件或命令历史
  ```
- 和产品负责人确认第 7 步的手动会员名单。
- 新版代码已经构建好，可以随时部署。

### 步骤

| 步骤 | 做什么 | 期望结果 | 失败怎么办 |
| --- | --- | --- | --- |
| 1 | 备份，停止写入 | 有可恢复的备份 | — |
| 2 | 快照 | `before.txt` | — |
| 3 | 部署新代码，启动一次 | 日志出现 `listening` | 回到旧代码；新表留着无害 |
| 4 | 迁移付费数据 | `check` 只剩已取消订阅的差异 | 修正后重跑，可重复执行 |
| 5 | 迁移第三方登录身份 | `linked` 等于 `legacy oauth users` | 可重复执行 |
| 6 | 迁移积分 | 打印迁移人数和总积分 | 事务回滚，什么都没改 |
| 7 | 登记手动会员 | — | 可重复执行 |
| 8 | 核对 | 快照没有差异，`verify.sql` 只剩已知差异 | 不要恢复写入，先查原因 |
| 9 | 恢复写入 | 冒烟检查通过 | 见[回滚](#回滚) |

**第 1 步：备份，停止写入。** 停掉所有会写数据库的进程：API、worker、定时任务。然后做一份
可恢复的备份：

```bash
pg_dump "$DATABASE_URL" -Fc -f before-upgrade.dump
```

停机期间 Stripe 发不进来的 Webhook 会自动重试（最长 3 天），恢复后会补上，不用手动处理。

**第 2 步：快照。** 记下迁移不能改变的数据：

```bash
psql "$DATABASE_URL" -At -f docs/migrate-from-saascore/snapshot.sql > before.txt
```

**第 3 步：部署新代码，启动一次。** 新代码启动时先自动建表，建完才开始监听端口。保持流量
挡在外面（维护页或者不挂负载均衡），看到日志出现 `listening` 后停掉它。

**第 4 步：迁移付费数据。** 把 `users` 上的 Stripe 字段搬到 billing 表：

```bash
go run ./cmd/legacy-billing-migrate --mode check      # 先看差异，退出码 2 表示有问题待处理
go run ./cmd/legacy-billing-migrate --mode backfill
go run ./cmd/legacy-billing-migrate --mode check
```

saascore 后期已经同时写 billing 表，`backfill` 不会覆盖已有的行，只补缺的行。第二次 `check`
**允许剩下**这一类差异：订阅已取消的用户，`users` 上的订阅 ID 被清空了，billing 表还留着，
报 `subscription_id_mismatch` / `subscription_product_type_mismatch`。两边状态都是 `canceled`，
不用处理。其他差异要先查清楚。加 `--json` 可以输出机器可读的报告。

**第 5 步：迁移第三方登录身份。**

```bash
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f docs/migrate-from-saascore/user_identities.sql
```

saascore 用 Google 返回的 subject 找用户，新模板先查 `user_identities`，查不到再按邮箱找。
不跑这一步的话，邮箱没变过的用户仍能登进原账号；但注册后改过 Google 邮箱的用户，会被当成
新用户，另建一个账号。邮箱验证码注册的用户不需要这一步。

saascore 没有限制同一个 Google subject 只能属于一个账号。如果出现重复，最早注册的账号得到
关联，脚本会打印 `could not be linked` 警告，其余账号在第 8 步 `verify.sql` 第 4 部分列出，需要人工判断。

**第 6 步：迁移积分。**

```bash
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f docs/migrate-from-saascore/credit_ledger.sql
```

每个正余额变成一笔不过期的期初积分，所有用户标记为已迁移。有负余额时脚本会整体回滚，
不改任何数据，要先人工处理这些用户。注意用这个文件，**不要用**
`templates/quickstart/migrations/20260906_credit_ledger.sql`：后者是给旧版模板库用的，需要
`user_credit_grants` 表，还会建模板示例用的 `note_exports` 表。

这一步完成之前，新代码对这些用户的积分操作都会返回 `credit_ledger_migration_required`，
不会算错账。

**第 7 步：登记手动会员。**

```bash
psql "$DATABASE_URL" -f docs/migrate-from-saascore/membership_overrides.sql
```

文件里的查询会列出"套餐不是 free、但没有任何 Stripe 记录"的用户。和产品负责人逐个确认后，
按文件里的示例写入 `membership_overrides`。规则是：`lifetime` 可以不设到期时间；
`starter` / `pro` / `premium` 必须设到期时间，不设会被忽略。想要永久的 pro，就填一个很远的日期。

**第 8 步：核对。**

```bash
psql "$DATABASE_URL" -At -f docs/migrate-from-saascore/snapshot.sql > after.txt
diff before.txt after.txt            # 没有输出才算通过
psql "$DATABASE_URL" -f docs/migrate-from-saascore/verify.sql
```

`verify.sql` 的第 1 到 4 部分应该都是空的。第 5 部分列出旧 `users.plan` 和新模板实际授予的套餐
不一致的用户，并给出原因提示，逐行确认：

- 付款失败、账单周期已过：属于[第 0 节](#上线后用户会感受到的变化)说的行为变化，确认接受即可。
- 没有 Stripe 记录：回到第 7 步。
- 没有 billing 记录：回到第 4 步。

如果改过 `membership.Evaluate`，把第 5 部分的查询同步改掉。

**第 9 步：恢复写入。** 正式启动新版本，放开流量，然后做冒烟检查：

- 用一个老账号分别通过邮箱验证码和 Google 登录，`GET /api/v1/account/profile` 返回的 `id`
  和迁移前一致。
- `GET /api/v1/account/membership` 和 `GET /api/v1/credits` 符合预期。
- 在 Stripe 后台给 Webhook 端点发一条测试事件，返回 2xx。
- 管理后台 `/admin` 能打开用户列表。

### 回滚

| 时间点 | 做法 |
| --- | --- |
| 第 3～8 步任意一步失败 | 先不要恢复写入，查清原因后修复重跑（第 4～7 步都可以重复执行）。要放弃升级，就重新部署旧代码；新建的表和列旧代码不会用到，留着无害。想完全回到原样，用第 1 步的备份恢复。 |
| 第 9 步之后 | 优先修复问题继续往前。回退到旧代码会有数据问题：新代码不再更新 `users.plan` 和 Stripe 相关列，旧代码会看到过期的套餐；旧代码直接改 `users.credits` 也不会记账，再升级时账本对不上。确实需要回退时，用备份恢复，并接受丢失上线后的写入。 |

## 4. 演练

在本机用模拟旧库把第 3 节完整跑一遍：

```bash
scripts/saascore-rehearsal/run.sh          # KEEP=1 保留临时目录里的日志和报告
```

需要 Go、`curl`、`jq` 和 PostgreSQL 服务端程序（`initdb`、`pg_ctl`、`psql`；不在 PATH 上时
设置 `PG_BIN`）。第一次运行会下载 `d6a572f` 的依赖。脚本做的事：

1. 在临时目录起一个只监听本地 socket 的 PostgreSQL，不碰本机已有的数据库。
2. 用 `d6a572f` 的 saascore 代码本身建出旧库表结构，再写入[样例数据](../scripts/saascore-rehearsal/seed.sql)：
   有 Stripe 订阅的（分别是双写和只写 `users` 的）、取消中的、终身的、付款失败的、已取消的、
   有积分的、Google 注册的、有邀请关系的、手动给会员的、管理员；同时用旧代码签发一个 token。
3. 按第 3 节的第 2～8 步执行，所有 SQL 都执行两次，确认可以重复执行。
4. 启动新代码，用每个样例用户登录，比较账号、套餐、积分、角色和
   [`expected.tsv`](../scripts/saascore-rehearsal/expected.tsv) 是否一致；再检查旧 token 返回 401、
   邀请码还在、积分流水只有一条期初、迁移后管理员加积分正常、Google 身份已关联。
5. 在旧库的副本上补测三种情况：两个 SQL 在新代码第一次启动前执行，新代码照样能启动；两个账号
   共用一个 Google subject 时，旧账号保留关联、`verify.sql` 列出另一个；有负余额时积分脚本整体回滚。

最近一次的结果：

```text
email               plan_old  plan_new  expected  cr_old  cr_new  role   same_account
alice@example.com   pro       pro       pro       120     120     user   yes
bob@example.com     starter   starter   starter   0       0       user   yes
carol@example.com   free      free      free      500     500     user   yes
dave@example.com    free      free      free      30      30      user   yes
erin@example.com    free      free      free      70      70      user   yes
frank@example.com   free      free      free      10      10      user   yes
grace@example.com   premium   premium   premium   0       0       user   yes
henry@example.com   lifetime  lifetime  lifetime  0       0       user   yes
ivy@example.com     pro       free      free      15      15      user   yes
jack@example.com    free      free      free      0       0       user   yes
root@example.com    free      free      free      0       0       admin  yes

token signed by saascore -> HTTP 401: every user signs in again
erin keeps referral code INVERIN0001
carol's history: one opening entry of 500
admin grant after the upgrade works: carol now has 505
dave's Google identity is linked

shared subject: user_identities.sql warns
SQL run before the first start: the application starts and carol has 500
shared subject: dave keeps the link, verify.sql lists dave.alias
negative balance: credit_ledger.sql stops and changes nothing

REHEARSAL PASSED
```

`ivy` 从 pro 变成 free 是预期结果：她最后一次付款失败，见[第 0 节](#上线后用户会感受到的变化)第 2 条。

## 5. 接口对照

所有路径都在 `/api/v1` 下，响应格式仍是 `{code, msg, data}`；需要前端区分的错误会在
`data.reason` 里给出固定值。

| 旧接口 | 新版 |
| --- | --- |
| `POST /auth/send-code`、`POST /auth/verify-code`（含 `referral_code`） | 不变 |
| `GET /auth/:provider/authorize`、`GET /auth/:provider/callback` | 浏览器登录需要 `redirect=1&challenge=…`，见[第 2 节](#2-前端修改) |
| `POST /auth/exchange-token`（含 `referral_code`） | 新增必填 `oauth_verifier` |
| `POST /auth/refresh`、`POST /websocket/ticket` | 不变 |
| `POST /auth/logout` | 会吊销当前 token |
| `POST /stripe/webhook`、`/stripe/checkout/session`、`/stripe/subscription/{preview,change,cancel,reactivate}`、`/stripe/portal/session`、`GET /stripe/subscription`、`GET /stripe/invoices` | 不变；付费关闭时返回 503 |
| `GET /referral/code`、`/referral/list`、`/referral/stats` | 不变 |

新增：`GET /capabilities`、`POST /auth/admin/login`、`GET|PATCH /account/profile`、
`GET /account/membership`、`GET /credits`、`GET /credits/transactions`、`GET /billing/purchases`、
`GET /billing/credit-checkouts/:id`、`POST /billing/credit-checkouts/:id/confirmation`、
`POST /stripe/subscription/scheduled-change/cancel`、`/uploads/images*`、`/site/*`、`/admin/*`。
模板示例 `/notes*` 不需要的话可以删掉。

## 6. 常见问题

| 现象 | 原因 |
| --- | --- |
| 积分接口返回 503 `credit_ledger_migration_required` | 第 6 步没跑，或者跑完后又有用户 `credits_version = 0` |
| 积分操作报 `credit ledger balance mismatch` | 有代码绕过 `deps.Credits` 直接改了 `users.credits` |
| Stripe Webhook 日志里找不到用户 | 第 4 步没有 backfill 完，`billing_customers` 缺这个客户 |
| 老用户用 Google 登录后变成新账号 | 第 5 步没跑，或者 `user_identities.sql` 报了无法关联的用户；用 `verify.sql` 第 4 部分查 |
| 会员用户迁移后变成 free | 看 `verify.sql` 第 5 部分的原因提示 |
