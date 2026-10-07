<!--
PR title = final squash commit: type(scope): subject
e.g. fix(billing): pass request ctx to Stripe calls — see docs/standards/GIT_WORKFLOW.md
-->

## 改了什么、为什么

<!-- 背景、问题、方案。关联 issue：Closes #123 -->

## 怎么验证的

<!-- 运行了哪些测试；手动验证的步骤；截图或日志 -->

- [ ] `make fmt && make test-race && make purity-check`
- [ ] `cd templates/quickstart && go test ./...`（修改了模板时）
- [ ] `cd templates/quickstart-nextjs && npm run verify`（修改了前端时）

## 影响与风险

- [ ] 修改了公开 API：只做了增量修改，并更新了对应包的 `CHANGELOG.md`
- [ ] 修改了接口：同步更新了前端类型 `lib/api.ts`
- [ ] 修改了数据库：新增了迁移文件，没有修改已有的迁移，可以回滚
- [ ] 新增了配置项：更新了 `deploy/config.yaml.example` 和文档，并已在各环境设置好
- [ ] 新增了依赖：在下面说明了理由

<!-- 回滚方式；需要注意的发布顺序 -->

## 自查清单

- [ ] 符合 [代码审查清单](../docs/standards/CODE_REVIEW.md#审查清单)
- [ ] 没有提交密钥；测试里的假密钥明显是假的
