# Git 工作流规范

## 分支

`main` 是唯一的长期分支，并且始终可以部署：推送到 `main` 后，CI 全部通过才会自动部署到生产环境（见 [DEPLOYMENT.md](./DEPLOYMENT.md#发布流程)）。

| 分支 | 命名 | 用途 |
| --- | --- | --- |
| 功能 | `feat/<scope>-<简述>` | 新功能，例：`feat/billing-annual-plan` |
| 修复 | `fix/<scope>-<简述>` | 缺陷修复，例：`fix/auth-oauth-state-replay` |
| 其他 | `chore/`、`docs/`、`refactor/`、`test/`、`ci/` | 与提交类型一致 |
| 紧急修复 | `hotfix/<简述>` | 线上故障，见下文 |

- 分支名只用小写字母、数字和 `-`；
- 一个分支只做一件事，生命周期尽量不超过 3 天；
- 禁止直接推送 `main`，所有改动都通过 PR 合并（见"分支保护"）。

## 提交信息：Conventional Commits

```text
<type>(<scope>): <subject>

<body：为什么改、怎么改，可选>

<footer：BREAKING CHANGE / Refs #123，可选>
```

| type | 含义 | 是否出现在 CHANGELOG |
| --- | --- | --- |
| `feat` | 新功能 | 是 |
| `fix` | 修复缺陷 | 是 |
| `perf` | 性能优化 | 是 |
| `refactor` | 重构，行为不变 | 否 |
| `docs` | 只改文档 | 否 |
| `test` | 只改测试 | 否 |
| `build` | 依赖、构建、Dockerfile | 否 |
| `ci` | CI 配置 | 否 |
| `chore` | 其他杂项 | 否 |
| `revert` | 回滚某个提交 | 是 |

**scope** 使用改动所在的包或模块名：`tracing`、`ginx`、`slog`、`httpx`、`pgx`、`auth`、
`billing`、`email`、`referral`、`quickstart`、`nextjs`、`deploy`、`docs`、`ci`。
跨多个包时，选影响最大的那个，或者省略 scope。

规则：

- subject 用祈使语气，中英文均可，不以句号结尾，不超过 72 个字符；
- 不写 `feat`、`update`、`fix bug` 这类没有信息量的提交信息；
- 破坏性变更在 type 后加 `!`，并在 footer 写 `BREAKING CHANGE: <迁移方式>`；
- 一个提交只做一件事；格式化、重命名这类改动和功能改动分开提交。

示例：

```text
feat(tracing): honour standard OTEL_EXPORTER_OTLP_* variables
fix(billing): pass request ctx to Stripe calls
docs(standards): add API error code table
feat(auth)!: require state cookie for OAuth callback

BREAKING CHANGE: clients must keep the oauth_flow cookie between authorize and callback.
```

CI 的 `pr-title` 检查 PR 标题是否符合这个格式（见 [CI_QUALITY.md](./CI_QUALITY.md)）；本地可以用 `./scripts/check-commit-msg.sh` 提前检查。

## Pull Request

1. 从最新的 `main` 拉分支，完成后推送分支并创建 PR；
2. **PR 标题必须符合提交信息格式**：合并时使用 squash，PR 标题就是最终进入 `main` 的提交信息；
3. 按 [PR 模板](../../.github/pull_request_template.md) 填写：改了什么、为什么、怎么验证的、风险和回滚方式；
4. CI 全部通过，并获得一个 approve 后才能合并（审查要求见 [CODE_REVIEW.md](./CODE_REVIEW.md)）；
5. 合并方式统一用 **Squash and merge**，合并后删除分支。

PR 的大小：

- 尽量控制在 400 行（不含生成代码和测试数据）以内；
- 大功能拆成多个可以独立合并的 PR，未完成的部分用配置开关关闭。

## 历史改写

- **禁止对 `main` 或任何已经推送、已有他人使用的分支执行 force push**；
- 只有在提交尚未推送时，才可以 amend、rebase 或 squash；
- 如果误把密钥推送到了远程：立即**作废并轮换该密钥**，然后按
  [SECURITY_STANDARD.md](./SECURITY_STANDARD.md#密钥泄露处理) 处理。改写历史不能撤回已经泄露的密钥。

## 紧急修复（hotfix）

1. 从 `main` 拉出 `hotfix/<简述>` 分支，只修复问题本身；
2. PR 标题用 `fix(<scope>): ...`，可以由一名审查者快速 approve；
3. 合并后确认部署结果，并在 24 小时内补上测试和故障复盘（见 [INCIDENT_RESPONSE.md](./INCIDENT_RESPONSE.md)）。

如果一时修不好，优先回滚：在 Dokploy 里回滚到上一个版本，或者对出问题的提交执行
`git revert` 并走 PR。

## 版本与发布

版本号和打 tag 的流程见 go-modules 仓库的 [VERSIONING.md](https://github.com/brizenchi/go-modules/blob/main/VERSIONING.md)。改动了 `foundation/*` 或
`modules/*` 公开 API 的 PR，必须同时更新对应包的 `CHANGELOG.md`。

## 分支保护（仓库设置）

GitHub → Settings → Branches → `main`：

- Require a pull request before merging（Require approvals: 1）；
- Require status checks to pass：勾选 [CI_QUALITY.md](./CI_QUALITY.md#必需的检查) 中列出的检查；
- Require linear history；
- 不允许 force push，不允许删除分支；
- 单人维护阶段可以勾选 "Allow specified actors to bypass"，把自己加进去用于紧急修复，但日常仍走 PR。
