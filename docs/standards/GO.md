# Go 规范

通用规则见 [CODE_STYLE.md](./CODE_STYLE.md)。

## 工具

| 工具 | 内容 | 在哪里运行 |
| --- | --- | --- |
| `gofmt -s` | 格式 | 本地提交前钩子、CI `go-* / check` |
| `go vet` | 编译器发现不了的错误 | CI `go-* / check` |
| `go test -race` | 测试 + 竞态检测 | CI `go-* / check` |
| `golangci-lint` | errcheck、staticcheck、gosec 等；有 `.golangci.yml` 时使用它 | CI `go-* / lint` |
| `govulncheck` | 依赖漏洞 | CI `go-* / vuln`（不阻止合并） |

## 命名

| 对象 | 规则 | 示例 |
| --- | --- | --- |
| 包 | 小写单词，不用下划线，不用复数 | `httpx`、`billing` |
| 文件 | 小写加下划线，按职责命名 | `access_log.go`、`provider_test.go` |
| 导出标识符 | 驼峰；缩写词全大写 | `UserID`、`HTTPClient` |
| 接口 | 描述行为 | `UserStore`、`Sender` |
| 错误变量 | `Err` 前缀，消息带包名 | `ErrNotFound = errors.New("billing: invoice not found")` |
| 构造函数 | `New` / `NewXxx`，接收 `Config` 结构体 | `NewClient(Config{...})` |

## 错误

- 包装时加上下文：`fmt.Errorf("load invoice %s: %w", id, err)`；
- 用 `errors.Is` / `errors.As` 判断，不比较错误字符串；
- 每个包在一个文件里（比如 `errors.go`）定义哨兵错误；
- HTTP 边界在一个函数里集中把错误映射成响应；`default` 分支记一次 `slog.ErrorContext` 并返回固定文案；
- 业务代码不使用 panic；请求中的 panic 由 recover 中间件兜底。

## context

- `ctx context.Context` 作为第一个参数；不保存到结构体里；
- 一路传递到 `db.WithContext(ctx)`、`http.NewRequestWithContext(ctx, …)`；
- 需要比请求活得更久的后台工作，使用 `context.WithoutCancel(ctx)`，保留 trace 信息。

## 日志

- 使用标准库 `log/slog`，输出 JSON；
- 业务代码使用 `slog.InfoContext(ctx, "fixed message", "key", value)`：消息是固定短语，变化的值放进字段；
- 不打印密钥、token、请求体全文。

## 并发

- 启动 goroutine 时说明它何时结束；用 `errgroup` 或 `sync.WaitGroup` 等待，通过 ctx 取消；
- 共享状态加锁；测试使用 `-race`。

## 测试

- 表驱动测试 + `t.Run`；测试名描述场景：`TestRefund_RejectsDoubleRefund`；
- 修改全局状态（`slog.SetDefault` 等）的测试用 `t.Cleanup` 恢复，不使用 `t.Parallel()`；
- 外部 HTTP 用 `httptest.Server` 模拟；数据库用内存数据库或测试容器；
- 优先使用标准库 `testing`。

## 依赖

- 只做增量修改的公开 API；破坏性变更升主版本（`/v2`）；
- `go mod tidy` 后提交 `go.mod` 和 `go.sum`。
