# 部署与环境规范

第三方服务的配置方法见 [SETUP_ZH.md](../SETUP_ZH.md)，配置项说明见 [CONFIG_STANDARD.md](../CONFIG_STANDARD.md)。

## 环境

| 环境 | `APP_ENV` | 用途 | 数据 | 第三方服务 |
| --- | --- | --- | --- | --- |
| 本地 | `dev` | 开发 | 本地 PostgreSQL | 邮件用 `log`，Stripe 用测试模式 |
| 预发布 | `staging` | 合并前后的完整验证、演示 | 独立的数据库，使用假数据 | 全部使用测试模式的 key |
| 生产 | `prod` | 正式服务 | 生产数据库 | 正式 key |

- `APP_ENV` **只使用这三个值**。可观测性数据里的 `deployment.environment.name` 来自它；
- `APP_ENV=prod` 时会启用生产环境的配置校验（比如邀请链接必须是 https），配置不符合要求时服务无法启动；
- 每个环境使用独立的数据库、密钥、OAuth 应用和 Stripe Webhook；**禁止**让预发布环境连接生产数据库。

## 发布流程

```text
推送 / 合并到 main ──▶ GitHub Actions：全部检查 ──通过──▶ deploy 任务调用 Dokploy API ──▶ 生产
                                         └──失败──▶ 不部署，修复后重新推送
```

**部署由 CI 触发，而不是由推送触发**：CI 没有通过的代码不会上线。

1. 推送或合并到 `main` 后，CI 运行全部检查（[CI_QUALITY.md](./CI_QUALITY.md#必需的检查)）；
2. 全部通过后，`deploy` 任务调用 Dokploy 部署，Dokploy 使用仓库根目录的 `Dockerfile` 构建
   （通过 `go.work` 使用同一个提交里的 `foundation` 和 `modules`）；
3. 部署完成后，执行下面的**发布后检查**；
4. 有数据库结构变更时，按 [DATABASE.md](./DATABASE.md#不停机变更先扩展后收缩) 拆成多次发布。

### 一次性配置

1. **Dokploy**：打开每个由 CI 部署的应用 → **关闭 Auto Deploy**。否则推送后 Dokploy 会立即部署，
   不等 CI；
2. **Dokploy**：Settings → Profile → **API/CLI** → 生成 API Token；
3. 记下应用 ID：打开应用页面，地址中 `.../services/application/<applicationId>` 的最后一段；
4. **GitHub**：Settings → Environments → 新建 `production`（可以在这里设置需要人工批准才能部署），
   然后在这个环境里添加 Secrets：

| Secret | 值 |
| --- | --- |
| `DOKPLOY_URL` | Dokploy 面板地址，例如 `https://dokploy.example.com` |
| `DOKPLOY_API_KEY` | 第 2 步生成的 token |
| `DOKPLOY_APPLICATION_IDS` | 应用 ID；多个应用（比如后端和前端）用逗号分隔 |

没有配置这些 Secret 时，`deploy` 任务会给出警告并跳过，不会导致 CI 失败。

### 紧急情况
CI 本身出了故障、又必须马上发布时，可以在 Dokploy 面板里手动点击 Deploy。事后补上 CI 检查，
并在复盘中记录原因。

### 发布后检查（每次部署后执行，5 分钟）
- [ ] `curl https://api.<域名>/health` 返回 200；
- [ ] 容器日志里有 `tracing ready`，其中 `auth_header=true`；1 分钟内没有 `opentelemetry error`；
- [ ] Grafana 里能查到新版本的链路，`service.version` 正确；
- [ ] 错误率、P99 延迟和发布前相比没有明显变化；
- [ ] 本次改动涉及的功能，手动走一遍。

## 发布时机与范围

- 尽量在工作时间、流量低的时段发布，发布后至少观察 30 分钟；
- 一次发布只包含一组相关的改动；数据库迁移和大的功能改动不在同一次发布中；
- 周五晚上和节假日前不做有风险的发布。

## 回滚

| 情况 | 做法 |
| --- | --- |
| 新版本有问题，没有数据库变更 | Dokploy 回滚到上一个版本（或对出问题的提交执行 `git revert` 再走 PR） |
| 有数据库变更 | 按"先扩展后收缩"拆分发布时，旧版本代码兼容新的数据库结构，可以直接回滚代码；数据迁移脚本要附带回滚 SQL |
| 不确定是否需要回滚 | **先回滚，再排查** |

- 在 Dokploy 里开启"部署失败时自动回滚"，健康检查使用 `/health`；
- 记录每次发布对应的提交，确保随时知道"上一个正常的版本"是哪个。

## 配置和密钥

- 非敏感的默认值放在 `deploy/config.yaml.example`；敏感值只放在部署平台的环境变量里；
- 在部署平台里填写环境变量时，**不要给值加引号，也不要写行内注释**：有的平台会把它们当成值的一部分；
- 修改环境变量后要重新部署才会生效；修改后检查启动日志，确认配置被正确读取；
- 新增必填的配置项时，先在所有环境里设置好，再发布读取它的代码。

## 功能开关

- 未完成的功能通过配置开关（`APP_HOST_*`）关闭后再合并，不要长期保留未合并的分支；
- 模块级别的开关见 [CONFIG_STANDARD.md](../CONFIG_STANDARD.md#模块开关)；
- 功能稳定后删除开关和旧代码。

## 可观测性

- 每个服务在部署时都要配置 OTLP 的环境变量和 `APP_PROJECT`、`APP_ENV`、服务名（见 [OBSERVABILITY.md](../OBSERVABILITY.md#配置)）；
- 每台服务器部署一个 Alloy 采集日志（Railway 等托管平台除外，见 OBSERVABILITY.md）；
- 新服务上线时导入告警规则（`deploy/alerts/rules.yaml`）。

## 上线验收清单

新项目首次上线，或者涉及登录、支付、邮件的重大改动上线时，按顺序执行：

### 基础
- [ ] `/health` 返回 200；API 和前端都有 HSTS、`nosniff`、`X-Frame-Options` 等安全头；
- [ ] `/api/v1/capabilities` 显示的模块开关和配置一致；
- [ ] http 自动跳转到 https；不在白名单里的域名发起的跨域请求被拒绝；
- [ ] 不带 token 访问用户接口和后台接口返回 401。

### 登录
- [ ] 邮箱验证码：能收到邮件且不在垃圾箱；频率限制和错误次数上限生效；登录、刷新、退出后 token 失效；
- [ ] Google / GitHub：新用户注册；同一个邮箱的已有用户被正确关联；取消授权有友好提示；回调地址不能重复使用；
- [ ] 管理员：密码登录成功；普通用户访问后台返回 403；日志里没有密码明文。

### 邮件
- [ ] 邮件服务商后台显示已送达；SPF、DKIM、DMARC 检查通过；
- [ ] 发送失败时，接口返回明确的错误，日志里没有 API key。

### 支付（Stripe）
- [ ] Webhook 地址正确，订阅了所有需要的事件，Dashboard 里的送达记录都是 200；
- [ ] 订阅、升级、预约降级、取消、恢复、账单门户、发票列表都正常；
- [ ] 在 Dashboard 里重发同一个事件，积分和订阅不会重复增加；
- [ ] 被拒的卡、需要 3DS 验证的卡、Checkout 过期、退款、续费失败等场景处理正确；
- [ ] 买断和积分包：权益正确发放；
- [ ] 测完把测试模式的 key 换回正式 key，用真实卡付一笔最小金额后退款。

### 邀请
- [ ] 邀请链接正确；被邀请的用户注册后建立邀请关系；对方付费后邀请人获得奖励。

### 可观测性
- [ ] 按 `X-Request-ID` 能在 Tempo 里找到对应的链路，SQL 和第三方调用挂在请求下面；
- [ ] Prometheus 里有 `target_info` 和 `http_server_request_duration_seconds`；
- [ ] Loki 里能按 `request_id` 查到日志，链路和日志能互相跳转；
- [ ] 告警规则已导入并处于 Normal 状态，通知渠道测试通过；
- [ ] 链路的 SQL 和 URL 里没有参数值、token 等敏感信息。

### 收尾
- [ ] 生产环境的采样率改回 0.1–0.2；
- [ ] 在服务器上扫描容器日志，确认没有密钥泄露：
  `docker logs <容器> 2>&1 | grep -Ei 'sk_live|whsec_|Bearer [A-Za-z0-9]'` 没有输出。
