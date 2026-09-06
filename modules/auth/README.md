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
`auth_oauth_flows`、`auth_exchange_codes`，不会创建 `users`。

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
