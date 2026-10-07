# go-modules：项目约定

> 本文件属于本项目。通用规范由 [keel](https://github.com/brizenchi/keel) 维护，
> 这里记录它们在 go-modules 里的具体做法。

## 架构

- 分层与归属：[ARCHITECTURE.md](../ARCHITECTURE.md)，决策记录：[adr/](../adr)。
- `foundation/*` 只放通用技术能力，不导入 `modules` 和模板（`make purity-check` 检查）；
- `modules/*` 是可复用的业务模块，按 `http → app → domain / port`、`adapter → port / domain` 分层，
  **模块之间不互相导入**，跨模块协作通过事件，由模板订阅；
- `templates/quickstart` 是组合根和产品代码；`templates/quickstart-nextjs` 是前端；
- `foundation/*`、`modules/*` 的公开 API 在同一个大版本内只做增量修改，并更新包内的 `CHANGELOG.md`；
  版本规则见 [VERSIONING.md](../../VERSIONING.md)。

## 目录

| 目录 | 内容 |
| --- | --- |
| `foundation/` | 日志、链路、HTTP 客户端、数据库、Redis、配置等通用能力 |
| `modules/` | auth、billing、email、referral |
| `templates/quickstart/` | 后端模板：`internal/feature/*`（产品功能）、`internal/platform`（服务商选择）、`internal/bootstrap`（启动和事件订阅） |
| `templates/quickstart-nextjs/` | 前端模板 |
| `docs/` | 架构、配置、可观测性、部署、规范 |
| `.claude/skills/` | AI 的固定流程：新增接口、数据库迁移、接入第三方服务 |

## 本地开发

```bash
make hooks                                     # 安装提交前检查
make fmt && make test-race && make purity-check
cd templates/quickstart && cp .env.example .env && go run ./cmd/quickstart
cd templates/quickstart-nextjs && npm ci && npm run dev
```

本地登录：`.env.example` 默认 `APP_EMAIL_PROVIDER=log`、`APP_AUTH_EMAIL_DEBUG=true`，验证码直接显示在登录框里。

## 实现约定

### API（对应 [API_STANDARD.md](./API_STANDARD.md)）
- 响应统一使用 `foundation/httpresp`：`OK`、`BadRequest`、`Unauthorized`、`Forbidden`、`NotFound`、
  `Conflict`、`TooManyRequests`、`InternalError`；503 和需要 `reason` 时用 `Custom`：

  ```go
  httpresp.Custom(c, http.StatusBadRequest, http.StatusBadRequest, msg, gin.H{"reason": "AUTH_INVALID_CODE"})
  ```
- 路由按权限挂在 `hostapi.Groups` 的 `Public`、`User`、`Admin` 分组；
- 列表接口返回 `{items, total, page, limit}`，`limit` 最大 100；
- 需要幂等的后台写操作要求 `Idempotency-Key`（参考 `internal/feature/operations/settings.go`）。

### 错误处理（对应 [CODE_STYLE.md](./CODE_STYLE.md#错误处理)）
- 每个模块在 `domain/errors.go` 定义哨兵错误；
- HTTP 层在一个 `respondAppError` 函数里映射错误，参考 `modules/auth/http/handler.go`；
  `default` 分支用 `slog.ErrorContext` 记一次，并返回固定文案。

### 数据库（对应 [DATABASE.md](./DATABASE.md)）
- 新表、新字段、新索引：修改 GORM 模型，启动时 `AutoMigrate`（`internal/platform/migrate.go`、`internal/bootstrap/host_migrate.go`）；
- 修改已有数据、改类型、删字段：新增 `templates/quickstart/migrations/YYYYMMDD_<描述>.sql`，**备份后手动执行**；
- 主实体（`users`）用 `varchar(36)` UUID，从属记录和流水（`notes`、积分流水）可以用自增 `bigint`；
- 测试使用 SQLite 内存数据库；SQL 日志和链路默认不带参数值（`foundation/pgx`）。

### 外部调用
- 模板里所有第三方 HTTP 调用使用注入的 `platform.Config.HTTPClient`（带链路、指标和日志）；
- 适配器接受可选的 `HTTPClient`，并把 `ctx` 传给每一次 SDK 调用（Stripe 参数里的 `Context`）。

### 前端（对应 [NODE.md](./NODE.md)）
- 所有请求通过 `lib/api.ts` 的 `apiRequest`；类型也定义在这里，和后端在同一个 PR 里修改；
- 失败时抛出 `ApiError`：`status`、`code`、`message`（不直接展示）、`reason`、`requestId`；
- 用户提示参考 `components/console-kit.tsx` 的 `ConsoleError` 和 `lib/request-state.ts` 的 `describeRequestFailure`，
  5xx 显示请求编号；
- 幂等写操作用 `newIntentKey()` 生成 key，重试时复用；
- 文案使用 `t({ en, zh })`（`lib/i18n.tsx`）；环境变量只在 `lib/env.ts` 读取；登录状态只通过 `lib/auth.ts`。

### 日志与可观测性
- 日志使用 `slog.*Context(ctx, …)`，`foundation/slog` 自动补充 `request_id`、`trace_id` 并脱敏；
- 字段、级别、链路、指标、告警：[OBSERVABILITY.md](../OBSERVABILITY.md)；
- 测试日志：`flog.Setup(flog.Config{Format: flog.FormatJSON, Output: &buf})`；测试链路：`tracetest.NewSpanRecorder()`。

## 部署

- **CI 全部通过后才部署**：`.github/workflows/ci.yml` 的 `deploy` 任务调用 Dokploy API（`scripts/deploy-dokploy.sh`）；
- 环境、一次性配置、回滚、发布后检查和上线验收清单：[DEPLOYMENT.md](../DEPLOYMENT.md)。

## 项目特有的检查

keel 的检查（`keel / …`）之外，`.github/workflows/ci.yml` 还运行：

| 检查 | 内容 |
| --- | --- |
| `template-quickstart` | 模板的 Linux 构建，以及脱离 workspace 的独立构建（`scripts/verify-quickstart-release.sh`） |
| `go mod tidy` | `go.mod`、`go.sum` 已整理 |
| `pkg purity` | 共享包没有导入宿主代码 |
| `observability-config` | 告警规则测试（promtool）、Alloy 配置检查 |
| `deploy` | 推送到 `main` 且以上检查全部通过后部署 |

这些检查也写在 `.keel/required-checks.txt` 末尾，由 `keel github` 设为必需。

## 其他

- `.gitleaksignore` 记录了引入 gitleaks 之前已有的测试假密钥（指纹），新增的假密钥必须用 `*_not-a-real-key` 写法；
- `templates/quickstart` 被复制成新项目时，用 `make init-quickstart`，它会同时安装 keel。
