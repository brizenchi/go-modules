# 项目约定

> 本文件属于本项目。通用规范由 [keel](https://github.com/brizenchi/keel) 维护，
> 这里记录它们在本项目里的具体做法。本项目由 go-modules 的 quickstart 模板创建。

## 架构

- `backend/`：Go 后端，组合根和产品代码；共享能力来自 `github.com/brizenchi/go-modules`
  （`foundation/*` 通用技术能力、`modules/*` 登录、支付、邮件、邀请），**在上游修改，不要复制进来**；
- `frontend/`：Next.js 前端；
- 分层与归属见 [ARCHITECTURE.md](../ARCHITECTURE.md)。

## 目录

| 目录 | 内容 |
| --- | --- |
| `backend/internal/feature/*` | 产品功能（参考 `note`：`note.go`、`repository.go`、`service.go`、`handler.go`） |
| `backend/internal/platform` | 服务商选择、模块组装、路由挂载 |
| `backend/internal/bootstrap` | 启动、事件订阅、业务回调、后台任务 |
| `backend/migrations` | 手动执行的 SQL 迁移 |
| `backend/deploy` | 配置示例、Alloy 日志采集、告警规则 |
| `frontend/lib` | 请求客户端（`api.ts`）、状态、环境变量 |

## 本地开发

```bash
lefthook install
cd backend && cp .env.example .env && go run ./cmd/quickstart
cd frontend && npm ci && npm run dev
```

## 实现约定

- API：响应使用 `foundation/httpresp`；`reason` 用 `httpresp.Custom` 放在 `data` 里；路由挂在 `Public`、`User`、`Admin` 分组；
- 错误：模块的哨兵错误在 `domain/errors.go`，HTTP 层用一个 `respondAppError` 映射；
- 数据库：新表、新字段走 GORM `AutoMigrate`；修改已有数据走 `backend/migrations/YYYYMMDD_<描述>.sql`，备份后手动执行；
- 外部调用：使用注入的 `platform.Config.HTTPClient`；
- 前端：请求统一通过 `frontend/lib/api.ts`，错误为 `ApiError`（`status`、`reason`、`requestId`），5xx 显示请求编号；
- 日志与可观测性：见 [OBSERVABILITY.md](../OBSERVABILITY.md)。

## 部署

CI 全部通过后由 `deploy` 任务调用 Dokploy 部署；配置和上线验收清单见 [DEPLOYMENT.md](../DEPLOYMENT.md)。

## 项目特有的检查

| 检查 | 内容 |
| --- | --- |
| `observability-config` | 告警规则测试、Alloy 配置检查 |
| `deploy` | 推送到 `main` 且检查全部通过后部署 |
