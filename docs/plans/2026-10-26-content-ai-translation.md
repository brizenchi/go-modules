# 内容管理 AI 中英翻译开发计划

> 执行说明：使用 `executing-plans` 技能按任务实施。任务使用复选框跟踪。本次文档不代表业务代码已实现。

**目标：** 管理员确认原文，选择目标语言并点击“翻译并检查”，最后自行决定是否发布。

**架构：** 在 quickstart 内容功能内增加翻译任务和 Gemini 原生协议适配器。复用现有内容草稿、发布快照、审计和预览。任务保存到数据库，由应用内工作器处理，页面查询进度；不引入外部队列。

**技术栈：** Go、GORM、Gin、标准库 HTTP/JSON、Next.js、严格 TypeScript、现有 Markdown 渲染器。

**状态：** 产品范围已在讨论中确认；本文的配置值、接口和后台任务设计是实施方案。日期：2026-10-26。

## 1. 已确认的产品范围

1. 第一版只支持中文 `zh` 与英文 `en`，支持双向翻译。
2. 后续目标语言是日语、韩语、西语、法语；本次不开放这些语言，也不改它们的公开页面路由。
3. 每篇文章明确记录一个原文语言。所有译文直接来自原文，不能经过另一种译文转译。
4. 管理员选择目标语言，点击一个“翻译并检查”按钮。没有独立的检查按钮，也没有人工审核流程。
5. 翻译标题、摘要和正文；已有 SEO 标题和描述一并翻译。原文为空的可选 SEO 字段保持为空。
6. AI 生成译文后自动检查。存在问题时，自动修改一次，再检查一次。最多四次模型调用，不无限循环。
7. 第二次检查仍有质量问题时，保存完整译文并显示中文警告。管理员仍可手动发布。
8. 重新翻译成功后直接替换目标语言草稿。无需额外的替换确认。
9. 生成失败、结果截断或技术性检查失败时，保留原草稿。
10. 任何翻译操作都不修改线上版本。只有现有“发布当前语言”操作更新相应发布快照。
11. 复用现有预览。第一版不做原文与译文并排审核、批量翻译、术语管理后台或多模型投票。
12. 长度上限放在服务端本地配置。默认中文 5,000 个字符、英文 3,000 个词；这是产品初始限制，不是平台能力承诺。

## 2. 项目约束

- 遵守根目录和两个模板目录的 `AGENTS.md`。
- 实施前阅读 `docs/standards/PROJECT.md`、`CODE_STYLE.md`、`API_STANDARD.md`、`DATABASE.md`、`SECURITY_STANDARD.md`、`TESTING.md`、`GO.md`、`NODE.md`，以及 `docs/CONFIG_STANDARD.md`。
- 新端点遵守 `.claude/skills/add-endpoint/SKILL.md`；接入平台遵守 `.claude/skills/add-third-party/SKILL.md`。
- 业务代码位于 `templates/quickstart/internal/feature/content/`。组合与配置位于模板中，不能让 foundation 依赖业务模块。
- 不新增嵌套 Go module。优先使用现有依赖和标准库。
- 不改变 `foundation/aix` 原有 OpenAI 兼容接口及其输出上限。Gemini 原生接口单独适配，避免影响其他 AI 功能。
- 出站 HTTP 使用注入的 `platform.Config.HTTPClient`。所有调用传递 context；数据库操作使用 `WithContext`。
- 不在数据库事务内调用 AI。HTTP 和数据库写入分阶段执行。
- 新模型与字段通过 GORM AutoMigrate 增加，不编辑已有 migration。
- 接口改动同时更新 `templates/quickstart-nextjs/lib/api.ts` 和调用封装。
- HTTP 错误使用 `foundation/httpresp`；稳定 `reason` 位于 `data`。集中映射内容功能错误，不重复记录错误。
- 日志使用固定消息和结构化字段。不记录 Key、文章正文、提示词、响应正文或查询字符串；不手动添加 request_id/trace_id。
- 自动化测试不访问真实平台；使用注入的假 HTTP transport 或项目允许的本地假服务。
- 本文不要求提交、推送或部署。生产部署只走现有 CI。

## 3. 现有能力与缺口

| 位置 | 已有能力 | 本次变化 |
| --- | --- | --- |
| `templates/quickstart/internal/feature/content/content.go` | 双语草稿、发布快照、版本控制 | 增加原文语言和翻译检查元数据 |
| `templates/quickstart/internal/feature/operations/content.go` | 管理内容 API、公开内容读取 | 新增翻译任务入口及集中错误映射 |
| `templates/quickstart-nextjs/components/content-admin.tsx` | 双语编辑、保存、发布、预览 | 增加原文选择、翻译按钮和状态提示 |
| `templates/quickstart-nextjs/lib/content-api.ts` | 内容 API 封装 | 新增提交及查询任务方法 |
| `templates/quickstart/internal/hostcfg/hostcfg.go` | 本地业务配置 | 独立的内容翻译配置 |
| `foundation/aix/` | OpenAI 兼容调用，输出最多 4,096 tokens | 保持原有行为，不用于本次 Gemini 调用 |

现有后端写响应超时和出站客户端默认超时约 30 秒。不能把最多四次 AI 请求直接串在一次管理 API 请求中。后台任务是技术实现，不增加管理员操作。

## 4. 管理员操作

1. 新文章选择原文语言。默认选择当前编辑语言，必须显示出来，不能在后台静默推断。
2. 已有文章没有原文语言时，提示管理员选择。既有保存、发布功能继续可用；只有 AI 翻译要求先确认原文。
3. 原文语言和编辑页签是不同概念。切换页签不会改变原文语言。
4. 原文有未保存修改时，点击“翻译并检查”先调用现有保存流程；保存失败则不创建任务。
5. 管理员选择另一种语言作为目标；源语言与目标语言相同时不能提交。
6. 页面显示“等待处理”“正在翻译”“正在检查”“正在修正”“正在复查”。运行中禁止对同一文章重复发起翻译。
7. 任务结束后刷新文章及版本。完整结果直接写入目标草稿，现有预览使用该草稿。
8. 质量警告不禁用发布按钮，也不增加强制审批弹窗。保留原有字段完整性、版本和权限验证。
9. 原文后续变化时，译文显示“原文有更新”。不自动重译、不自动下架、不阻止管理员发布。
10. 切换原文语言需要保存，并令已有检查状态失效；不自动替换任何语言内容。

审计继续记录 actor、操作和版本。翻译操作原因可由系统生成“AI 翻译：zh → en”，不增加必填输入。原有保存、发布的操作原因规则保持现状。

## 5. 翻译与检查规则

### 5.1 模型输入输出

- 每次请求明确源语言、目标语言和本次任务，不携带无关聊天历史。
- 原文和译文作为数据传入。提示词明确忽略文章内部的指令，不启用搜索、工具调用或网址抓取。
- 生成要求忠于原文，不总结、不增加事实、不改产品名、代码、变量或链接目的地址。
- 返回对象固定为 `title`、`summary`、`body`、`seo_title`、`seo_description`，均为字符串。
- 检查请求独立发送，输入完整原文和当前译文。返回 `passed` 布尔值及 `issues` 数组。
- 每个 issue 包含 `code`、`field`、`message_zh`。code 限于 `omission`、`meaning`、`addition`、`terminology`、`format`，field 限于上述五个字段。
- AI 中文说明作为不可信的检查数据展示，只按纯文本渲染；不得当成 API 错误消息或指令。限制最多 20 条、每条 300 个 Unicode 字符。
- `passed=true` 必须对应空 issues；不符合结构的返回属于技术失败。

### 5.2 有限修正

```text
生成 → 程序校验 → AI 检查
                    ├─ 通过：保存草稿，状态 passed
                    └─ 有质量问题：修正一次 → 程序校验 → AI 复查
                                                       ├─ 通过：保存草稿，状态 passed
                                                       └─ 仍有问题：保存草稿，状态 warning
任一步技术失败：任务 failed，保留已有草稿和线上版本
```

第二次检查未通过不等于调用失败。前者允许保存和发布，后者不写入候选结果。检查服务中断时也保留旧草稿，不能把“未完成检查”显示为“检查通过”。

### 5.3 完整性与质量的边界

- 技术失败：非成功 HTTP 状态、超时、空响应、JSON 解析失败、必填字段缺失、字段超长、输出达到 token 限制、平台拦截、未完成的 continuation。
- `finishReason=STOP` 是接受响应的必要条件之一，但不是语义完整性的证明。
- 程序检查 Markdown 代码块、行内代码、链接目的地址及明确占位符是否保留；异常加入质量问题，触发同一轮有限修正。
- 数字和单位差异加入检查问题，不用简单字符串相等判断数值错误，避免误判语言格式差异。
- 质量问题允许管理员覆盖判断。无法解析或残缺的技术结果不覆盖草稿。
- 最终 warning 应合并程序检查和 AI 检查的问题；AI 返回 passed 不能抹掉程序仍发现的问题。
- 人工编辑译文后，之前的检查结果标记为失效，不能继续显示“当前译文已检查”。不新增检查按钮。

## 6. 配置与平台验证

新增 `host.content_translation`，不复用 `host.ai` 的协议假设。非敏感配置放在本地 `deploy/config.yaml`，示例写入 `templates/quickstart/deploy/config.yaml.example`。

```yaml
host:
  content_translation:
    enabled: false
    base_url: "https://api.aigocode.dev"
    model: "gemini-3.8-flash"
    max_output_tokens: 16384
    max_zh_characters: 5000
    max_en_words: 3000
    request_timeout_seconds: 90
    task_timeout_seconds: 420
```

- Key 仅使用 `APP_HOST_CONTENT_TRANSLATION_API_KEY` 或未提交的本地 `.env`。不写入 YAML 示例、Git、浏览器或日志。
- 上述 16,384 token、90 秒和 420 秒是工程起始配置，不是平台已验证的能力。
- base_url 必须是固定配置的 HTTPS 根地址，无凭据、查询参数或 fragment；禁止客户端覆盖地址。
- 发送到 `/v1beta/models/{model}:generateContent`，通过 `x-goog-api-key` 请求头认证，不放在 URL。
- 拒绝跨地址重定向，避免把自定义认证头发送到其他主机；更换地址由配置明确完成。
- 适配器基于注入 HTTPClient 的副本设置自己的 Timeout，复用其 Transport 与观测能力，不修改全局客户端。
- 启用时验证 Key、模型、地址及所有正数限制；单次超时不得大于任务总超时。禁用时无需 Key。
- 配置修改后重启后端。第一版不做管理后台配置和热更新。

长度计算：标题、摘要、正文及非空 SEO 字段合并计算。中文按非空白 Unicode 字符计数（含标点和 Markdown）；英文按空白分隔的非空 token 计词。前后端使用相同样例验证，后端为准。另保留现有字段及请求体字节限制，防止无空格长文本绕过英文词数限制。长度只限制 AI 翻译，不缩小既有编辑和保存能力。

不依赖中转平台尚未确认的 countTokens 接口。第一版依靠可调的产品长度限制、输出预算、结束原因和内容校验；若实测支持计数接口，可再加入输入 token 预检。不能用官方输入上限推断中转站必然接受同样长度。

### 实施前接口验证

- [x] 从用户本地环境读取 Key，不要求把 Key 发到聊天中。
- [x] 对用户账户确认 `.app` 与文档 `.dev` 中实际可用的端点、Gemini 分组权限及业务 API 使用范围。
- [x] 使用合成短文验证模型名称、认证、结构化返回及 finishReason。
- [x] 使用接近产品上限的合成中英文文章测试两方向，检查末段、列表、链接、代码及返回字段。
- [ ] 检查至少一次“生成 + 检查”全流程的耗时和平台用量记录。
- [x] 记录实测结论到本计划附录；不能把未实测的限制描述为已支持。真实调用是手动接入验证，不进入测试套件和 CI。

## 7. 数据与任务一致性

### 内容元数据

在 `content_articles` 追加可兼容字段：`source_locale`（空值表示尚未确认）和 `translation_meta_json`。通过 Article API 增加同名概念的 `source_locale` 与 `translation_meta` 字段，不修改现有 translations/published 结构。

每个目标语言的元数据记录：源语言、源内容 hash、目标草稿 hash、检查状态、中文问题、模型名称、完成时间和任务 ID。hash 使用固定字段顺序的规范 JSON 及 SHA-256。时间使用 RFC 3339 UTC。

旧客户端未提交 source_locale 时保留已有值；新建旧格式文章仍可保存。旧数据不自动猜测源语言。翻译元数据只能由服务端写入。

### 持久任务

新增 GORM 模型 `TranslationJob`，表名 `content_translation_jobs`。保存 article_id、actor_id、source_locale、target_locale、输入文章 version、源内容快照及 hash、幂等键与请求摘要、状态、阶段、错误 reason、结果元数据、created_at/updated_at、deadline、租约标识。

- 状态：`queued`、`running`、`succeeded`、`failed`。质量 warning 属于 succeeded 的检查结果。
- 任务创建与审计在短事务内提交。相同 actor、action、幂等键有唯一约束；相同请求返回同一个任务，不同请求摘要返回 409。
- 每篇文章最多一个活动任务，用数据库条件更新/活动任务标识实现，不能仅靠进程内锁。
- 工作器从数据库领取任务。领取使用条件更新和租约，支持多个应用实例争抢时只有一个执行者。
- 工作器每次只执行一个任务；不新增用户可见的批量队列管理。
- 用应用生命周期启动和停止工作器。工作器 context 带任务 deadline 和追踪关联，不存储在结构体中，不使用已结束 HTTP 请求的取消信号。
- 应用重启后可继续领取 queued 任务。已运行且租约到期的任务标记失败，不自动重放收费的上游调用。
- 租约覆盖总任务期限并留回收余量。最终写入同时校验租约标识、deadline、活动任务 ID 和输入文章 version，拒绝失效工作器提交。
- 模型 HTTP 调用在事务外执行。最终短事务原子替换目标草稿、更新元数据、增加文章 version、结束任务并写审计。
- 文章在任务运行期间发生任何写入时，最终提交返回版本冲突，丢弃候选结果。不要覆盖另一个管理员的修改。
- 同一目标语言重复提交的新任务必须使用新幂等键。网络重试复用旧键，仅查询/返回原任务。
- 无法承诺上游收费 exactly-once；进程中断可能已消耗平台额度，但不会自动重试导致更多消费。
- 任务完成后清除临时原文快照和候选文本，仅保留结果元数据；幂等记录按现有审计保留策略保留。

## 8. API 合约

所有端点仅管理员可用。文章是站点级内容，沿用现有管理员授权；任务访问仍验证管理员身份和关联文章存在，不提供公开访问。

### 创建任务

`POST /api/v1/admin/articles/:id/translations`，要求 `Idempotency-Key`。

```json
{"version": 12, "source_locale": "zh", "target_locale": "en", "reason": "AI 翻译：zh → en"}
```

服务端从已保存文章读取原文，要求 source_locale 与保存值一致。客户端不上传任意原文给任务入口。

新任务返回 202，沿用 httpresp envelope：

```json
{"code": 202, "msg": "accepted", "data": {"id": "job-id", "article_id": "article-id", "status": "queued", "phase": "queued"}}
```

### 查询任务

`GET /api/v1/admin/content-translations/:id` 返回 200。字段：id、article_id、status、phase、check_status、issues、reason、article_version、created_at、updated_at。未产生结果的可选字段固定省略。

任务失败通过 status=failed 和 reason 表示；查询成功不等于翻译成功。完成后重新读取现有文章接口。文章详情附加 `active_translation_job_id`，便于刷新页面后恢复查询。

新增 `GET /api/v1/admin/content-translation-settings`，仅返回 enabled、支持语言和非敏感长度限制，用于客户端提示。不得返回 Key、平台认证信息或完整内部配置。

| HTTP / 场景 | reason |
| --- | --- |
| 400 原文未确认或与保存值不符 | `CONTENT_SOURCE_REQUIRED` / `CONTENT_SOURCE_MISMATCH` |
| 400 目标语言不支持或与原文相同 | `CONTENT_TRANSLATION_LANGUAGE_INVALID` |
| 400 超过产品长度 | `CONTENT_TRANSLATION_TOO_LONG` |
| 409 内容版本变化 | `CONTENT_VERSION_CONFLICT` |
| 409 已有活动任务 | `CONTENT_TRANSLATION_IN_PROGRESS` |
| 409 幂等键对应其他请求 | `CONTENT_TRANSLATION_IDEMPOTENCY_CONFLICT` |
| 503 功能未启用 | `CONTENT_TRANSLATION_DISABLED` |
| 任务失败：上游不可用/限流 | `CONTENT_TRANSLATION_PROVIDER_UNAVAILABLE` / `CONTENT_TRANSLATION_RATE_LIMITED` |
| 任务失败：超时/中断 | `CONTENT_TRANSLATION_TIMEOUT` / `CONTENT_TRANSLATION_INTERRUPTED` |
| 任务失败：不完整/格式非法 | `CONTENT_TRANSLATION_INCOMPLETE` / `CONTENT_TRANSLATION_INVALID_OUTPUT` |

认证、403、404 沿用项目规则。未知错误只在边界记录一次，返回通用消息。前端根据 reason 显示文案；5xx 展示请求 ID。质量 warning 是正常数据，不使用错误 HTTP 状态。

## 9. 文件与实施任务

以下新文件名称是计划中的边界，不要求拆分或重写现有无关功能。

### 任务一：配置和 Gemini 适配器

**新增：** `templates/quickstart/internal/feature/content/gemini.go`、`gemini_test.go`、`translation_types.go`（均在 content 包）。

**修改：** `templates/quickstart/internal/hostcfg/hostcfg.go`、`templates/quickstart/internal/bootstrap/config.go`、`config_schema_test.go`、`templates/quickstart/deploy/config.yaml.example`、`docs/CONFIG_STANDARD.md`。

**边界：** `TranslationProvider.Generate(ctx, source, sourceLocale, targetLocale)` 返回 Translation；`Review(ctx, source, candidate, sourceLocale, targetLocale)` 返回 ReviewResult；`Repair(ctx, source, candidate, review, sourceLocale, targetLocale)` 返回 Translation。每个方法均同时返回 error。source/candidate 类型使用现有 Translation；ReviewResult 定义 Passed 和 Issues，Issue 定义 Code、Field、MessageZH。

- [x] 先写配置缺失、非法地址、禁用状态、认证头、超时、截断和非法返回测试。
- [x] 运行 `go test ./internal/hostcfg ./internal/bootstrap ./internal/feature/content`，确认新测试先失败。
- [x] 实现配置验证、Gemini 请求构造及有界响应读取。响应体最大 2 MiB，超限返回 INVALID_OUTPUT。
- [x] 实现生成、检查、修正三个独立请求模板和返回结构验证。
- [x] 运行同一组测试，确认通过。测试不调用真实 API。
- [ ] 执行第 6 节手动平台验证，记录平台实际输出能力；失败时保持 enabled=false。

### 任务二：内容元数据与任务存储

**新增：** `templates/quickstart/internal/feature/content/translation_repository.go`、`translation_repository_test.go`、`translation_validation.go`、`translation_validation_test.go`。

**修改：** `templates/quickstart/internal/feature/content/content.go`、`templates/quickstart/internal/bootstrap/host_migrate.go`（如现有 Models 注册已覆盖，只扩展 content.Models）。

**边界：** 内容服务负责创建、查询、领取、完成和失败任务。任务仓储封装全部数据库访问；终态提交必须同时校验版本和租约。

- [x] 先写旧文章加载、source_locale 兼容保存、字数边界及 hash 失效测试。
- [x] 写幂等重复、同键异体、同篇并发任务、版本变化和过期工作器提交失败测试。
- [x] 运行 `go test ./internal/feature/content`，确认新增场景失败。
- [x] 增加 GORM 字段与任务模型，不执行生产数据迁移，不修改已有 SQL migration。
- [x] 实现短事务提交及唯一约束，确保每种失败都不写 LiveJSON。
- [x] 运行同一测试，验证正常完成也不改变 published 快照。

### 任务三：工作器与有限检查流程

**新增：** `templates/quickstart/internal/feature/content/translation_service.go`、`translation_service_test.go`、`translation_worker.go`、`translation_worker_test.go`。

**修改：** `templates/quickstart/internal/bootstrap/host_jobs.go`、`templates/quickstart/internal/bootstrap/app.go`（生命周期接入）。

**边界：** 服务消费 TranslationProvider、任务仓储和注入时钟。工作器以应用 context 运行，通过任务租约确保唯一执行，不负责 HTTP 响应。

- [x] 先用假 provider 写调用序列测试：通过为两次调用，修正后为四次调用。
- [x] 写复查仍失败保存 warning、检查技术失败保留草稿、任务取消和租约到期测试。
- [x] 运行 `go test ./internal/feature/content ./internal/bootstrap`，确认新场景失败。
- [x] 实现状态机、阶段更新、deadline、程序检查与质量问题合并。
- [x] 接入工作器启停与租约回收；测试使用假时钟，不 sleep 等待。
- [x] 运行 `go test -race ./internal/feature/content ./internal/bootstrap`，确认没有并发重复执行和失效写入。

### 任务四：管理 API

**新增：** `templates/quickstart/internal/feature/operations/content_translation.go`、`content_translation_test.go`。

**修改：** `templates/quickstart/internal/feature/operations/operations.go`、`content.go`、`templates/quickstart/internal/hostapi/` 中实际依赖结构及组合位置（沿现有 Deps 注入，不新增全局客户端）。

- [x] 先写未登录、非管理员、文章缺失、版本冲突、禁用和相同幂等请求测试。
- [x] 运行 `go test ./internal/feature/operations`，确认新端点用例失败。
- [x] 注册第 8 节三个端点，集中扩展现有 contentError 映射，不建立第二套错误响应格式。
- [x] API 只创建任务和查询状态，不同步等待 AI；任务及审计记录正确关联。
- [x] 运行同一测试，验证公开文章接口不会返回任务、原文草稿或检查信息。

### 任务五：管理页面

**新增：** `templates/quickstart-nextjs/lib/content-translation.ts`、`templates/quickstart-nextjs/tests/content-translation.test.ts`。

**修改：** `templates/quickstart-nextjs/lib/api.ts`、`lib/content-api.ts`、`components/content-admin.tsx`、`tests/content-api.test.ts`。

- [x] 先写源/目标选择、字数、任务状态文案、warning 可发布、failed 不覆盖的纯逻辑测试。
- [x] 运行 `npm test`，确认新行为用例失败。
- [x] 增加严格类型和 apiRequest 封装。组件不能直接 fetch，也不能读取 Key。
- [x] 实现自动保存后创建任务；重试复用同一意图键，主动重新翻译使用新键。
- [x] 每 2 秒查询活动任务，终态停止。组件卸载停止查询，刷新页面通过 active_translation_job_id 恢复。
- [x] 对轮询失败显示可重试提示，不把网络失败误判为任务失败。
- [x] 工作中禁用同篇编辑和翻译操作，跨标签页冲突仍由后端版本校验处理。
- [x] 复用预览和发布按钮；质量警告保持可发布，修改后的检查结果显示失效。
- [x] 运行 `npm run verify`，检查窄屏提示、按钮标签及 role=status/alert。

### 任务六：端到端验收与说明

**修改：** `templates/quickstart/internal/feature/content/README.md`、`docs/CONFIG_STANDARD.md`，以及本计划的验收记录。

- [x] 在隔离的本地数据库准备中文、英文及超长文章，不使用生产数据。
- [x] 发布旧译文，再重新翻译；确认访客仍看到旧内容，管理员预览看到新草稿。
- [x] 手动发布目标语言；确认只更新该语言，其他语言保持原版本。
- [x] 使用假 provider 触发 warning；确认管理员仍能发布，且审计记录对应检查状态。
- [x] 触发截断、检查超时、重启和版本冲突；确认旧草稿及线上内容保持不变。
- [x] 完成下面的检查命令并记录实际结果。未执行的检查必须写明，不能宣称通过。

## 10. 验收矩阵

| 场景 | 预期 |
| --- | --- |
| 中文译英文、英文译中文 | 标题、摘要、正文完整；可选 SEO 按原文处理 |
| 首次检查通过 | 两次模型调用，保存 passed 草稿 |
| 修正后通过 | 四次调用，保存 passed 草稿 |
| 修正后仍有质量问题 | 四次调用，保存 warning，允许手动发布 |
| 任一步 API 失败、截断、非法 JSON | failed，原草稿和线上内容不变 |
| 重复点击或网络重试 | 相同任务，不重复扣费调用；进程中断的不确定收费另行说明 |
| 原文在执行时被修改 | 最终写入冲突，不能覆盖任何草稿 |
| 检查后人工修改译文 | 检查状态失效，仍允许发布 |
| 切换原文语言 | 保留内容，之前检查失效，下一次翻译使用新原文 |
| 长度恰好上限/超过一个单位 | 前者允许，后者只禁止翻译，仍可保存 |
| 未配置 Key 或功能关闭 | 页面说明不可用；既有编辑、预览、发布不受影响 |
| 刷新页面/工作器重启 | 页面能查询已保存任务；中断任务有终态，不无限等待 |
| 普通用户或匿名访问任务 | 拒绝访问，不泄露原文和结果 |

## 11. 检查命令与发布控制

后端模板目录 `templates/quickstart`：

```bash
gofmt -s -l internal/feature/content internal/feature/operations internal/hostcfg internal/bootstrap
go vet ./...
go test ./...
go test -race ./...
```

前端目录 `templates/quickstart-nextjs`：

```bash
npm run verify
```

仓库根目录：

```bash
make purity-check
git diff --check
```

若实施时确实改动 foundation 或 modules，另运行 `make fmt && make test-race && make purity-check`、`go vet ./...`，并为公共 API 的增量变化更新对应 CHANGELOG 的 Unreleased。当前方案不要求改这些包。

关闭 `host.content_translation.enabled` 即可停止接受新任务。工作器停止领取新任务，并将未执行任务标记失败；运行任务在关闭/退出时取消，提交前再次检查开关与租约。保留数据库新增字段，既有内容继续可读。不可通过删除表或回退已有数据来关闭功能。

## 12. 外部资料与已知限制

资料核对日期：2026-10-26。以下为文档证据，不是本账户实测结果。

- [Google Gemini 3.8 Flash](https://ai.google.dev/gemini-api/docs/models/gemini-3.8-flash)：官方输入 1,048,576 tokens，输出 65,536 tokens，支持结构化输出。
- [Google GenerateContent](https://ai.google.dev/api/generate-content#FinishReason)：读取结束原因，不能把 MAX_TOKENS 或未完成 continuation 当作成功。
- [AIGoCode 模型列表](https://www.aigocode.net/docs/api/models)：列出 gemini-3.8-flash，实际权限依赖 Key 分组。
- [AIGoCode Gemini 协议](https://www.aigocode.net/docs/api/gemini-compatible)：Gemini 分组使用原生协议，不提供 OpenAI 兼容协议。
- [AIGoCode 地址说明](https://www.aigocode.net/docs/getting-started/base-url)：当前文档地址为 api.aigocode.dev；用户提供的是 api.aigocode.app，接入时必须验证。
- [AIGoCode 限制说明](https://www.aigocode.net/docs/account/limits-and-credits)：未明确给出本账户的最大输入输出；不得直接套用 Google 官方上限。

AI 检查不能保证翻译绝对准确。界面使用“检查未发现明显问题”，不使用“保证准确”。第一版不自动发布，不增加人工审核门槛；管理员保留最终发布决定。

### 本文交付记录

- 本次仅编写开发文档，未实现功能、未调用收费 API、未修改业务代码。
- 接口验证属于实施任务，当前没有本账户实测数据。

### 2026-10-26 实施记录

- 任务一至任务六的代码与自动化测试已全部落地：Gemini 原生适配器（`content/gemini.go`）、
  任务存储与元数据（`translation_repository.go`、`translation_validation.go`）、后台工作器与
  有限检查流程（`translation_service.go`、`translation_worker.go`、`bootstrap/host_jobs.go`）、
  管理 API（`operations/content_translation.go`）与管理页面（`lib/content-translation.ts`、
  `components/content-admin.tsx`）。
- 端到端验收由 `operations/content_translation_flow_test.go` 在内存数据库中以假 provider 覆盖：
  任务运行与 warning 状态下访客始终看到旧内容；显式发布后仅目标语言更新、原文语言保持原版本；
  管理员可发布带 warning 的译文；技术失败保留全部草稿。截断（MAX_TOKENS）、超时、
  重启回收（租约过期）与版本冲突由 `internal/feature/content` 的单元测试逐项覆盖。
### 2026-10-26 第四轮：代码块翻译与表单交互

- 用户反馈代码块内注释未翻译：这是原提示词"代码块原样保留"的设计行为，按用户需求调整——
  生成/修正提示词要求翻译代码块内的人语言注释与显示文本（给出示例），命令、语法、标识符、
  文件路径不变；审查提示词同步对齐（翻译注释不算损伤，注释未翻译才算）；程序检查放宽为
  代码块结构保留（围栏数量与语言标记一致）即可，不再要求逐字相同。实测：`# 安装依赖包` →
  `# install dependencies`、`echo "安装完成"` → `echo "Setup complete"`，命令与包名原样，
  检查 passed。首轮还暴露审查端规则未同步导致 warning 与生成行为矛盾，已一并修复。
- 表单交互按用户要求改造：保存草稿始终可点，提交时校验并圈红（aria-invalid）必填项
  （路径格式、操作原因 3-500 字），路径非法提示显示在字段下方（field-error），不再加载即红；
  移除输入框原生 pattern/required，表单 noValidate，避免浏览器原生气泡与自绘提示混用。

### 2026-10-26 第三轮：新增文章路径守卫

- 用户实测暴露：新增文章保留默认路径 `/blog/`（未填别名）直接点"翻译并检查"，自动保存被后端
  400 拒绝，界面只显示"请检查填写内容后重试"，未说明原因。根因是翻译按钮不触发表单原生校验。
- 修复：前端新增 `isValidContentPath`（与后端 validKind 规则逐条一致，含栏目前缀锚定）；路径
  字段非法时实时红框与示例提示；"翻译并检查"与"保存草稿"两个入口均前置校验，给出具体示例
  文案后再放行请求。
- 复现与修复后回归均通过；探针数据已清理。

### 2026-10-26 浏览器实测与第二轮修复记录

- 管理页面以管理员身份完成十二项交互覆盖：新增文章直接翻译（自动创建草稿）、双向翻译全流程、
  未确认原文/空草稿/超长上限三重守卫、运行中刷新恢复、并发版本冲突（任务失败且草稿完好）、
  主动重翻译新键、原文更新指示、失败展示、检查失效、阶段轮询文案。全部通过。
- 本轮修复：新增文章点"翻译并检查"无反应（改为自动走现有创建流程）；补齐第 4.9 条"原文有更新"
  指示（后端 MarkSourceStaleness，GET/保存/发布响应统一携带）；任务终态清除旧提交消息；
  终态刷新改用任务内 article_id 消除新建后 id 竞态。
- 实测新发现：Gemini 思考型输出偶发一次 INVALID_OUTPUT（重试成功），系统按设计保留原稿。
- 第四轮补充（用户反馈对齐问题）：错误提示把 .editorMeta 双列字段撑高后左列被居中下移。修复：
  `.editorMeta` 加 `align-items: start`、`.editorMeta .field` 加 `align-content: start`，错误态下
  两列输入框上下边缘一致（几何断言 yDelta=0），提示只向本列下方延伸。

- 检查命令实测（2026-10-26，全部通过）：`gofmt -s -l` 无输出；`go vet ./...` 通过；
  `go test ./...` 与 `go test -race ./...` 18 个包全部通过；`make purity-check` 通过；
  `git diff --check` 通过；`npm run verify` 通过（317 项测试、lint、构建与公开内容检查）。
- 端点确认（2026-10-26，用户提供）：实际使用 `https://api.aigocode.dev`，与配置默认值一致。

### 2026-10-26 平台实测记录（AIGoCode Gemini 原生协议）

- 端点 `https://api.aigocode.dev/v1beta/models/gemini-3.8-flash:generateContent`；Key 从本地
  `.env` 读取，全程未出现在任何输出中。curl 与 Go 客户端均通过认证；Python urllib 被 Cloudflare
  拦截（error 1010，TLS 指纹判定），生产使用的 Go 客户端不受影响。
- 模型为思考型：响应先带 `thought: true` 的推理片段，再给答案片段。适配器已修复为跳过推理片段
  （`internal/feature/content/gemini.go`，含回归测试）；此前拼接两类文本导致结构化解析失败——
  这正是本次实测要抓的问题。
- 实测耗时（探针经生产适配器真实调用）：短文生成 9.1s；短文检查 8.0s；接近上限长文（zh 源
  4,933 非空白字符）zh→en 23.3s、译文 1,308 词；接近上限长文（en 源 2,988 词）en→zh 45.9s；
  另一次超标长文（约 25,824 字符，超出产品上限）实测 89.8s 仍正常完成。全部 finishReason=STOP，
  链接目的地址与代码块完整保留，结构化五字段返回、可选 SEO 字段保持为空。
- 检查调用结构验证通过；短文示例返回 passed=false 与 1 条质量问题，审查偏严格——warning 流程
  会较常见，管理员仍可发布，属设计内行为。
- 基于实测将默认 `request_timeout_seconds` 由 90 调整为 240、`task_timeout_seconds` 由 420 调整
  为 1200（思考型模型长文单次可近 90 秒，最多四次调用需要更大任务预算）。
- 未执行（如实说明）：平台侧用量/计费记录未抓取（探针未输出 token 计数），以 AIGoCode 控制台
  为准，上方对应清单项保持未勾选。浏览器端人工走查待用户进行；本地 `deploy/config.yaml` 已启用
  `host.content_translation.enabled: true` 供走查，示例配置与文档默认仍为 false。
- 未执行项（如实说明）：第 6 节的手动平台验证——真实 AIGoCode/Gemini 调用、`.dev` 与 `.app`
  端点实测、结构化返回与 finishReason 观察、耗时与用量记录——需要账户与 API Key，实施期间
  保持 `enabled: false`，接入时按第 6 节清单执行。浏览器端人工走查（窄屏提示、按钮标签的实际
  观感）未执行，`npm run verify` 已覆盖 lint/构建/role 属性等静态检查。
