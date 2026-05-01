# email

> Portable, provider-agnostic transactional email: Brevo, Resend, SMTP, log/dev fallback.

[![Go Reference](https://pkg.go.dev/badge/github.com/brizenchi/go-modules/modules/email.svg)](https://pkg.go.dev/github.com/brizenchi/go-modules/modules/email)

Never persists user data, never imports project-specific types. Hosts
inject a configured `Sender` and call `SendService.Send(ctx, msg)`.

## Install

```bash
go get github.com/brizenchi/go-modules/modules/email
```

## Layering

```
domain/    pure types: Message, Address, Attachment, errors
port/      interfaces: Sender, Renderer
adapter/   concrete implementations
  brevo/      Brevo HTTP API (server-side templates supported)
  resend/     Resend HTTP API (no server-side templates — use Renderer)
  smtp/       net/smtp (any SMTP server: SES, Postfix, Mailhog, ...)
  log/        no-op logger Sender (dev/tests)
  gotemplate/ Renderer using html/text template
app/       use cases (SendService, SendTemplate)
email.go   Module + multi-tenant Manager
```

## Quick start

### Brevo

```go
import (
    "github.com/brizenchi/go-modules/modules/email"
    "github.com/brizenchi/go-modules/modules/email/adapter/brevo"
    "github.com/brizenchi/go-modules/modules/email/domain"
)

sender, err := brevo.New(brevo.Config{
    APIKey: os.Getenv("BREVO_API_KEY"),
    Sender: domain.Address{Name: "Acme", Email: "no-reply@acme.com"},
})
if err != nil { log.Fatal(err) }

mod := email.New(sender, nil)
_, err = mod.Service.Send(ctx, &domain.Message{
    To:          []domain.Address{{Email: "user@example.com"}},
    Subject:     "Welcome",
    HTMLBody:    "<p>Hi!</p>",
    TemplateRef: "3", // brevo template id
    Variables:   map[string]any{"name": "Bob"},
})
```

### Resend

```go
import "github.com/brizenchi/go-modules/modules/email/adapter/resend"

sender, err := resend.New(resend.Config{
    APIKey: os.Getenv("RESEND_API_KEY"),
    Sender: domain.Address{Name: "Acme", Email: "no-reply@acme.com"},
})
```

Resend has no server-side templates — render locally first (e.g. via
`adapter/gotemplate`) and set `HTMLBody` / `TextBody` directly. Passing
`TemplateRef` returns `ErrTemplateNotFound`.

### SMTP

```go
import "github.com/brizenchi/go-modules/modules/email/adapter/smtp"

sender, err := smtp.New(smtp.Config{
    Host: "smtp.gmail.com", Port: 587,
    Username: "...", Password: "...",
    Sender: domain.Address{Email: "..."},
})
```

### Multi-tenant (one Manager, many Senders)

```go
mgr := email.NewManager()
mgr.Register("default", email.New(brevoSender, nil))
mgr.Register("premium", email.New(resendSender, nil))

mod, _ := mgr.Get("default")
mod.Service.Send(ctx, msg)
```

## Adapter comparison

| Adapter | Network | Server-side templates | Notes |
|---------|---------|-----------------------|-------|
| `brevo`     | HTTP    | ✓ (`TemplateRef` = id) | Cheap, established, EU-friendly |
| `resend`    | HTTP    | ✗ (use `Renderer`)     | Developer-first, React-friendly |
| `smtp`      | SMTP    | ✗                      | Use any SMTP: SES, Postfix, ... |
| `log`       | none    | ✗                      | Dev/CI fallback, prints to slog |

## Testing

```bash
go test -race ./...
```

Coverage: brevo 68.2%, resend 85%, smtp 68.9%, log 100%, app 94.1%.

## Changelog

See [CHANGELOG.md](./CHANGELOG.md).
