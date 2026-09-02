# 模板

| 目录 | 用途 |
| --- | --- |
| `quickstart` | Go 后端模板，拥有用户模型、模块组合和业务监听器 |
| `quickstart-nextjs` | 与后端 API 配套的 Next.js 前端模板 |

先读 [quickstart 开发指南](./quickstart/README.md)，再按
[配置指南](../docs/SETUP_ZH.md) 开通数据库和第三方服务。

模板复制后允许分叉。共享安全修复通过升级 `go-modules` 依赖获得；用户字段和业务
回调不会被依赖升级自动覆盖。

当前分支包含尚未发布的共享 API 时，复制出来的后端可以临时在 `go.mod` 中 replace
到本地 `go-modules`。用于 CI 或生产前，应先发布新标签并改回正常版本依赖，详见
[版本管理](../VERSIONING.md)。
