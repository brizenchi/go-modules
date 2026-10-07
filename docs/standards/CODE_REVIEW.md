# 代码审查规范

## 目标

审查要发现的是：**正确性问题、安全问题、违反架构边界的写法、以后难以维护的代码**。
代码风格交给 gofmt、golangci-lint 和 ESLint 处理，审查时不再讨论。

## 时效

| 角色 | 要求 |
| --- | --- |
| 审查者 | 1 个工作日内给出第一轮意见；hotfix 尽快处理 |
| 作者 | 每条意见都要回应：修改，或者说明不修改的理由 |
| 双方 | 来回超过 3 轮仍有分歧时，改为当面或语音沟通，结论写回 PR |

## 意见分级

在评论开头标明级别，让作者知道哪些必须改：

| 前缀 | 含义 |
| --- | --- |
| `blocking:` | 必须修改才能合并：缺陷、安全问题、破坏架构边界 |
| `suggestion:` | 建议修改，作者可以自行决定 |
| `nit:` | 小问题，不改也可以 |
| `question:` | 不理解的地方，需要作者解释 |

## 审查清单

### 正确性
- [ ] 边界情况：空值、空列表、重复请求、并发、超时
- [ ] 错误都被处理了，并且按 [错误处理规范](./CODE_STYLE.md#错误处理) 只记录一次日志
- [ ] `ctx` 一路传递；外部调用设置了超时
- [ ] 涉及金额、积分的逻辑是幂等的（Webhook 重放、用户重复提交）

### 架构
- [ ] 符合 [ARCHITECTURE.md](../ARCHITECTURE.md) 的分层：`foundation` 不依赖 `modules`；模块之间不直接互相导入
- [ ] 新的业务功能放在 `internal/feature/*`，跨模块的组合放在 `internal/bootstrap` 或 `internal/platform`
- [ ] 公开 API 的改动只做增量修改；破坏性变更有迁移说明

### 接口和数据
- [ ] 符合 [API_STANDARD.md](./API_STANDARD.md)：路径、响应外壳、错误码、分页、时间格式
- [ ] 前端类型（`lib/api.ts`）和后端同步更新
- [ ] 数据库改动符合 [DATABASE.md](./DATABASE.md)：已执行过的迁移文件没有被修改，大表加了索引

### 安全
- [ ] 符合 [SECURITY_STANDARD.md](./SECURITY_STANDARD.md)：有鉴权和越权检查、输入校验、不打印敏感字段
- [ ] 没有提交真实密钥；测试里的假密钥明显是假的

### 可观测性
- [ ] 日志使用 `slog.*Context(ctx, ...)`，级别符合 [OBSERVABILITY.md](../OBSERVABILITY.md#日志级别)
- [ ] 新增的第三方调用使用注入的 `HTTPClient`

### 测试
- [ ] 符合 [TESTING.md](./TESTING.md)：新逻辑有测试，修复的缺陷有回归测试
- [ ] 测试不依赖外部网络，不依赖执行顺序

## 作者自查

提交审查前，作者先过一遍上面的清单，并在本地运行：

```bash
make fmt && make test-race && make purity-check
cd templates/quickstart && go test ./...
```

## AI 生成的代码

AI 生成的代码和人写的代码执行同一套标准。作者要能解释每一行代码；不能只因为 AI 写了，就跳过测试或审查。
