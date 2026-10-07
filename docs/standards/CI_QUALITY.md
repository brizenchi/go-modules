# CI 质量门禁与依赖管理

## 分工

**CI 是唯一的标准**：所有检查都在 GitHub Actions 中运行，不通过就不能合并。本地只保留必须在代码离开电脑之前完成的检查。

| 位置 | 运行时机 | 检查 | 配置 |
| --- | --- | --- | --- |
| 编辑器 / AI 助手 | 修改文件后 | 格式化；拦截绕过检查的命令 | `.editorconfig` `.claude/settings.json` |
| 本地 Git 钩子 | `git commit` | **密钥扫描** gofmt | `lefthook.yml` |
| CI | 每个 PR 和推送到 `main` | 全部检查 | `.github/workflows/keel.yml` 和项目自己的 workflow |

本地只保留这几项检查的原因：

- **密钥扫描**：CI 在推送之后才运行，而密钥一旦推送就要视为已经泄露，只能在本地拦住；
- **格式化**：不到一秒就能完成，避免 CI 因为格式问题失败；
- 测试、lint、提交信息等其他检查都交给 CI。本地再跑一遍，只会拖慢提交。

## 本地钩子

每个开发者安装一次：

```bash
brew install lefthook gitleaks      # 或者按各工具官网的说明安装
keel hooks                          # 相当于 lefthook install
```

不要用 `git commit --no-verify` 跳过钩子；Claude Code 的 Hooks 会拦住 AI 执行这条命令。

## CI 的组成

`.github/workflows/keel.yml` 由 [keel](https://github.com/brizenchi/keel) 生成，调用的是 keel 里带版本号（`@v2`）的可复用 workflow：

| 检查 | 内容 |
| --- | --- |
| `commits / pr-title` | PR 标题要符合 Conventional Commits（squash 合并后，它就是最终的提交信息） |
| `secrets / gitleaks` | 扫描完整 git 历史中的密钥 |
| `go-* / check` | gofmt、go vet、go test -race |
| `go-* / lint` | golangci-lint |
| `go-* / vuln` | govulncheck（**不阻止合并**） |
| `node-* / check` | 安装依赖后运行配置的脚本（默认 lint、test、build） |
| `node-* / audit` | 生产依赖的漏洞审计（**不阻止合并**） |

本项目使用 `ci_mode=reusable`：`keel.yml` 由项目自己 workflow 中的 `keel` 任务调用，
所以检查名前面会带上 `keel / `，部署等后续任务可以写 `needs: keel`。

项目自己的检查（构建、端到端测试、部署等）写在别的 workflow 文件里，keel 不会改动它们。

漏洞扫描不阻止合并：依赖里新公布的漏洞，不应该挡住一个无关的紧急修复。它报出问题时，按下面"漏洞"一节处理。

## 必需的检查

合并前必须通过的检查列在 `.keel/required-checks.txt` 里，由 `keel github` 写进分支 ruleset。
项目自己 workflow 里的检查，追加到这个文件末尾，再运行一次 `keel github` 即可。

CI 失败时，先在本地复现，不要靠反复重跑来碰运气。偶尔失败的测试要修好，或者建 issue 跟踪。

## 依赖管理

### 自动升级
`.github/dependabot.yml` 每周检查一次依赖，把次版本和补丁版本的升级合并成一个 PR；GitHub Actions 每月检查一次。

- 补丁版本和次版本：CI 通过就可以合并；
- 主版本：先读升级说明，单独处理；
- 互相关联的包（同一个框架的多个子包）一起升级。

### 引入新依赖
在 PR 描述里说明：

1. 为什么需要它，标准库或现有依赖为什么不够用；
2. 维护情况：最近一次发布的时间、issue 有没有人处理；
3. 许可证：MIT、BSD、Apache-2.0 可以直接用；GPL、AGPL 等要先讨论；
4. 会带进多少间接依赖。共享库要保持轻量。

### 漏洞
- 漏洞扫描报出高危漏洞时，1 周内修复，或者确认不受影响并记录原因；
- Dependabot 安全提醒和安全更新由 `keel github` 开启。

## 管理 keel

```bash
keel status                    # 当前版本、组件、钩子、GitHub 设置
keel update                    # 升级到新版本，保留本地修改
keel config                    # 重新回答所有问题
keel enable ai / keel disable dependabot      # 开关组件
keel add go_modules dir=services/api          # 新增要检查的目录
```

每条命令执行完都会列出改动的文件，用 `git diff` 检查之后再提交（`chore: update keel`）。
如果要改语言、目录或组件，请用这些命令，不要直接手改 `keel.yml` 里的任务列表。
