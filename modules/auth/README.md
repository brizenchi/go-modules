# auth 认证模块

提供邮箱验证码、Google/GitHub OAuth、JWT、WebSocket 临时票据和认证领域事件。

## 边界

auth 不拥有用户表，也不发送欢迎邮件、赠送积分或处理邀请。宿主必须实现：

- `port.UserStore`：把当前 SaaS 的用户模型映射成最小 `domain.Identity`；
- `port.RoleResolver`：决定登录身份的粗粒度角色；
- 事件监听器：处理 `UserSignedUp`、`UserLoggedIn` 后的业务。

## 目录

```text
domain/   Identity、OAuthProfile、Token、错误
port/     UserStore、IdentityProvider、TokenSigner、EventBus 等接口
adapter/
  emailcode/  邮箱验证码流程
  google/     Google OAuth
  github/     GitHub OAuth
  jwt/        HMAC JWT 和 WebSocket 票据
  gormstore/  验证码、每日次数、OAuth 交换码表
  memstore/   测试和单实例开发内存 store
  eventbus/   进程内事件总线
app/      登录、OAuth、会话用例
http/     Gin handler、middleware、默认路由
```

## 组装

```go
module := auth.New(auth.Deps{
    UserStore:         hostUserStore,
    RoleResolver:      hostRoleResolver,
    TokenSigner:       signer,
    WSTicketSigner:    ticketSigner,
    TokenTTL:          time.Hour,
    WSTicketTTL:       30 * time.Second,
    ExchangeCodeStore: authStore,
    OAuthFlowStore:    authStore,
    EmailCodeIssuer:   issuer,
    EmailCodeVerifier: verifier,
    IdentityProviders: providers,
    Bus:               eventbus.NewInProc(),
    FrontendURL:       "https://app.example.com/login",
    OAuthCookieSecure: true,
})
```

生产 GORM store：

```go
if err := gormstore.AutoMigrate(db); err != nil {
    return err
}
store := gormstore.New(db)
```

它只创建 `auth_email_codes`、`auth_email_daily_counts`、
`auth_oauth_flows`、`auth_exchange_codes`、`auth_token_revocations`，不会创建 `users`。

## 有效期与退出

`auth.Deps.TokenTTL` 同时控制邮箱登录、OAuth 换取令牌和刷新；`WSTicketTTL` 控制
WebSocket ticket。未配置时分别保留 7 天、5 分钟默认值。JWT adapter 的 TTL 配置只是
调用方未指定有效期时的后备值，宿主需要把配置同时传入模块的 `Deps`。

需要服务端退出时，将 `jwt.Config.Revocations` 配置为 `gormstore.New(db)`。
`POST /auth/logout` 持久化当前 bearer token 的 SHA-256 和到期时间，不保存原始 token；
之后该 token 不能访问受保护接口或刷新。不同进程使用同一数据库即可共享吊销状态。
每个新 access token 都有独立 `jti`，退出不会误伤同一秒登录的其他会话。

这是单 token 吊销，不是全设备退出：之前通过 refresh 签发的独立 token、已有 WS ticket
和已建立的 WebSocket 连接不会被一并关闭。已经开始执行的请求也不会被撤回。
过期吊销记录在后续退出时清理；如果不再发生退出，记录可保留但不再影响验证。

`TokenSigner` 接口保持兼容。适配器可额外实现 `ContextTokenVerifier`、`TokenRevoker`；
中间件使用请求 context 验证 token。未实现吊销，或吊销存储读写失败时，退出返回 503，
不会假装成功；配置了存储后，验证遇到存储故障也返回 503，拒绝放行。前端应仅在退出
成功或 token 已无效（401）时清除当前凭证，其他错误保留凭证供重试。

## OAuth 浏览器绑定

OAuth 登录使用两层一次性浏览器绑定：provider callback 必须带回签名 state，
并匹配发起浏览器的 `HttpOnly; SameSite=Lax` flow cookie；callback 产生的短期
exchange code 还必须与前端保存在 `sessionStorage` 的随机 verifier 一起提交。
因此复制 callback 或前端 `?code=` URL 到另一浏览器不会登录攻击者账号。

- `GET /auth/:provider/authorize` 必须带 `challenge`（随机 verifier 的 SHA-256，
  unpadded base64url）。返回 JSON `redirect_url`；跨 origin fetch 必须使用
  `credentials: include`。
- 推荐浏览器入口是
  `GET /auth/:provider/authorize?redirect=1&challenge=...`：后端设置 cookie 后直接
  302 到 provider，避免第三方 cookie/CORS 对 flow cookie 的影响。
- `POST /auth/exchange-token` 必须同时提交
  `{"code":"...","oauth_verifier":"..."}`。verifier 只保存在发起 tab 的
  `sessionStorage`，不得放入 URL。

旧客户端如果没有发送 `challenge` 和 `oauth_verifier` 会得到 400，必须升级。
生产 HTTPS callback 必须配置 `OAuthCookieSecure: true`；本地纯 HTTP 才使用 false。

## Provider

```go
providers := map[string]authport.IdentityProvider{
    "google": googleProvider,
    "github": githubProvider,
}
```

不注册某个 Provider 就等于关闭它。只使用 GitHub 时不需要创建 Google Provider。

## 事件

```go
module.Subscribe(authevent.KindUserSignedUp, onUserSignedUp)
module.Subscribe(authevent.KindUserLoggedIn, onUserLoggedIn)
```

进程内总线同步执行监听器，错误会记录但不阻止其他监听器。需要崩溃重放时，在宿主
监听器写 outbox。

完整宿主实现见 `templates/quickstart/internal/user` 和
`templates/quickstart/internal/platform/auth_provider.go`。
