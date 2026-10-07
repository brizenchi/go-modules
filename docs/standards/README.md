# 工程规范总览

多人协作和 AI 助手都遵守同一套规范。每一项都标明了**规范在哪里**、**由什么强制执行**。
能用工具检查的规则都已经自动化，不依赖人记住。

> 在由 `init-quickstart` 生成的项目中，路径 `templates/quickstart` 对应 `backend/`，
> `templates/quickstart-nextjs` 对应 `frontend/`；只涉及 `foundation/*`、`modules/*` 的条目
> 适用于 go-modules 仓库本身。

## 规范清单

### 协作流程
| # | 规范 | 文档 | 强制执行 |
| --- | --- | --- | --- |
| 1 | Git 分支与提交信息 | [GIT_WORKFLOW.md](./GIT_WORKFLOW.md) | CI `pr-title` |
| 2 | PR 与代码审查 | [CODE_REVIEW.md](./CODE_REVIEW.md)、[PR 模板](../../.github/pull_request_template.md)、[CODEOWNERS](../../.github/CODEOWNERS) | 分支保护：需要 1 个 approve 和全部 CI 通过 |
| 3 | 版本与发布 | go-modules 的 [VERSIONING.md](https://github.com/brizenchi/go-modules/blob/main/VERSIONING.md) | release workflow |
| 4 | 本地提交前检查（密钥、格式） | [CI_QUALITY.md](./CI_QUALITY.md#本地钩子) | `lefthook.yml`（`make hooks`） |
| 5 | 架构决策记录 | [docs/adr](../adr) | 审查 |

### 代码
| # | 规范 | 文档 | 强制执行 |
| --- | --- | --- | --- |
| 6 | 代码风格与静态检查 | [CODE_STYLE.md](./CODE_STYLE.md)、`.editorconfig` | gofmt（钩子 + CI）、golangci-lint、ESLint |
| 7 | 分层与目录 | [ARCHITECTURE.md](../ARCHITECTURE.md) | `make purity-check`（CI） |
| 8 | 命名 | [CODE_STYLE.md](./CODE_STYLE.md#命名) | 审查 |
| 9 | 错误处理 | [CODE_STYLE.md](./CODE_STYLE.md#错误处理) | errcheck（golangci-lint）、审查 |

### 接口
| # | 规范 | 文档 | 强制执行 |
| --- | --- | --- | --- |
| 10 | API 设计 | [API_STANDARD.md](./API_STANDARD.md) | `httpresp`、审查 |
| 11 | 错误码与 `reason` | [API_STANDARD.md](./API_STANDARD.md#错误码) | 前端 `ApiError.reason` |
| 12 | 接口契约 | [API_STANDARD.md](./API_STANDARD.md#接口契约) | 审查（计划引入 OpenAPI 生成） |

### 数据
| # | 规范 | 文档 | 强制执行 |
| --- | --- | --- | --- |
| 13 | 数据库与迁移 | [DATABASE.md](./DATABASE.md) | 审查；SQL 日志和链路默认不带参数值 |
| 14 | 配置 | [CONFIG_STANDARD.md](../CONFIG_STANDARD.md) | 启动时校验；`config.yaml.example` 的 schema 测试 |
| 15 | 密钥管理 | [SECURITY_STANDARD.md](./SECURITY_STANDARD.md#密钥管理) | gitleaks（钩子 + CI）、GitHub 推送保护、日志脱敏 |

### 可观测性与运维
| # | 规范 | 文档 | 强制执行 |
| --- | --- | --- | --- |
| 16 | 日志 | [OBSERVABILITY.md](../OBSERVABILITY.md#日志字段) | `foundation/slog`（字段、脱敏、去重） |
| 17 | 指标、告警、SLO | [OBSERVABILITY.md](../OBSERVABILITY.md#告警) | `deploy/alerts`（promtool 测试，CI） |
| 18 | 故障响应与复盘 | [INCIDENT_RESPONSE.md](./INCIDENT_RESPONSE.md) | 复盘文档 `docs/incidents/` |
| 19 | 部署与环境 | [DEPLOYMENT.md](./DEPLOYMENT.md) | **CI 全部通过后才部署**（`deploy` 任务）；健康检查与自动回滚 |

### 质量
| # | 规范 | 文档 | 强制执行 |
| --- | --- | --- | --- |
| 20 | 测试 | [TESTING.md](./TESTING.md) | CI `test -race`、覆盖率 |
| 21 | CI 质量门禁 | [CI_QUALITY.md](./CI_QUALITY.md#必需的检查) | `.github/workflows/*`、分支保护 |
| 22 | 依赖管理 | [CI_QUALITY.md](./CI_QUALITY.md#依赖管理) | Dependabot、govulncheck、npm audit |

### 安全
| # | 规范 | 文档 | 强制执行 |
| --- | --- | --- | --- |
| 23 | 安全编码 | [SECURITY_STANDARD.md](./SECURITY_STANDARD.md) | gosec（golangci-lint）、审查 |
| 24 | 漏洞上报 | [SECURITY.md](../../SECURITY.md) | GitHub 私密漏洞报告 |
| 25 | 审计 | [SECURITY_STANDARD.md](./SECURITY_STANDARD.md#鉴权与越权) | 后台审计日志 |

### 前端
| # | 规范 | 文档 | 强制执行 |
| --- | --- | --- | --- |
| 26 | 前端代码 | [FRONTEND.md](./FRONTEND.md) | ESLint、`npm run verify`（CI） |
| 27 | 前端请求与错误展示 | [FRONTEND.md](./FRONTEND.md#请求规范) | `lib/api.ts`（`ApiError.requestId` / `reason`） |

### 文档与上手
| # | 规范 | 文档 | 强制执行 |
| --- | --- | --- | --- |
| 28 | 包文档 | 每个包的 `README.md` 和 `CHANGELOG.md` | PR 模板 |
| 29 | 新成员上手 | [ONBOARDING.md](./ONBOARDING.md) | — |

### AI 助手
| # | 规范 | 文档 | 强制执行 |
| --- | --- | --- | --- |
| 30 | AI 规则入口与自动化 | [AI_ASSISTANTS.md](./AI_ASSISTANTS.md)、`AGENTS.md`、`CLAUDE.md`、`.claude/skills` | `.claude/settings.json`（Hooks）、Git 钩子、CI |

## 修改规范

1. 修改本目录下的文档，在 PR 中说明原因；
2. 同步更新根目录和各子目录 `AGENTS.md` 中对应的要点（见 [AI_ASSISTANTS.md](./AI_ASSISTANTS.md#维护)）；
3. 能用工具检查的新规则，加到 CI；只有必须在提交前拦住的（比如密钥），才加到 lefthook。
