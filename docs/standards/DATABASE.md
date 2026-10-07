# 数据库规范（PostgreSQL + GORM）

## 表的归属

每张表只属于一个所有者，见 [ARCHITECTURE.md](../ARCHITECTURE.md#用户所有权)：

- 模块只创建和迁移自己的表（`auth_*`、`billing_*`、`referral_*`）；
- 宿主 SaaS 拥有 `users`、`user_identities`，以及 `internal/feature/*` 的表；
- 不跨模块直接读写别人的表，通过端口（接口）或事件交互。

## 命名

| 对象 | 规则 | 示例 |
| --- | --- | --- |
| 表 | 小写加下划线，复数；模块的表加模块前缀 | `notes`、`billing_subscriptions`、`auth_email_codes` |
| 字段 | 小写加下划线 | `user_id`、`current_period_end` |
| 主键 | `id` | |
| 外键 | `<被引用表的单数>_id` | `user_id` |
| 时间 | `<动作>_at` | `created_at`、`deleted_at`、`last_login_at` |
| 布尔 | `is_<形容词>` 或描述状态 | `is_active`、`email_verified` |
| 索引 | `idx_<表>_<字段>`；唯一索引用 `uniq_` | `idx_notes_user_id`、`uniq_users_email` |

## 字段类型

| 数据 | 类型 | 说明 |
| --- | --- | --- |
| 主实体 ID | `varchar(36)`，UUID | 用户等会在多处被引用的实体，例如 `users.id` |
| 从属记录、流水的 ID | `bigint` 自增 | 例如 `notes.id`、`user_credit_transactions.id`。如果 ID 会出现在公开链接里、又不希望暴露数据量，改用 UUID |
| 时间 | `timestamptz`，存 **UTC** | GORM 默认的 `time.Time` |
| **金额** | `bigint`，单位是**最小货币单位**（分） | 另外用一个字段存币种；**禁止使用浮点数** |
| 积分、数量 | `bigint` | |
| 枚举 | `varchar(n)` | 合法值在代码里校验，不使用数据库的 enum 类型 |
| 短文本 | `varchar(n)`，`n` 是明确的上限 | |
| 长文本 | `text` | |
| 结构不固定的附加数据 | `jsonb` | 只用于不参与查询和约束的少量数据；结构稳定后改成正式的字段或表 |

会被修改的业务表要有 `created_at`、`updated_at`；只追加、不修改的流水表（比如积分流水）只需要 `created_at`。

## 迁移

当前的方式：

| 场景 | 方式 |
| --- | --- |
| **新建表、新增字段、新增索引** | 修改 GORM 模型，启动时通过 `AutoMigrate` 自动执行（见 `internal/platform/migrate.go`） |
| **修改已有数据、修改字段类型、删除字段、数据回填** | 在 `templates/quickstart/migrations/` 下新增 SQL 文件，**人工备份后手动执行** |

SQL 迁移文件的规则：

- 文件名：`YYYYMMDD_<描述>.sql`，例如 `20260907_auth_token_revocations.sql`；
- 文件开头注释写明：做了什么、执行前要做什么（备份、停写）、怎么回滚；
- 整个文件用 `BEGIN; ... COMMIT;` 包住；
- 尽量写成可以重复执行的语句（`IF NOT EXISTS`、`ON CONFLICT DO NOTHING`）；
- **已经在任何环境执行过的迁移文件，禁止修改**。需要修正时，新增一个文件。

### 不停机变更：先扩展，后收缩

修改字段或删除字段，要分成多次发布，保证新旧两个版本的代码都能正常运行：

```text
发布 1：新增字段（可为空）+ 代码同时写新旧两个字段
发布 2：回填历史数据（SQL 迁移）+ 代码改为读新字段
发布 3：代码不再使用旧字段
发布 4：删除旧字段（SQL 迁移）
```

禁止在同一次发布里"删除字段"和"修改读取这个字段的代码"同时进行。

### 大表注意事项
- 给已有大表加索引时使用 `CREATE INDEX CONCURRENTLY`（不能放在事务里，单独写一个文件）；
- 新增 `NOT NULL` 字段时，先加可空字段并回填，再加约束。

## 查询

- 所有查询都使用 `db.WithContext(ctx)`，否则 SQL 不会出现在请求的链路里，也无法随请求取消；
- 只使用参数化查询（GORM 的 `Where("email = ?", email)`）；**禁止**把用户输入拼接进 SQL；
- 列表查询必须有 `LIMIT`，并且排序要稳定（见 [API_STANDARD.md](./API_STANDARD.md#分页)）；
- 避免 N+1 查询：在循环里查库之前，先考虑 `Preload` 或者 `WHERE id IN (...)`；
- 只查询需要的字段；大字段（`text`、`jsonb`）不出现在列表查询里；
- 慢查询（超过 200ms，可以通过 `db.slow_query_ms` 配置）会以 WARN 级别记录，出现后要处理。

## 事务

- 需要原子性的多步写操作放在一个事务里：`db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {...})`；
- 事务里**不调用外部 HTTP 接口**（比如 Stripe、邮件），避免长时间占用锁；需要时先写数据库，提交后再调用，或者使用 outbox 模式；
- 涉及余额、积分的扣减使用条件更新（`UPDATE ... WHERE balance >= ?`）或者乐观锁（版本号），防止并发时余额变成负数。

## 软删除与数据保留

- 默认使用物理删除；需要保留审计记录或支持恢复的数据，才使用 `deleted_at` 软删除；
- 使用软删除的表，唯一索引要考虑已删除的数据（使用部分索引 `WHERE deleted_at IS NULL`）；
- 用户注销时按隐私政策处理个人数据：删除或者匿名化。

## 本地与测试

- 单元测试和仓储测试使用 SQLite 内存数据库（`gorm.io/driver/sqlite`），见 [TESTING.md](./TESTING.md)；
- 依赖 PostgreSQL 特性的 SQL（`jsonb` 操作符、`CONCURRENTLY`）要在 PostgreSQL 上验证后再合并；
- 生产数据库的连接信息只从环境变量读取，禁止复制生产数据到本地。
