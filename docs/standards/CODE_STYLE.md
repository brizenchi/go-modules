# 代码规范（Go）

前端规范见 [FRONTEND.md](./FRONTEND.md)。

## 自动化工具

| 工具 | 检查内容 | 运行方式 |
| --- | --- | --- |
| `gofmt -s` | 代码格式 | `make fmt`；提交前钩子和 CI 都会检查 |
| `golangci-lint` | errcheck、govet、staticcheck、gosec 等，配置见 `.golangci.yml` | `make lint`；CI 检查 |
| `make purity-check` | `foundation`、`modules` 不导入宿主代码 | CI 检查 |
| `.editorconfig` | 缩进、换行符、文件末尾换行 | 编辑器自动生效 |

工具能检查的，审查时就不再讨论。

## 基本原则

- 先读周围的代码，**保持一致**：命名、注释密度、错误处理方式都和同一个包里的现有代码一样；
- 优先使用标准库；引入新的第三方依赖需要在 PR 里说明理由（见 [CI_QUALITY.md](./CI_QUALITY.md#依赖管理)）；
- 不写"以后可能用到"的抽象。接口只在有两个以上实现，或者需要隔离外部依赖（端口）时才定义。

## 命名

### Go 代码
| 对象 | 规则 | 示例 |
| --- | --- | --- |
| 包 | 小写单词，不用下划线和复数 | `httpx`、`emailcode` |
| 文件 | 小写加下划线，按职责命名 | `access_log.go`、`provider_test.go` |
| 导出标识符 | 驼峰；缩写词全大写 | `UserID`、`HTTPClient`、`ParseURL` |
| 接口 | 描述行为 | `UserStore`、`Sender` |
| 错误变量 | `Err` 前缀，消息带包名前缀 | `ErrInvalidCode = errors.New("auth: invalid or expired verification code")` |
| 构造函数 | `New` / `NewXxx`，接收 `Config` 结构体 | `pgx.Open(pgx.Config{...})` |
| 布尔值 | `Is`、`Has`、`Enabled` 等形式 | `AuthEnabled()` |

### 其他命名
| 对象 | 规则 | 规范文档 |
| --- | --- | --- |
| 数据库表和字段 | 小写加下划线；表名用复数；模块的表加模块前缀 | [DATABASE.md](./DATABASE.md#命名) |
| JSON 字段 | 小写加下划线 | [API_STANDARD.md](./API_STANDARD.md#数据格式) |
| URL 路径 | 小写，多个单词用 `-` 连接 | [API_STANDARD.md](./API_STANDARD.md#路径) |
| 环境变量 | `APP_` 前缀，层级之间用 `_` | [CONFIG_STANDARD.md](../CONFIG_STANDARD.md) |
| 日志字段 | 小写加下划线，使用固定的字段名 | [OBSERVABILITY.md](../OBSERVABILITY.md#日志字段) |
| 自定义指标 | `app.<领域>.<名称>`，用 OTel 的点号写法 | [OBSERVABILITY.md](../OBSERVABILITY.md#自定义指标和-span) |

## 错误处理

### 原则：每个错误只处理一次

一个错误要么**返回**给上层，要么**在这一层处理掉**（记日志、降级、转换成响应），不要两件事都做。
不然同一个错误会在日志里出现好几次。

```go
// ❌ 记了日志又返回，上层还会再记一次
if err != nil {
	slog.ErrorContext(ctx, "load user failed", "error", err)
	return err
}

// ✅ 包装上下文后返回，在边界统一处理
if err != nil {
	return fmt.Errorf("load user %s: %w", userID, err)
}
```

### 各层的职责
| 层 | 做什么 |
| --- | --- |
| repository、adapter | 把底层错误包装后返回（`%w`）；把"找不到"转换成领域错误，比如 `domain.ErrUserNotFound` |
| app、service | 用 `errors.Is` 判断领域错误并处理业务分支；其他错误包装后返回 |
| http handler（**边界**） | 用一个 `respondAppError` 函数把领域错误映射成 HTTP 响应（见下文）；**只对未知错误**记 ERROR 日志 |
| 后台任务、事件监听器（**边界**） | 处理不了的错误记一次日志，并决定是否重试 |

### 领域错误
- 每个模块在 `domain/errors.go` 里定义哨兵错误：`var ErrXxx = errors.New("<模块>: <描述>")`；
- 调用方用 `errors.Is` 判断，不比较字符串；
- 需要携带数据时，定义一个实现了 `error` 接口的结构体，用 `errors.As` 取出来。

### HTTP 边界的映射
参照 `modules/auth/http/handler.go` 中的 `respondAppError`：

```go
func respondAppError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidCode):
		httpresp.BadRequest(c, err.Error())          // 已知错误：返回明确的提示，不记 ERROR
	case errors.Is(err, domain.ErrUserNotFound):
		httpresp.NotFound(c, err.Error())
	default:
		slog.ErrorContext(c.Request.Context(), "auth: internal error", "error", err) // 未知错误：记一次日志
		httpresp.InternalError(c, "internal authentication error")                // 不把内部细节返回给客户端
	}
}
```

- 未知错误**不能**把 `err.Error()` 返回给客户端，里面可能有 SQL、内部地址或第三方返回的内容；
- 状态码和错误码的选择见 [API_STANDARD.md](./API_STANDARD.md#错误码)。

### panic
- 业务代码里不使用 panic 表达错误；
- 只在程序启动时遇到无法继续的配置错误，或者出现"不可能发生"的状态时才使用；
- 请求处理过程中的 panic 由 `ginx.Recover` 兜底：返回 500，并记下完整堆栈。

## context

- 第一个参数是 `ctx context.Context`；不把 ctx 存进结构体；
- 从 handler 一路传到 repository 和外部调用：`db.WithContext(ctx)`、`http.NewRequestWithContext(ctx, ...)`；
- 日志使用 `slog.InfoContext(ctx, ...)`，这样 `request_id`、`trace_id` 会自动带上；
- 不使用 `context.Background()` 替代调用方传进来的 ctx。只有在启动程序、或者需要脱离请求生命周期的后台任务里才可以使用。

## 并发

- 启动 goroutine 时必须说明它什么时候结束；使用 `errgroup` 或 `sync.WaitGroup` 等待，并通过 ctx 取消；
- 在 goroutine 里继续处理请求相关的工作时，要传递 ctx；如果需要比请求活得更久，用 `context.WithoutCancel(ctx)`，保留 trace 信息；
- 共享状态要加锁，或者通过 channel 传递；`make test-race` 必须通过。

## 注释

- 导出的标识符都要有文档注释，以标识符的名字开头；
- 注释写**为什么**这样做，而不是复述代码做了什么；
- `TODO` 要写明负责人或 issue 编号：`// TODO(brizen): ... #123`。

## 配置与常量

- 配置通过 `Config` 结构体注入，库代码里不直接读取环境变量（`foundation/config` 和启动代码除外）；
- 不写魔法数字：超时、重试次数、额度上限都定义为常量，或者放进配置。
