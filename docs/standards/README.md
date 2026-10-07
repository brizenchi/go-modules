# 工程规范总览

本目录由 [keel](https://github.com/brizenchi/keel) 生成。通用规范通过
`keel update` 升级（会保留本地修改）；本项目特有的约定写在 [PROJECT.md](./PROJECT.md)。

每一项都标明了**规范在哪里**和**由什么强制执行**。

## 协作
| 规范 | 文档 | 强制执行 |
| --- | --- | --- |
| 分支与提交信息 | [GIT_WORKFLOW.md](./GIT_WORKFLOW.md) | CI `commits / pr-title`；分支 ruleset（`keel github`） |
| PR 与代码审查 | [CODE_REVIEW.md](./CODE_REVIEW.md) | PR 模板、CODEOWNERS、ruleset（approve + 必需的检查） |
| 本地提交前检查 | [CI_QUALITY.md](./CI_QUALITY.md#本地钩子) | `lefthook.yml`（`keel hooks`） |
| CI 与依赖管理 | [CI_QUALITY.md](./CI_QUALITY.md) | `.github/workflows/keel.yml`、Dependabot |

## 代码
| 规范 | 文档 | 强制执行 |
| --- | --- | --- |
| 通用代码规范、命名、错误处理 | [CODE_STYLE.md](./CODE_STYLE.md) | 审查 |
| Go | [GO.md](./GO.md) | gofmt、go vet、golangci-lint、go test -race |
| Node / TypeScript | [NODE.md](./NODE.md) | lint、test、build 脚本 |
| 测试 | [TESTING.md](./TESTING.md) | CI |

## 接口、数据与安全
| 规范 | 文档 | 强制执行 |
| --- | --- | --- |
| API 设计 | [API_STANDARD.md](./API_STANDARD.md) | 审查 |
| 数据库与迁移 | [DATABASE.md](./DATABASE.md) | 审查 |
| 安全与密钥 | [SECURITY_STANDARD.md](./SECURITY_STANDARD.md) | gitleaks（本地 + CI）、GitHub 推送保护、依赖扫描 |
| 漏洞上报 | [SECURITY.md](../../SECURITY.md) | GitHub 私密漏洞报告 |


## 运维
| 规范 | 文档 | 强制执行 |
| --- | --- | --- |
| 故障响应与复盘 | [INCIDENT_RESPONSE.md](./INCIDENT_RESPONSE.md) | 复盘文档 `docs/incidents/` |
| 部署、可观测性 | [PROJECT.md](./PROJECT.md) | 由项目决定 |

## 上手与 AI
| 规范 | 文档 | 强制执行 |
| --- | --- | --- |
| 新成员上手 | [ONBOARDING.md](./ONBOARDING.md) | — |
| AI 编程助手 | [AI_ASSISTANTS.md](./AI_ASSISTANTS.md)、`AGENTS.md`、`CLAUDE.md` | `.claude/settings.json`、Git 钩子、CI |

## 管理 keel

```bash
keel status               # 版本、组件、钩子、GitHub 设置
keel config               # 重新选择组件、修改配置
keel update               # 升级到新版本，保留本地修改
keel github --dry-run     # 预览 GitHub 设置；去掉 --dry-run 应用：squash 合并、分支 ruleset、密钥扫描
```

## 修改规范

- 通用规范：在 keel 仓库修改并发布，各项目 `keel update`；
- 本项目的约定：直接修改 [PROJECT.md](./PROJECT.md) 和 `AGENTS.md` 的项目部分；
- 能用工具检查的新规则加到 CI；只有必须在提交前拦住的才加到 lefthook。
