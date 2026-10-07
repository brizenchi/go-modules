# 新成员上手指南

## 第一天

### 1. 安装通用工具
```bash
brew install git gh lefthook gitleaks uv
```
Go：版本以 `go.mod` 为准（`brew install go`）。
Node：版本见 `.copier-answers.yml` 中的 `node_projects`（默认 22），包管理器以锁文件为准。

### 2. 获取代码并安装钩子
```bash
git clone git@github.com:brizenchi/go-modules.git
cd go-modules
lefthook install          # 提交前检查：密钥扫描、格式
```

### 3. 运行项目
按 [PROJECT.md](./PROJECT.md#本地开发) 中的步骤启动项目。

## 第一周：阅读

1. [PROJECT.md](./PROJECT.md)：本项目的架构、目录、部署和特有约定；
2. [规范总览](./README.md)，重点：
   - [GIT_WORKFLOW.md](./GIT_WORKFLOW.md)：分支、提交信息、PR；
   - [CODE_STYLE.md](./CODE_STYLE.md) 和对应语言的规范；
   - [API_STANDARD.md](./API_STANDARD.md)、[TESTING.md](./TESTING.md)。

## 第一个 PR

1. `git switch -c fix/<scope>-<简述>`；
2. 修改代码并补上测试，在本地运行 PR 模板中列出的检查；
3. `git commit -m "fix(<scope>): <说明>"`；
4. 推送并创建 PR，按模板填写，标题和提交信息格式一致；
5. 根据审查意见修改，CI 通过后 squash 合并。

## 使用 AI 助手

根目录的 `AGENTS.md`（Codex、Cursor 等读取）和 `CLAUDE.md`（Claude Code 读取）包含这些规范的要点，
AI 会自动遵守。生成的代码仍然由你负责审查和测试。详见 [AI_ASSISTANTS.md](./AI_ASSISTANTS.md)。

## 权限

找维护者开通需要的权限（仓库写权限、监控平台只读、第三方服务的测试账号）。
生产环境的密钥不提供给个人，本地开发使用测试环境的 key。
