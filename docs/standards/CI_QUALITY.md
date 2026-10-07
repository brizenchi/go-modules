# CI 质量门禁与依赖管理

## 分工

**CI 是唯一的标准**：所有检查都在 GitHub Actions 中运行，不通过就不能合并。本地只保留必须在代码离开电脑前完成的检查。

| 位置 | 运行时机 | 检查 | 配置 |
| --- | --- | --- | --- |
| 编辑器 / AI 助手 | 修改文件后 | 格式化；拦截绕过检查的命令 | `.editorconfig`、`.claude/settings.json` |
| 本地 Git 钩子 | `git commit` | **密钥扫描**、gofmt | `lefthook.yml` |
| CI | 每个 PR 和推送到 `main` | 全部检查 | `.github/workflows/standards.yml` |

为什么本地只留这几项：

- **密钥扫描**：CI 在推送之后才运行，密钥一旦推送就要视为泄露，只能在本地拦住；
- **格式化**：不到一秒，避免 CI 因为格式问题失败；
- 其他检查（测试、lint、提交信息）统一交给 CI，本地再跑一遍只会拖慢提交。

## 本地钩子

每个开发者安装一次：

```bash
brew install lefthook gitleaks      # 或参考各自官网的安装方式
lefthook install
```

不要用 `git commit --no-verify` 跳过钩子；Claude Code 的 Hooks 会拦截 AI 执行这个命令。

## CI 的组成

`.github/workflows/standards.yml` 由 dev-standards 生成，调用
[brizenchi/dev-standards](https://github.com/brizenchi/dev-standards) 中带版本号（`@v1`）的可复用 workflow：

| 检查 | 内容 |
| --- | --- |
| `commits / pr-title` | PR 标题符合 Conventional Commits（squash 合并后成为提交信息） |
| `secrets / gitleaks` | 扫描完整 git 历史中的密钥 |
| `go-* / check` | gofmt、go vet、go test -race |
| `go-* / lint` | golangci-lint |
| `go-* / vuln` | govulncheck（**不阻止合并**） |
| `node-* / check` | 安装依赖并运行配置的脚本（默认 lint、test、build） |
| `node-* / audit` | 生产依赖漏洞审计（**不阻止合并**） |

本项目使用 `ci_mode=reusable`：`standards.yml` 由项目自己的 workflow 中的 `standards` 任务调用，
因此检查名带有 `standards / ` 前缀，部署等后续任务可以 `needs: standards`。

项目自己的检查（构建、端到端测试、部署等）写在其他 workflow 文件里，dev-standards 不会修改它们。

漏洞扫描不阻止合并：依赖里新公布的漏洞，不应该挡住一个无关的紧急修复。它失败时按下面"漏洞"一节处理。

## 必需的检查

合并前必须通过的检查列在 `.github/required-checks.txt`，由 `.standards/bin/setup-github` 写入分支 ruleset。
项目自己 workflow 中的检查，追加到这个文件后重新运行 `setup-github` 即可。

CI 失败时先在本地复现，不要靠反复重跑碰运气；偶发失败的测试要修复，或者建 issue 跟踪。

## 依赖管理

### 自动升级
`.github/dependabot.yml` 每周检查依赖，按"次版本和补丁版本"合并成一个 PR；GitHub Actions 每月检查。

- 补丁版本和次版本：CI 通过即可合并；
- 主版本：阅读升级说明，单独处理；
- 相互关联的包（同一个框架的多个子包）一起升级。

### 引入新依赖
在 PR 描述中说明：

1. 为什么需要，标准库或现有依赖为什么不行；
2. 维护情况：最近的发布时间、issue 是否有人处理；
3. 许可证：MIT、BSD、Apache-2.0 可以直接使用；GPL、AGPL 等先讨论；
4. 引入了多少间接依赖。共享库要保持轻量。

### 漏洞
- 漏洞扫描报告高危漏洞时，1 周内修复，或确认不受影响并记录原因；
- Dependabot 安全提醒和安全更新由 `setup-github` 开启。

## 升级 dev-standards

```bash
uvx copier update              # 合并新版本的规范，保留本地修改
uvx copier update --defaults   # 不重新回答问题
```

解决冲突标记（如果有）后提交：`chore: update dev-standards`。修改目录或语言配置也通过
`uvx copier update` 重新回答问题完成，不要直接手改 `standards.yml` 的任务列表。
