<!--
PR title = final squash commit: type(scope): subject
e.g. fix(api): reject expired tokens — see docs/standards/GIT_WORKFLOW.md
-->

## 改了什么、为什么

<!-- 背景、问题、方案。关联 issue：Closes #123 -->

## 怎么验证的

<!-- 运行了哪些测试；手动验证的步骤；截图或日志 -->
- [ ] `make fmt && make test-race && make purity-check`
- [ ] `cd templates/quickstart && go test ./...`（修改了模板时）
- [ ] `cd templates/quickstart-nextjs && npm run verify`（修改了前端时）

## 影响与风险

- [ ] 修改了 `foundation/*` 或 `modules/*` 的公开 API：只做增量修改，并更新了对应包的 `CHANGELOG.md`
- [ ] 修改了接口：调用方（前端、其他服务）的类型或客户端在同一个 PR 里更新
- [ ] 修改了数据库：新增了迁移，没有修改已执行过的迁移，可以回滚
- [ ] 新增了配置项：示例配置和文档已更新，各环境已设置
- [ ] 新增了依赖：在下面说明了理由

<!-- 回滚方式；需要注意的发布顺序 -->

## 自查清单

- [ ] 符合 [代码审查清单](../docs/standards/CODE_REVIEW.md#审查清单)
- [ ] 没有提交密钥；测试里的假密钥明显是假的（`*_not-a-real-key`）
