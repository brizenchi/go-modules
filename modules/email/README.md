# email 邮件模块

提供服务商无关的事务邮件发送能力。

## 适配器

- `adapter/resend`
- `adapter/brevo`
- `adapter/smtp`
- `adapter/log`：本地开发，只记录邮件
- `adapter/gotemplate`：本地 Go 模板渲染

## 组装

```go
sender, err := resend.New(resend.Config{
    APIKey: "re_...",
    Sender: emaildomain.Address{
        Email: "no-reply@example.com",
        Name:  "My SaaS",
    },
})
if err != nil {
    return err
}

module := email.New(sender, nil)
```

发送内联内容：

```go
_, err := module.Send(ctx, &emaildomain.Message{
    To:       []emaildomain.Address{{Email: "user@example.com"}},
    Subject:  "欢迎加入",
    TextBody: "你的账号已经创建成功。",
})
```

## 边界

email 不知道用户、注册、支付或邀请。下面这些决定属于宿主：

- 注册成功后是否发送欢迎邮件；
- 哪种登录方式触发邮件；
- 使用哪个模板和语言；
- 失败后是否进入重试队列。

quickstart 在 `internal/platform/email_provider.go` 选择发送器，在
`internal/bootstrap/host_hooks.go` 决定什么时候发送。

## 多租户

同一进程确实需要多套发信凭据时使用 `email.Manager`，按项目键注册不同 Module。
十个独立部署的 SaaS 通常各自只需要一个 Module。
