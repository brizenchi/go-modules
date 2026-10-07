# AI 编程助手

规范通过三层交给 AI 助手（Claude Code、Codex、Cursor 等）：**规则入口**告诉它要遵守什么，
**按需加载的说明**告诉它特定任务的步骤，**自动检查**保证它即使忽略了说明也提交不了不合规的代码。

```text
规范原文（给人看）      docs/standards/*.md
        │ 提炼
规则入口（自动加载）    AGENTS.md          ← Codex、Cursor 等读取
                        CLAUDE.md          ← Claude Code 读取，内容是 @AGENTS.md
        │
固定流程（按需加载）    .claude/skills/<名称>/SKILL.md（项目自己添加）
        │
自动检查（强制）        .claude/settings.json → lefthook → CI
```

## AGENTS.md 的结构

| 部分 | 维护者 | 说明 |
| --- | --- | --- |
| 开头到 "Project-specific rules" 之前 | keel | `keel update` 时同步新版本 |
| "Project-specific rules" | **本项目** | 写本项目的架构、目录、命令；更新时会保留 |

子目录可以有自己的 `AGENTS.md`（加上只有一行 `@AGENTS.md` 的 `CLAUDE.md`），只写这个目录特有的规则。

编写原则：

- **短**：根目录的 `AGENTS.md` 控制在 100 行左右，每条规则都会占用 AI 每次会话的上下文；
- **命令式、可执行**：写"必须……"，并给出文件路径和命令；
- **写结论，附链接**：详细内容放在 `docs/standards/`；
- **用英文写 AGENTS.md**：AI 对英文指令的遵循更稳定；给人看的规范用中文。

## 自动检查

| 层 | 内容 | 对谁生效 |
| --- | --- | --- |
| Claude Hooks（`.claude/settings.json`） | 修改文件后格式化（Go、Python、使用 Prettier 的前端）；拦截 `--no-verify`、`git push --force`、强制添加 `.env` | Claude Code |
| Git 钩子（`lefthook.yml`） | 密钥扫描、格式 | 所有人和所有 AI（安装后） |
| CI | 全部检查 | 所有提交，无法跳过 |

**真正重要的规则一定要有自动检查**：AI 可能忽略文字说明，但绕不过 CI。

## Skill（可选）

把"做某件事的完整步骤"（新增接口、数据库迁移、接入第三方服务……）写成
`.claude/skills/<名称>/SKILL.md`，Claude Code 会按需加载。其他工具可以直接阅读这些文件，
记得在 `AGENTS.md` 的项目部分列出它们的路径。

## 验证 AI 是否读到了规则

- Claude Code：输入 `/memory`，查看加载的 `CLAUDE.md` 和它引用的 `AGENTS.md`；
- 任何工具：问它"这个项目的提交信息格式是什么"，看回答是否正确。

## 维护

- 修改通用规范：在 keel 仓库修改并发布新版本，各项目执行 `keel update`；
- 修改本项目的规则：直接编辑 `AGENTS.md` 的项目部分和 [PROJECT.md](./PROJECT.md)；
- AI 反复犯同一个错误时：能用工具检查的加到 CI（必须在提交前拦住的才加到 lefthook），否则在 `AGENTS.md` 里补一条明确的规则；
- 个人偏好放在 `~/.claude/CLAUDE.md` 或 `.claude/settings.local.json`，不提交到仓库。
