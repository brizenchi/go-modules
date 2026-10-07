# API 设计规范

适用于所有 HTTP 接口，包括 `modules/*/http` 和 `templates/quickstart/internal/feature/*`。

## 路径

| 规则 | 示例 |
| --- | --- |
| 统一前缀和版本：`/api/v1` | `/api/v1/account/profile` |
| 资源用名词；多个单词用 `-` 连接；集合用复数 | `/api/v1/notes`、`/api/v1/credits/transactions` |
| 嵌套最多两层 | `/api/v1/notes/:id/export` |
| 动作无法用 HTTP 方法表达时，用动词子路径 | `/api/v1/stripe/subscription/cancel`、`/api/v1/auth/send-code` |
| 后台管理接口统一放在 `/api/v1/admin/*` | `/api/v1/admin/users` |
| 第三方回调使用固定路径，不随功能调整而变化 | `/api/v1/stripe/webhook`、`/api/v1/auth/:provider/callback` |

路由按权限分组挂载：`Public`、`User`（需要登录）、`Admin`（需要管理员）。见 `internal/hostapi.Groups`。

## HTTP 方法

| 方法 | 用途 | 幂等 |
| --- | --- | --- |
| `GET` | 查询，不能有副作用 | 是 |
| `POST` | 创建资源，或执行动作 | 否，需要时使用 `Idempotency-Key` |
| `PATCH` | 部分更新 | 是 |
| `PUT` | 整体替换（少用） | 是 |
| `DELETE` | 删除 | 是 |

## 请求头

| 请求头 | 说明 |
| --- | --- |
| `Authorization: Bearer <token>` | 用户身份 |
| `Content-Type: application/json` | 请求体统一使用 JSON；文件上传除外 |
| `X-Request-ID` | 可选。客户端可以自己生成并传入；没有传时由服务端生成。响应头里一定会返回 |
| `Idempotency-Key` | 有副作用、可能被重复提交的写操作要求传入，最长 128 个字符 |
| `traceparent` | 可选，W3C 链路上下文。服务之间调用时会自动带上 |

## 响应外壳

所有 JSON 响应都使用 `foundation/httpresp`：

```json
{ "code": 200, "msg": "ok", "data": { ... } }
```

- 成功：`httpresp.OK(c, data)`；
- 失败：`httpresp.BadRequest`、`Unauthorized`、`Forbidden`、`NotFound`、`Conflict`、`TooManyRequests`、`InternalError`；
- 外壳里的 `code` 和 HTTP 状态码保持一致（`OKWith` 的"软错误"除外，新代码不建议使用）；
- 只有 Webhook、文件下载、重定向这类必须遵守第三方协议的接口，才可以不使用外壳。

## 错误码

| HTTP / code | 何时使用 | httpresp |
| --- | --- | --- |
| 400 | 参数不合法、格式错误、业务规则不允许（比如验证码错误） | `BadRequest` |
| 401 | 没有登录，或者 token 无效、已过期 | `Unauthorized` |
| 403 | 已经登录，但没有权限（不是本人的资源、不是管理员） | `Forbidden` |
| 404 | 资源不存在。**查询别人的资源时也返回 404**，不暴露资源是否存在 | `NotFound` |
| 409 | 状态冲突：重复创建、版本冲突、幂等键被不同的请求体使用 | `Conflict` |
| 429 | 超出频率限制 | `TooManyRequests` |
| 500 | 未知错误。消息使用固定文案，不返回内部细节 | `InternalError` |
| 503 | 功能没有启用，或者依赖的服务暂时不可用 | `Custom(c, 503, 503, msg, nil)` |

### 前端如何区分具体的错误：`reason`

只看状态码，前端没法区分"验证码错误"和"邮箱格式错误"。**新增的接口**如果需要前端根据错误做不同的处理，
在 `data` 里返回一个稳定的 `reason`：

```go
httpresp.Custom(c, http.StatusBadRequest, http.StatusBadRequest, "auth: invalid or expired verification code",
	gin.H{"reason": "AUTH_INVALID_CODE"})
```

```json
{ "code": 400, "msg": "auth: invalid or expired verification code", "data": { "reason": "AUTH_INVALID_CODE" } }
```

- `reason` 格式：`<模块>_<原因>`，全大写，用下划线连接；一旦发布就不能修改；
- `msg` 给开发人员看，内容是英文，前端**不要**直接展示给用户。用户看到的文案由前端根据 `reason` 和多语言配置决定；
- 现有接口目前只返回 `msg`，以后修改到这些接口时再逐步补上 `reason`，保证向后兼容。

### 错误信息的安全要求
- 500 错误不返回 `err.Error()`；
- 登录失败不区分"用户不存在"和"密码错误"，统一返回同一个错误；
- 内部错误的细节只写进日志，用户可以通过响应头里的 `X-Request-ID` 反馈问题。

## 数据格式

| 类型 | 规则 | 示例 |
| --- | --- | --- |
| 字段名 | 小写加下划线 | `created_at`、`current_period_end` |
| 时间 | RFC 3339，统一使用 **UTC** | `"2026-10-06T03:19:30Z"` |
| 金额 | **整数，单位是最小货币单位**（分），同时返回币种 | `{"amount_due_now": 1999, "currency": "usd"}` |
| ID | 字符串（UUID） | `"8f2c..."` |
| 枚举 | 小写加下划线的字符串 | `"period_end"`、`"immediate_prorated"` |
| 布尔 | `true` / `false`，不使用 0 和 1 | |
| 空值 | 可选字段用 `null` 或者省略（`omitempty`），同一个字段的处理方式要保持一致 | |

## 分页

使用页码分页（和现有接口一致）：

```text
GET /api/v1/credits/transactions?page=1&limit=20
```

```json
{ "code": 200, "msg": "ok", "data": { "items": [...], "total": 135, "page": 1, "limit": 20 } }
```

- `page` 从 1 开始；`limit` 默认 20，**最大 100**，超出时按 100 处理或返回 400；
- 必须有稳定的排序（比如 `created_at DESC, id DESC`），避免翻页时数据重复或遗漏；
- 数据量很大或者需要实时滚动的列表，可以改用游标分页（`cursor` / `next_cursor`），在接口文档里注明。

## 筛选和排序

- 筛选直接使用字段名：`?status=active&plan=pro`；
- 排序使用 `sort=-created_at`（`-` 表示倒序），只开放有索引的字段；
- 搜索关键词使用 `q`，限制最大长度（目前是 200 个字符）。

## 幂等

- 创建订单、发放积分、修改服务商配置这类**有副作用、重试会造成损失**的接口，要求传入 `Idempotency-Key`；
- 服务端保存"key → 请求摘要 → 结果"：相同的 key 和请求体返回同一个结果；相同的 key 但请求体不同，返回 409；
- Stripe 等 Webhook 依靠事件 ID 去重，见 `billing_*` 表。

## 认证、限流、CORS

- 用户接口挂在 `User` 分组下，由 auth 中间件校验 token；资源归属必须在服务层再检查一次（防止越权，见 [SECURITY_STANDARD.md](./SECURITY_STANDARD.md#鉴权与越权)）；
- 登录、发送验证码等容易被滥用的接口要有频率限制，超出时返回 429；
- CORS 白名单通过 `http.allowed_origins` 配置，不使用 `*`。

## 兼容性与废弃

- 在 `/api/v1` 下，**只做增量修改**：可以新增字段、新增接口、新增可选参数；
- 不能删除或重命名字段，不能改变字段类型和含义，不能把可选参数改成必填；
- 确实需要破坏性变更时，新增 `/api/v2/...` 接口，旧接口保留至少一个发布周期，并在响应头里加上 `Deprecation: true`；
- 前端和后端在同一个 PR 里修改时，也要保证"先部署后端、后部署前端"的过程中不会出错。

## 接口契约

现阶段：

- 后端的请求和响应结构体定义在 handler 所在的包中；
- 前端的类型统一放在 `templates/quickstart-nextjs/lib/api.ts`；
- **修改接口时，前端类型要在同一个 PR 里同步修改**，审查时会检查这一点。

后续计划：引入 OpenAPI 文档，由后端代码生成规范文件，前端根据它自动生成类型，取代手动同步。引入之后，以生成的规范文件为准。
