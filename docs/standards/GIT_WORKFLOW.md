# Git 工作流规范

## 分支

`main` 是唯一的长期分支，始终保持可发布状态。

| 分支 | 命名 | 用途 |
| --- | --- | --- |
| 功能 | `feat/<scope>-<简述>` | 新功能，例：`feat/billing-annual-plan` |
| 修复 | `fix/<scope>-<简述>` | 缺陷修复，例：`fix/auth-token-expiry` |
| 其他 | `chore/`、`docs/`、`refactor/`、`test/`、`ci/` | 与提交类型一致 |
| 紧急修复 | `hotfix/<简述>` | 线上故障，见下文 |

- 分支名只用小写字母、数字和 `-`；
- 一个分支只做一件事，生命周期尽量不超过 3 天；
- 所有改动都通过 PR 合并到 `main`（见"分支保护"）。

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
| `build` | 依赖、构建 | 否 |
| `ci` | CI 配置 | 否 |
| `chore` | 其他杂项 | 否 |
| `revert` | 回滚某个提交 | 是 |

**scope** 写改动所在的模块或目录，本项目常用的有：`tracing`、`ginx`、`slog`、`httpx`、`pgx`、`rdx`、`auth`、`billing`、`email`、`referral`、`quickstart`、`nextjs`、`deploy`、`docs`、`ci`。跨多个模块时，选影响最大的那个，或者省略 scope。

规则：

- subject 用祈使语气，中英文均可，不以句号结尾，不超过 72 个字符；
- 不写 `update`、`fix bug` 这类没有信息量的提交信息；
- 破坏性变更在 type 后加 `!`，并在 footer 写 `BREAKING CHANGE: <迁移方式>`；
- 一个提交只做一件事；格式化、重命名和功能改动分开提交。

示例：

```text
feat(api): add pagination to invoice list
fix(auth): reject expired refresh tokens
docs(standards): clarify migration rollback
feat(auth)!: require PKCE for OAuth login

BREAKING CHANGE: clients must send code_verifier on the token exchange.
```

CI 的 `commits / pr-title` 检查 PR 标题是否符合格式。本地可以用
`keel lint-commit` 提前检查。

## Pull Request

1. 从最新的 `main` 拉分支，完成后推送并创建 PR；
2. **PR 标题必须符合提交信息格式**：合并时使用 squash，PR 标题就是最终的提交信息；
3. 按 PR 模板填写：改了什么、为什么、怎么验证的、风险和回滚方式；
4. CI 全部通过并获得要求的 approve 后才能合并（审查要求见 [CODE_REVIEW.md](./CODE_REVIEW.md)）；
5. 统一用 **Squash and merge**，合并后自动删除分支。

PR 的大小：尽量控制在 400 行以内（不含生成代码和测试数据）；大功能拆成多个可以独立合并的 PR，
未完成的部分用配置开关关闭。

## 历史改写

- **禁止对 `main` 或任何已推送、他人在用的分支 force push**；
- 只有在提交尚未推送时，才可以 amend、rebase 或 squash；
- 误把密钥推送到了远程：立即**作废并轮换该密钥**，再按
  [SECURITY_STANDARD.md](./SECURITY_STANDARD.md#密钥泄露处理) 处理。改写历史撤回不了已经泄露的密钥。

## 紧急修复（hotfix）

1. 从 `main` 拉出 `hotfix/<简述>`，只修复问题本身；
2. PR 标题用 `fix(<scope>): ...`，可以由一名审查者快速 approve；
3. 合并后确认发布结果，24 小时内补上测试和故障复盘（见 [INCIDENT_RESPONSE.md](./INCIDENT_RESPONSE.md)）。

一时修不好时优先回滚：对出问题的提交执行 `git revert` 并走 PR，或者使用部署平台的回滚功能。

## 分支保护

运行 `keel github`（需要 `gh` 和仓库管理员权限）会创建名为 `keel` 的 ruleset：

- 禁止删除和 force push 默认分支，要求线性历史；
- 必须通过 PR 合并，只允许 squash，未解决的评论不能合并；
- 必须通过 `.keel/required-checks.txt` 中列出的检查；
- 默认要求 1 个 approve；单人维护时用 `--solo`（不要求 approve，但仍然必须走 PR 和 CI）；
- 仓库管理员可以绕过（紧急情况使用），用 `--no-admin-bypass` 关闭。

修改规则后重新运行即可，脚本是幂等的。
