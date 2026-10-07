# 测试规范

## 测什么、在哪一层测

| 类型 | 范围 | 工具 | 位置 |
| --- | --- | --- | --- |
| 单元测试 | 纯逻辑：领域规则、计算、校验、映射 | `testing`，表驱动 | 和被测代码放在同一个包，`*_test.go` |
| 仓储测试 | SQL 和 GORM 的行为 | SQLite 内存数据库 | `adapter/gormrepo`、`internal/feature/*/repository` |
| HTTP 测试 | 路由、中间件、参数校验、响应外壳、状态码 | `httptest` + gin | `http/*_test.go`、`internal/http/middleware` |
| 适配器测试 | 第三方 API 的调用和解析 | `httptest.Server` 模拟第三方 | `adapter/stripe`、`adapter/resend` 等 |
| 组合测试 | 模板的组装、配置校验、事件订阅 | 真实组装 + SQLite | `templates/quickstart/internal/bootstrap` 等 |
| 前端测试 | 工具函数、状态逻辑、请求封装 | `npm test` | `templates/quickstart-nextjs/tests` |
| 上线验收 | 登录、支付、邮件、可观测性的完整流程 | 人工按清单执行 | [DEPLOYMENT.md](./DEPLOYMENT.md#上线验收清单) |

尽量把逻辑写在不依赖 HTTP 和数据库的地方，用单元测试覆盖；HTTP 测试只覆盖接口契约。

## 必须写测试的情况

- 新增的公开函数、接口、领域规则；
- **修复缺陷时，先写一个能复现问题的测试**，再修复；
- 涉及金额、积分、权限、幂等的逻辑：正常情况、边界值、重复请求、无权访问都要覆盖；
- 新增配置项的校验规则。

## 写法

- 测试名称说明场景和期望：`TestRecover_MarksSpanAsError`、`TestLogger_HidesBindValuesByDefault`；
- 多组输入使用表驱动测试 + `t.Run`；
- 测试之间相互独立：不依赖执行顺序，用 `t.Cleanup` 恢复全局状态（比如 `slog.SetDefault`、`otel.SetTracerProvider`、`stripe.SetBackend`）；
- 修改了全局状态的测试不使用 `t.Parallel()`；
- 断言失败时打印实际值：`t.Fatalf("status = %d, want 500", w.Code)`；
- 优先使用标准库 `testing`；不引入断言库，除非整个包都在用。

## 外部依赖

- **测试不访问外部网络**：第三方 API 用 `httptest.Server` 模拟；不调用真实的 Stripe、Resend、Grafana；
- 数据库用 SQLite 内存数据库；必须依赖 PostgreSQL 特性的测试，用 build tag 或环境变量开关，默认跳过；
- 时间、随机数需要确定时，通过参数或接口注入，不在测试里 `time.Sleep` 等待。

## 测试数据

- 假密钥必须**一眼就能看出是假的**：`sk_test_not-a-real-key`、`whsec_not-a-real-secret`、`re_not-a-real-key`。
  格式逼真的假密钥会被 GitHub 推送保护和 gitleaks 拦截；
- 邮箱使用 `example.com`、`example.test` 等保留域名；
- 不使用真实用户数据或生产数据。

## 运行

```bash
make test           # foundation + modules
make test-race      # 加上 race 检测（CI 使用）
cd templates/quickstart && go test ./...
cd templates/quickstart-nextjs && npm test
```

## 覆盖率

- `foundation/*` 和 `modules/*`：每个包的覆盖率不低于 **70%**，PR 不能降低覆盖率；
- 模板：覆盖组装、配置校验、关键业务路径，不设硬性的百分比；
- 覆盖率只是参考，重点是关键分支有没有测到。

## 可观测性相关的测试

需要检查日志、链路时的写法（参考现有的测试）：

- 日志：`flog.Setup(flog.Config{Format: flog.FormatJSON, Output: &buf})`，然后检查 `buf` 里的 JSON；
- 链路：使用 `tracetest.NewSpanRecorder()` 创建 TracerProvider，检查 span 的名称、属性和状态；
- 告警规则：修改 `deploy/alerts/rules.yaml` 后运行 `promtool test rules rules_test.yaml`。
