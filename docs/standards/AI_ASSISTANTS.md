# AI 编程助手

本仓库的规范通过三层交给 AI 助手（Claude Code、Codex、Cursor 等）：**规则入口**告诉它要遵守什么，
**Skill** 告诉它某件事的完整步骤，**自动检查**保证它即使忽略了说明也无法提交不合规的代码。

```text
规范原文（给人看）  docs/standards/*.md、docs/*.md
        │ 提炼
        ▼
规则入口（AI 自动加载）  AGENTS.md（根目录 + 子目录）  ← Codex、Cursor 等直接读取
                         CLAUDE.md（内容只有 @AGENTS.md） ← Claude Code 读取，引用同一份规则
        │
固定流程（按需加载）    .claude/skills/*/SKILL.md
        │
自动检查（强制）        .claude/settings.json（Claude Hooks）→ lefthook（Git 钩子）→ CI
```

## 规则入口

| 文件 | 何时加载 | 内容 |
| --- | --- | --- |
| `AGENTS.md` | 每次会话 | 全局规则：分层、ctx、日志、错误处理、接口、数据库、安全、测试、提交格式、完成前的检查命令 |
| `foundation/AGENTS.md` | 修改 foundation 时 | 依赖限制、只做增量 API 修改、CHANGELOG |
| `modules/AGENTS.md` | 修改模块时 | 端口和适配器、模块之间不互相导入、错误映射 |
| `templates/quickstart/AGENTS.md` | 修改后端模板时 | 各类代码放在哪里、路由分组、越权检查 |
| `templates/quickstart-nextjs/AGENTS.md` | 修改前端时 | 请求封装、错误展示、测试 |
| 各目录的 `CLAUDE.md` | 同上 | 只有一行 `@AGENTS.md`，让 Claude Code 读取同一份规则 |

编写原则：

- **短**：根目录的 `AGENTS.md` 控制在 100 行左右。每条规则都会占用 AI 每次会话的上下文；
- **命令式、可执行**：写"必须用 `slog.InfoContext(ctx, …)`"，而不是"建议注意日志"；
- **写结论，附链接**：详细说明放在 `docs/standards/`，`AGENTS.md` 只写要点和文件路径；
- **用英文写 AGENTS.md**：AI 对英文指令的遵循更稳定；面向人的规范文档使用中文。

## Skill

| Skill | 用途 |
| --- | --- |
| `add-endpoint` | 新增接口或功能：位置、路由分组、handler、越权、前端类型、测试 |
| `add-migration` | 数据库变更：选择 AutoMigrate 还是 SQL 文件、先扩展后收缩、文件格式 |
| `add-third-party` | 接入第三方服务：适配器、注入 HTTPClient、密钥、Webhook、测试、告警 |

Claude Code 会根据任务自动选择合适的 Skill，也可以手动输入 `/add-endpoint` 调用。
其他工具没有 Skill 机制，`AGENTS.md` 里列出了这些文件的路径，它们可以直接阅读。

## 自动检查

| 层 | 内容 | 对谁生效 |
| --- | --- | --- |
| Claude Hooks（`.claude/settings.json`） | 修改 Go 文件后自动执行 gofmt；拦截 `--no-verify`、`git push --force`、强制添加 `.env` | Claude Code |
| Git 钩子（`lefthook.yml`） | 只有密钥扫描和 gofmt：必须在代码离开电脑前完成的检查 | 所有人和所有 AI（安装后） |
| CI | 全部检查；**通过后才部署**，见 [CI_QUALITY.md](./CI_QUALITY.md) | 所有提交，无法跳过 |

**真正重要的规则一定要有自动检查**。AI 可能忽略文字说明，但无法绕过 CI。

## 使用方法

```bash
make hooks          # 每个开发者执行一次，安装 Git 钩子
claude              # Claude Code：自动加载 CLAUDE.md、Skills 和 Hooks
codex               # Codex：自动加载 AGENTS.md
```

确认 AI 读到了规则：

- Claude Code：输入 `/memory`，可以看到当前加载的 `CLAUDE.md` 和它引用的 `AGENTS.md`；
- 任何工具：问它"这个项目的提交信息格式是什么"、"新接口应该放在哪里"，看回答是否正确。

## 维护

- **修改规范时，同步修改 `AGENTS.md`**：先改 `docs/standards/` 里的原文，再更新对应 `AGENTS.md` 中的要点；
- 发现 AI 反复犯同一个错误时：如果能用工具检查，就加到 CI（必须在提交前拦住的，才加到 lefthook）；否则在 `AGENTS.md` 里补一条明确的规则；
- 新增的固定流程（比如"发布新版本"），写成 `.claude/skills/<名称>/SKILL.md`，并在根目录 `AGENTS.md` 里登记；
- 个人偏好（比如回答语言）放在自己的 `~/.claude/CLAUDE.md` 或 `.claude/settings.local.json` 里，不提交到仓库。

## 新项目

用 `make init-quickstart` 创建的新项目会自动带上规则入口、规范文档、Git 钩子配置、Claude Hooks 和 Skills。
见 go-modules 的 [init-quickstart.sh](https://github.com/brizenchi/go-modules/blob/main/scripts/init-quickstart.sh)。
