# 可观测性标准

所有基于 quickstart 的 SaaS 使用同一套 OpenTelemetry（OTel）约定：链路（traces）和
指标（metrics）通过 OTLP 发往采集器；日志是 stdout 上的 JSON，由平台采集。三者用
`trace_id` 和 `request_id` 关联。

## 组件和第三方包

| 能力 | 位置 | 依赖 |
| --- | --- | --- |
| SDK 初始化、采样、传播、资源属性 | `foundation/tracing.Setup` | `go.opentelemetry.io/otel` SDK、OTLP HTTP/gRPC exporter |
| 入站 HTTP | `foundation/tracing.Middleware` | 官方 `otelgin` |
| 出站 HTTP | `foundation/httpx`（`Tracing`、`Logging`） | 官方 `otelhttp` |
| SQL | `foundation/pgx`（`Tracing`） | `XSAM/otelsql`（OTel registry） |
| Redis | `foundation/rdx`（`Tracing`、`Metrics`） | 官方 `redisotel` |
| 日志 | `foundation/slog` | 标准库 `log/slog` |
| request id、访问日志、panic | `foundation/ginx` | — |

`modules/*` 不依赖 OTel：适配器只接收一个可选的 `HTTPClient`，由模板注入带埋点的客户端。

## 请求链路

```text
CORS → RequestID → tracing.Middleware → AccessLog → Recover → 业务路由
```

| 中间件 | 作用 | 为什么放在这里 |
| --- | --- | --- |
| CORS | 处理预检请求 | 预检不产生 span 和访问日志 |
| RequestID | 读取或生成 `X-Request-ID`，写入 ctx | 必须先于 span 创建，才能写到 server span 上 |
| tracing.Middleware | 提取 `traceparent`/`baggage`，创建 server span，记录 HTTP 指标 | 后面的处理都在这个 span 内 |
| AccessLog | 每个请求记一条日志 | 在 span 内，日志带 `trace_id`/`span_id` |
| Recover | 捕获 panic，记录完整堆栈，把 span 标为 Error，返回 500 | 放在最内层，panic 之后 span、指标和访问日志仍然完整 |

`/health` 不产生 span，也不写访问日志。

## 出站调用

模板在 `internal/platform.NewOutboundHTTPClient` 创建一个共享客户端，交给 Google、
GitHub、Resend、Brevo 和 Stripe 适配器：

- 每次尝试一个 client span，自动注入 W3C `traceparent`；
- 记录 `http.client.request.duration` 指标；
- 每次尝试一条 `component=http_client` 日志，包含 host、path、status、duration；
- 不记录 query string。

Stripe 调用会传递请求的 `ctx`，所以支付调用挂在对应请求的链路下面。新增第三方
适配器时，用 `cfg.HTTPClient` 发请求，并用 `http.NewRequestWithContext(ctx, ...)`
构造请求。

## 日志字段

所有日志都是 JSON，写到 stdout。公共字段：

| 字段 | 来源 |
| --- | --- |
| `time`、`level`、`msg` | slog |
| `service`、`version`、`project`、`env` | 启动配置 |
| `request_id` | `ginx.RequestID` |
| `trace_id`、`span_id`、`trace_flags` | 当前 span |
| `user_id` | auth 中间件 |

访问日志（`msg="http request"`）字段：`component=http`、`operation=request`、
`outcome`、`method`、`path`、`route`、`status_code`、`duration_ms`、`client_ip`、
`user_agent`、`request_size`、`response_size`、`errors`。级别规则：5xx 记 ERROR，
4xx 记 WARN，其余记 INFO。

业务代码统一使用 `slog.InfoContext(ctx, ...)` 这类带 ctx 的方法，关联字段会自动
补全，不需要手动传 `request_id` 或 `trace_id`。

## 敏感数据

- 键名为 `password`、`token`、`authorization`、`cookie`、`api_key`、`client_secret`
  等的日志值，一律输出为 `[REDACTED]`。项目自己的密钥字段加到 `log.redact_keys`。
- SQL 日志和 SQL span 只包含占位符，不包含参数值。只有本地排查问题时才开启
  `db.log_sql_params`。
- Redis span 只记录命令名，不记录参数。
- 访问日志和出站日志都不记录 query string。
- 不合法的 `X-Request-ID`（超过 128 个字符，或包含 `[A-Za-z0-9._:-]` 以外的字符）
  会被替换成新生成的 ID，防止日志注入。

## 采样

采样器是 `ParentBased(TraceIDRatioBased(sample_rate))`：

- 新链路按 `tracing.sample_rate` 采样；
- 上游已经采样的请求一定继续采样，分布式链路不会在中间断开；
- 即使没有配置 endpoint，也会生成 trace_id，日志关联照常可用。

建议开发环境用 `1.0`，生产环境从 `0.1`–`0.2` 起步。

## 配置

```yaml
server:
  name: my-saas
  version: ""            # 通常由 CI 注入 APP_SERVER_VERSION

log:
  level: info
  format: json
  redact_keys: []

tracing:
  endpoint: otel-collector:4318   # 也接受 http:// 或 https:// 前缀
  protocol: http                  # http | grpc
  insecure: true
  sample_rate: 0.2
  authorization: ""               # 例如 OpenObserve 的 "Basic <base64>"
  url_path: ""                    # 例如 OpenObserve 的 /api/default/v1/traces
  metrics:
    enabled: true
    url_path: ""                  # 留空时由 url_path 推导（/v1/traces → /v1/metrics）
    interval_seconds: 60
```

标准环境变量 `OTEL_SERVICE_NAME` 和 `OTEL_RESOURCE_ATTRIBUTES` 可以覆盖资源属性。

## 日志存储

应用只写 stdout，不自己切割文件，也不直接推送日志。日志的收集、保留和查询由
部署平台负责，常见做法：

- Docker / Kubernetes 日志驱动，加 Vector 或 Fluent Bit；
- OpenTelemetry Collector 的 `filelog` receiver，与 traces 和 metrics 进入同一后端。

## 已知边界

- Stripe Webhook 的解析接口（`VerifyAndParseWebhook`）没有 ctx 参数。Webhook 处理中
  补拉 Checkout Session 的那次调用会成为一条独立链路，修改端口签名需要升级主版本。
- `foundation/ossx`（S3/OSS 上传）还没有埋点。
- 日志没有走 OTLP 直接导出，而是由采集器从 stdout 收集；Go 的 OTel Logs SDK 仍是
  beta 版本。
