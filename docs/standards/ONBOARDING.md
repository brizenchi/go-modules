# 新成员上手指南

## 第一天：把项目跑起来

### 1. 安装工具
```bash
# Go 1.25（版本以根目录 go.mod 为准）、Node 22、PostgreSQL 16
brew install go node postgresql@16 lefthook gitleaks
```

### 2. 获取代码并安装 Git 钩子
```bash
git clone git@github.com:brizenchi/go-modules.git
cd go-modules
make hooks          # 安装提交前检查：格式、提交信息、密钥扫描
make test           # 共享包的测试应该全部通过
```

### 3. 启动后端
```bash
cd templates/quickstart
cp .env.example .env        # 按注释填写；本地邮件使用 log，可以不配置第三方服务
createdb quickstart          # 或者使用 .env 里配置的数据库
go run ./cmd/quickstart
curl localhost:8080/health   # 返回 200
```

### 4. 启动前端
```bash
cd templates/quickstart-nextjs
npm ci
npm run dev                  # http://localhost:3000
```

用邮箱验证码登录：`.env.example` 默认设置了 `APP_EMAIL_PROVIDER=log`（不真正发信）和 `APP_AUTH_EMAIL_DEBUG=true`，
验证码会直接显示在登录框里（来自接口返回的 `debug_code`）。生产环境必须关闭 `APP_AUTH_EMAIL_DEBUG`。

## 第一周：读文档

按顺序阅读：

1. [ARCHITECTURE.md](../ARCHITECTURE.md)：分层、用户模型的归属、事件组合；
2. [templates/quickstart/README.md](../../templates/quickstart/README.md)：业务代码放在哪里；
3. [规范总览](./README.md)，重点看：
   - [GIT_WORKFLOW.md](./GIT_WORKFLOW.md)：分支、提交信息、PR；
   - [CODE_STYLE.md](./CODE_STYLE.md)：命名、错误处理、context；
   - [API_STANDARD.md](./API_STANDARD.md)：接口和错误码；
   - [TESTING.md](./TESTING.md)；
4. [OBSERVABILITY.md](../OBSERVABILITY.md)：日志怎么写、怎么在 Grafana 里查问题。

## 第一个 PR

选一个小任务（文档修正、补测试、小缺陷），完整走一遍流程：

1. `git switch -c fix/<scope>-<简述>`；
2. 修改代码并补上测试；
3. `make fmt && make test-race`，修改模板时再运行 `cd templates/quickstart && go test ./...`；
4. 提交：`git commit -m "fix(<scope>): <说明>"`，钩子会检查格式；
5. 推送并创建 PR，按模板填写；
6. 根据审查意见修改，CI 通过后 squash 合并。

## 使用 AI 助手

项目根目录的 `AGENTS.md`（Codex 等工具读取）和 `CLAUDE.md`（Claude Code 读取）包含了这些规范的要点。
AI 助手会自动遵守它们，但生成的代码仍然由你负责审查和测试。详见 [AI_ASSISTANTS.md](./AI_ASSISTANTS.md)。

## 需要权限时

找维护者开通：GitHub 仓库的写权限、Dokploy（只读即可）、Grafana（Viewer）、Stripe 测试模式。
生产环境的密钥不会提供给个人；本地开发使用测试模式的 key。
