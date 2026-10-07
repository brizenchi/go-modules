# 前端规范（Next.js）

适用于 `templates/quickstart-nextjs` 以及从它复制出去的前端项目。

## 目录

| 目录 | 放什么 |
| --- | --- |
| `app/` | 路由和页面（App Router）。页面只负责组合组件和读取数据，不写复杂逻辑 |
| `components/` | 可复用的组件；样式使用同名的 `*.module.css` |
| `lib/` | 不依赖 React 的逻辑：API 请求（`api.ts`）、状态计算、格式化、环境变量（`env.ts`） |
| `content/` | 博客、文档等静态内容 |
| `tests/` | 测试，文件名是 `<被测模块>.test.ts` |

逻辑尽量放进 `lib/`，写成纯函数，这样不需要渲染组件就能测试。

## 代码风格

- TypeScript 严格模式；不使用 `any`，确实不知道类型时使用 `unknown` 再收窄；
- `npm run lint`（ESLint，`eslint-config-next`）必须通过；
- 组件使用函数组件和 Hooks；组件文件名使用小写加 `-`：`sign-in-panel.tsx`；
- 样式使用 CSS Modules，不写内联样式（动态计算的值除外）；
- 面向用户的文案使用 `t({ en: "...", zh: "..." })`（`lib/i18n.tsx`），不直接写死某一种语言。

## 请求规范

### 统一使用 `lib/api.ts`
- **所有后端请求都通过 `apiRequest` 或 `api.ts` 里封装好的函数发出**，组件里不直接调用 `fetch`；
- 请求和响应的类型定义在 `api.ts` 里，和后端的结构体保持一致。后端修改接口时，**同一个 PR 里同步修改这里**（见 [API_STANDARD.md](./API_STANDARD.md#接口契约)）；
- `apiRequest` 负责：拼接 API 地址、带上 `Authorization`、解析 `{code,msg,data}` 外壳、失败时抛出 `ApiError`、遇到 401 时清除登录状态。

### 错误处理
`ApiError` 包含这些字段：

| 字段 | 含义 |
| --- | --- |
| `status` | HTTP 状态码 |
| `code` | 外壳里的 `code` |
| `message` | 后端的 `msg`，给开发人员看，**不要直接展示给用户** |
| `reason` | 后端返回的稳定错误原因，比如 `AUTH_INVALID_CODE`（见 API_STANDARD.md） |
| `requestId` | 这次请求的 `X-Request-ID` |

- 用户看到的提示由前端根据 `status` 或 `reason` 决定，并且要有多语言版本。参考 `components/console-kit.tsx` 里的 `ConsoleError`，以及 `lib/request-state.ts` 里的 `describeRequestFailure`；
- **服务端错误（5xx）要显示请求编号**，方便用户反馈、开发人员在 Grafana 里定位：`ConsoleError` 和 `describeRequestFailure` 已经实现了这一点；
- 网络错误（`TypeError`）提示"无法连接服务器"，并允许重试；
- 401 时引导用户重新登录；403 提示没有权限；不要把这两种情况当成"出错了"处理。

### 写操作
- 提交按钮在请求进行中禁用，防止重复提交；
- 后端要求 `Idempotency-Key` 的操作，用 `newIntentKey()` 生成一次，**重试时使用同一个 key**；
- 不要在请求失败后自动重试写操作，除非确认这个接口是幂等的。

### 数据加载状态
使用 `lib/request-state.ts` 中的 `ResourceState`：`idle` → `loading` → `ready` 或者 `error`。每个页面都要处理加载中、空数据、出错三种状态。

## 认证

- 登录状态的读写只通过 `lib/auth.ts`；不在组件里直接操作 localStorage 或 cookie；
- token 不出现在 URL、日志或错误上报里；
- 页面上的权限判断（比如隐藏管理员入口）只用于界面展示，后端仍然要做完整的鉴权。

## 环境变量

- 浏览器端能读到的变量必须以 `NEXT_PUBLIC_` 开头，**不能放任何密钥**；
- 统一在 `lib/env.ts` 里读取和校验，提供默认值；组件里不直接读 `process.env`。

## 安全

- 不使用 `dangerouslySetInnerHTML` 渲染用户输入；Markdown 内容使用经过安全处理的渲染器；
- 登录后、支付后的跳转地址只允许本站的路径；
- 外部链接使用 `rel="noopener noreferrer"`。

## 可访问性和性能

- 表单控件有关联的 `label`；错误提示使用 `role="alert"`；可点击的元素使用 `button` 或 `a`；
- 图片有 `alt`；优先使用 `next/image`；
- 能在服务端渲染的公开页面（首页、定价、博客）使用服务端组件；只有需要交互的部分才使用 `"use client"`。

## 测试

- 测试使用 Node 内置的 `node:test`，运行 `npm test`；
- `lib/` 里的逻辑必须有测试；修改 `api.ts` 时，用替换 `globalThis.fetch` 的方式测试请求和错误处理（参考 `tests/api.test.ts`）；
- 合并前运行 `npm run verify`（测试 + lint + 构建 + 内容检查），CI 也会运行它。
