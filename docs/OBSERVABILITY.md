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

## 日志级别

| 级别 | 什么时候用 | 例子 | 会不会告警 |
| --- | --- | --- | --- |
| `ERROR` | 需要人工处理的问题：未知错误、数据不一致、依赖彻底不可用 | 未知的 500、Webhook 处理失败、panic | 会（通过错误率指标） |
| `WARN` | 不正常但系统已经自动处理，或者是客户端的问题 | 4xx、重试后成功、慢查询、降级 | 不会，定期查看 |
| `INFO` | 重要的业务事件和状态变化 | 用户注册、订阅开通、服务启动、配置加载完成 | 不会 |
| `DEBUG` | 排查问题时临时需要的细节 | 第三方接口的返回摘要 | 生产环境默认不输出 |

规则：

- 一个错误只记一次日志，在处理它的边界记录（见 [CODE_STYLE.md](./standards/CODE_STYLE.md#error-handling)）；
- 客户端的错误（参数错误、未登录）不记 ERROR，否则会掩盖真正的问题；
- 不在循环里、或者每个请求都会走到的代码里记 INFO，访问日志已经记录了每个请求；
- 日志的 `msg` 用固定的英文短语，变化的值放进字段：
  `slog.InfoContext(ctx, "subscription activated", "plan", plan)`，不写 `fmt.Sprintf("user %s activated %s", ...)`。
  这样才能在 Loki 里按消息聚合统计。

### 业务事件日志

重要的业务事件使用统一的字段，方便在 Loki 里统计和审计：

```go
slog.InfoContext(ctx, "subscription activated",
	"component", "billing",      // 所在的模块
	"operation", "activate",     // 动作
	"outcome", "success",        // success / failure
	"plan", plan,
)
```

`user_id`、`request_id`、`trace_id` 会从 ctx 中自动补上，不需要手动传入。

## 自定义指标和 span

HTTP、SQL、外部调用的指标和链路都是自动采集的。只有需要**业务层面**的数据时，才需要自己写：

```go
var logins, _ = otel.Meter("quickstart").Int64Counter("app.user.logins")
logins.Add(ctx, 1, metric.WithAttributes(attribute.String("provider", "google")))

ctx, span := otel.Tracer("quickstart").Start(ctx, "grant signup credits")
defer span.End()
```

- 指标名：`app.<领域>.<名称>`，使用 OTel 的点号写法，导出到 Prometheus 后会变成 `app_user_logins_total`；
- 属性只用取值有限的字段（`provider`、`plan`、`status`）；**禁止**把 `user_id`、`email`、`request_id` 作为指标属性，否则序列数会爆炸，产生费用；
- 业务指标写在宿主的事件订阅里（`internal/bootstrap/subscriptions.go`），不改动共享模块；
- span 名称使用"动作 + 对象"的固定短语，变化的值放进属性。

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

推荐直接使用标准 OTel 环境变量。所有语言和项目通用，`tracing.endpoint` 留空即可：

```dotenv
OTEL_EXPORTER_OTLP_ENDPOINT=https://otlp-gateway-prod-ap-southeast-1.grafana.net/otlp
OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf
OTEL_EXPORTER_OTLP_HEADERS=Authorization=Basic%20<base64(instanceID:token)>
```

在部署面板里填写时**不要给值加引号**：引号会原样传给程序，导致 Authorization 头被丢弃，
后端返回 401。启动日志 `tracing ready` 里的 `endpoint_source` 和 `auth_header` 可以确认
配置是否生效。

也可以写在 YAML 中（`tracing.endpoint` 不为空时优先使用 YAML）：

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

应用只写 stdout，不自己切割文件，也不直接推送日志。每台服务器部署一个 Grafana Alloy，
负责采集这台机器上所有容器的 stdout，推送到该服务器所属账号的 Loki。

```text
容器 stdout ──▶ Alloy（每台服务器一个）──▶ Loki
应用 OTel SDK ─────── OTLP ───────────────▶ Tempo / Prometheus
```

配置文件在 `templates/quickstart/deploy/alloy/`（`config.alloy`、`docker-compose.yml`）。
JSON 日志会被提取为标签 `service`、`project`、`env`、`level`；`trace_id` 和
`request_id` 存为 structured metadata，可以用来过滤，但不会成为高基数标签。非 JSON
日志原样保留，只带 `container` 标签，所以其他语言的服务也能一起采集。

### 部署 Alloy（每台服务器一次）

1. 获取 Loki 凭据：grafana.com → 进入对应的 Stack → Loki 卡片 → **Details**，记下 URL
   和 User；然后在 **Access Policies** 里创建一个带 `logs:write` 权限的 token。
2. 在 Dokploy 里新建一个 **Compose** 服务，代码源指向本仓库，Compose Path 填
   `templates/quickstart/deploy/alloy/docker-compose.yml`，再设置环境变量（值不加引号）：
   ```dotenv
   LOKI_URL=https://logs-prod-<n>.grafana.net/loki/api/v1/push
   LOKI_USERNAME=<Loki User>
   LOKI_PASSWORD=<token>
   SERVER_NAME=<这台服务器的名字>
   ```
   不用 Dokploy 时，在服务器上把这两个文件放进同一个目录，执行 `docker compose up -d`。
3. 验证：Grafana → Explore → `...-logs` 数据源 → `{service="template"}`。按请求 ID 查：
   `{service="template"} | request_id="<X-Request-ID>"`。

Alloy 需要挂载 Docker socket 才能读取容器日志。这是官方的标准做法，但它赋予了 Alloy
访问 Docker API 的权限，所以只使用官方镜像，并固定版本号。

### 托管平台（Railway 等）

托管平台不开放 Docker socket，无法部署 Alloy 采集其他服务的输出。目前的情况：

- 链路和指标：和自建服务器一样，通过 `OTEL_EXPORTER_OTLP_*` 环境变量直接发送，不受影响；
- 日志：只能在平台自带的日志页面查看，或者使用平台的日志转发功能（以平台文档为准）；
- 计划：在 `foundation/slog` 中增加 OTLP 日志导出（stdout 照常保留），实现后托管平台和自建服务器都只需要配置环境变量。

### 链路与日志互相跳转

- **从链路跳到日志**：Connections → Data sources → `...-traces` → **Trace to logs**，
  数据源选 `...-logs`，Tags 填 `service.name` 映射为 `service`，并勾选
  **Filter by trace ID**。
- **从日志跳到链路**：`...-logs` → **Derived fields**，新增一个字段：名称 `TraceID`，
  类型 **Label**，标签名 `trace_id`，内部链接选择 `...-traces`，查询填 `${__value.raw}`。

## 告警

规则文件在 `templates/quickstart/deploy/alerts/rules.yaml`，测试文件是
`rules_test.yaml`（运行 `promtool test rules rules_test.yaml`）。

| 告警 | 条件 | 级别 |
| --- | --- | --- |
| ServiceTelemetryMissing | 10 分钟没有收到遥测数据（服务宕机或导出失败） | critical |
| HighServerErrorRate | 5xx 比例超过 1%，持续 5 分钟；流量低于每分钟 3 次时不触发 | critical |
| HighLatencyP99 | P99 超过 2 秒，持续 10 分钟 | warning |
| StripeWebhookFailing | 10 分钟内出现任何一次 Webhook 非 2xx 响应 | critical |
| OutboundDependencyErrors | 某个第三方域名的 5xx 或网络错误超过 10%，持续 10 分钟 | warning |

导入步骤：

1. 确认 job 标签：Explore → `...-prom` → `target_info`。OTLP 数据会被转换成
   `job="<service.namespace>/<service.name>"`，例如 `template/template`。把规则文件里的
   `template/template` 替换为实际的值。
2. Alerting → Contact points：添加通知方式（邮件、Telegram、Slack 等），并把它设置为
   默认 Notification policy 的接收方。
3. Alerting → Alert rules → **Import**，导入 `rules.yaml`。也可以用
   `mimirtool rules load rules.yaml`。
4. 验证：在 Alert rules 页面，各条规则的状态应为 Normal。

不是基于本模板的服务（指标名不同）至少要保留 `ServiceTelemetryMissing`，其余规则需要
按该服务实际的指标名调整。

### 告警级别与 SLO

| 级别 | 通知方式 | 对应的故障级别 |
| --- | --- | --- |
| `critical` | 立即通知（手机推送、Telegram） | P1 / P2，见 [INCIDENT_RESPONSE.md](./standards/INCIDENT_RESPONSE.md) |
| `warning` | 工作时间查看（邮件、频道消息） | P3 |

- 每条告警都要能回答"收到之后该做什么"，写在 `annotations.description` 里；没人处理的告警应该删除或调整阈值；
- 正式对客户承诺可用性之后，在 Grafana 的 SLO 功能里定义目标（比如"月可用性 99.9%"、"99% 的请求在 1 秒内完成"），按错误预算的消耗速度告警，取代固定阈值。

## 已知边界

- Stripe Webhook 的解析接口（`VerifyAndParseWebhook`）没有 ctx 参数。Webhook 处理中
  补拉 Checkout Session 的那次调用会成为一条独立链路，修改端口签名需要升级主版本。
- `foundation/ossx`（S3/OSS 上传）还没有埋点。
- 日志没有走 OTLP 直接导出，而是由采集器从 stdout 收集；Go 的 OTel Logs SDK 仍是
  beta 版本。
