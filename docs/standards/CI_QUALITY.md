# CI 质量门禁与依赖管理

## 各层的分工

**线上 CI 是唯一的标准**：所有检查都在这里执行，并且**只有全部通过后才会部署**。本地只保留
必须在代码离开电脑之前完成的两项检查。

| 位置 | 运行时机 | 检查内容 | 配置文件 |
| --- | --- | --- | --- |
| 编辑器 / AI 助手 | 保存或修改文件后 | 格式化；拦截 AI 绕过检查的命令 | `.editorconfig`、`.claude/settings.json` |
| 本地 Git 钩子 | `git commit` | **只有密钥扫描和 gofmt** | `lefthook.yml` |
| CI（GitHub Actions） | 每个 PR 和推送到 `main` | 全部检查；通过后部署 | `.github/workflows/*.yml` |

为什么只在本地保留这两项：

- **密钥扫描**：CI 在推送之后才运行，密钥一旦推送到 GitHub 就要视为已经泄露，必须在本地拦住；
- **gofmt**：不到一秒，避免 CI 因为格式问题失败后再来回修改；
- 其他检查（测试、lint、提交信息……）本地再跑一遍只会拖慢提交，统一交给 CI。

## 本地钩子

安装一次即可（每个开发者都要执行）：

```bash
brew install lefthook gitleaks     # 或者：go install github.com/evilmartians/lefthook@latest
make hooks                          # 等同于 lefthook install
```

| 钩子 | 检查 |
| --- | --- |
| `pre-commit` | 暂存的 Go 文件格式（gofmt）；暂存内容的密钥扫描（gitleaks） |

提交信息的格式由 CI 的 `pr-title` 检查（squash 合并后 PR 标题就是提交信息）。想在本地提前检查时，
可以运行 `./scripts/check-commit-msg.sh .git/COMMIT_EDITMSG`。

不要使用 `git commit --no-verify` 跳过钩子；Claude Code 的 Hooks 会拦截 AI 执行这个命令。

## 必需的检查

在分支保护里把下面这些设为 Required（见 [GIT_WORKFLOW.md](./GIT_WORKFLOW.md#分支保护仓库设置)）：

| 检查 | 内容 |
| --- | --- |
| `test` | 构建 + `go test -race -cover` + `go vet`（foundation、modules） |
| `template-quickstart` | 模板测试、构建、脱离 workspace 的独立构建 |
| `template-nextjs` | 前端测试、lint、构建、`npm audit` |
| `gofmt` | 格式 |
| `go mod tidy` | `go.mod` 和 `go.sum` 已整理 |
| `pkg purity` | 共享包没有导入宿主代码 |
| `golangci-lint` | 静态检查 |
| `govulncheck` | Go 依赖漏洞 |
| `secrets` | gitleaks 扫描本次改动 |
| `pr-title` | PR 标题符合提交信息格式（squash 合并后会成为提交信息） |
| `observability-config` | 告警规则测试（promtool）、Alloy 配置检查 |

`govulncheck` 也会运行，但**不阻止部署**：依赖里新公布的漏洞，不应该挡住一个无关的紧急修复。
它失败时按下面"漏洞"一节处理。

## 部署门禁

`ci.yml` 中的 `deploy` 任务在推送到 `main`、并且上面除 `govulncheck` 以外的检查**全部通过**后，
调用 Dokploy API 部署（`scripts/deploy-dokploy.sh`）。配置方法见
[DEPLOYMENT.md](./DEPLOYMENT.md#发布流程)。

- CI 失败 = 不部署。修复后再推送一次即可；
- 连续推送多次时，只有最新的提交会被部署：较早的提交通过 CI 时如果 `main` 已经前进，它会跳过部署，
  避免 Dokploy 拉取到还没通过 CI 的最新代码。

CI 失败时，先在本地复现；不要靠反复重跑碰运气通过。确实是偶发失败的测试，要修复，或者标记出来并建 issue 跟踪。

## 依赖管理

### 自动升级
`.github/dependabot.yml` 每周检查以下依赖，并提交 PR：

- 根模块的 Go 依赖、模板的 Go 依赖；
- 前端的 npm 依赖；
- GitHub Actions 的版本。

同一类依赖会合并成一个 PR（比如所有 `go.opentelemetry.io/*`），避免产生大量 PR。

### 处理升级 PR
- 补丁版本和次版本：CI 通过就可以合并；
- 主版本：阅读升级说明，必要时修改代码，单独处理；
- OTel 的 `contrib` 包和 `otel` 主包要一起升级，版本号要对应。

### 引入新依赖的要求
在 PR 描述中说明：

1. 为什么需要它，标准库或现有依赖为什么不能满足；
2. 维护情况：最近一次发布的时间、star 数、issue 是否有人处理；
3. 许可证：MIT、BSD、Apache-2.0 可以直接使用；GPL、AGPL 等需要先讨论；
4. 引入了多少间接依赖。`foundation/*` 要保持轻量（见 go-modules 的 [CONTRIBUTING.md](https://github.com/brizenchi/go-modules/blob/main/CONTRIBUTING.md)）：之前放弃 gorm 的 OTel 插件，就是因为它会带进 ClickHouse、MySQL 等无关的数据库驱动。

### 漏洞
- `govulncheck` 和 `npm audit` 报告高危漏洞时，在 1 周内修复，或者确认它不影响我们并记录原因；
- GitHub 的 Dependabot 安全提醒要打开（Settings → Code security）。
