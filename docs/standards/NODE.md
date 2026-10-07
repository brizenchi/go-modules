# Node / TypeScript 规范

通用规则见 [CODE_STYLE.md](./CODE_STYLE.md)。

## 工具

| 工具 | 内容 | 在哪里运行 |
| --- | --- | --- |
| TypeScript `strict` | 类型检查 | `build` 或 `typecheck` 脚本 |
| ESLint | 静态检查 | `lint` 脚本，CI `node-* / check` |
| Prettier（如果项目使用） | 格式 | 编辑器、AI 助手的格式化钩子 |
| 测试框架 | 单元测试 | `test` 脚本，CI `node-* / check` |
| 依赖审计 | 生产依赖漏洞 | CI `node-* / audit`（不阻止合并） |

CI 会按 `.copier-answers.yml` 中配置的顺序运行 `package.json` 里的脚本，没有的脚本会跳过。
包管理器根据锁文件自动识别（`package-lock.json`、`pnpm-lock.yaml`、`yarn.lock`），**必须提交锁文件**。

## 代码

- TypeScript 严格模式；不使用 `any`，类型不确定时使用 `unknown` 再收窄；
- 不依赖 React 等框架的逻辑放在独立的模块里，写成纯函数，方便单独测试；
- 文件名使用小写加 `-`：`sign-in-panel.tsx`；
- 面向用户的文案集中管理，支持多语言时不要直接写死在组件里。

## 请求规范

- **所有后端请求通过一个统一的客户端模块发出**，组件里不直接调用 `fetch`；
- 统一客户端负责：拼接地址、带上认证信息、解析统一的响应格式、失败时抛出带
  `status`、`reason`（稳定的错误原因）、`requestId`（响应头 `X-Request-ID`）的错误对象、处理 401；
- 请求和响应的类型和后端保持同步，后端修改接口时在同一个 PR 里更新（见 [API_STANDARD.md](./API_STANDARD.md#接口契约)）。

### 错误展示
- 用户看到的提示根据 `status` 或 `reason` 决定，**不直接展示后端返回的错误消息**；
- **5xx 错误显示请求编号**，方便用户反馈、开发者在日志和链路里定位；
- 网络错误提示"无法连接服务器"并允许重试；401 引导重新登录；403 提示无权限。

### 写操作
- 请求进行中禁用提交按钮，防止重复提交；
- 需要幂等的操作生成一个 `Idempotency-Key`，**重试时复用同一个 key**；
- 不自动重试非幂等的写请求。

## 认证与环境变量

- 登录状态的读写集中在一个模块里；token 不出现在 URL、日志或错误上报中；
- 浏览器端能读到的变量（如 `NEXT_PUBLIC_*`、`VITE_*`）**不能放密钥**；
- 环境变量在一个模块里统一读取和校验，组件里不直接读 `process.env`。

## 安全

- 不使用 `dangerouslySetInnerHTML` 渲染用户输入；
- 登录后、支付后的跳转地址只允许本站路径；
- 外部链接加 `rel="noopener noreferrer"`。

## 可访问性和性能

- 表单控件有关联的 `label`；错误提示使用 `role="alert"`；可点击元素使用 `button` 或 `a`；
- 图片有 `alt`；公开页面优先服务端渲染，只在需要交互的地方使用客户端组件。

## 测试

- 请求客户端、状态计算、格式化等纯逻辑必须有测试；
- 测试请求逻辑时替换 `fetch`，不访问真实后端。
