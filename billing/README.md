# billing

> Portable, payment-provider-agnostic billing: Stripe-backed checkout, subscriptions, credits, webhooks.

[![Go Reference](https://pkg.go.dev/badge/github.com/brizenchi/go-modules/billing.svg)](https://pkg.go.dev/github.com/brizenchi/go-modules/billing)

The module is host-agnostic — never imports project-specific user/order
models. Hosts integrate via three pluggable points.

## Install

```bash
go get github.com/brizenchi/go-modules/billing
```

## Layering

```
domain/   pure types: enums, errors, snapshots, persistence model,
          ReservedMetadataKeys
event/    domain events (subscription.activated, ...)
port/     interfaces (Provider, EventBus, Repository, CustomerStore,
          UserResolver)
adapter/  concrete implementations
  stripe/      Stripe checkout + webhooks (stripe-go/v76)
  repo/        GORM BillingEvent repository (idempotency)
  eventbus/    in-process synchronous bus
app/      use cases (Checkout, Cancel, Reactivate, Webhook, Query)
http/     Gin handlers + Mount()
```

## Host responsibilities

1. **`port.CustomerStore`** — load/save the provider customer ID against the host's user table
2. **`port.UserResolver`** — resolve `userID` from webhook hints (email / customer / subscription IDs)
3. **Event listeners** — react to `SubscriptionActivated`, `CreditsPurchased`, etc. (grant quota, send email, ...)

## Quick start

```go
import (
    "github.com/brizenchi/go-modules/billing"
    "github.com/brizenchi/go-modules/billing/adapter/eventbus"
    "github.com/brizenchi/go-modules/billing/adapter/repo"
    "github.com/brizenchi/go-modules/billing/adapter/stripe"
)

provider := stripe.NewProvider(stripe.Config{
    Enabled:        true,
    SecretKey:      os.Getenv("STRIPE_SECRET_KEY"),
    WebhookSecret:  os.Getenv("STRIPE_WEBHOOK_SECRET"),
    SubscriptionPrices: map[domain.PlanType]map[domain.BillingInterval]string{
        domain.PlanStarter: {domain.IntervalMonthly: "price_starter_m"},
        domain.PlanPro:     {domain.IntervalMonthly: "price_pro_m"},
    },
    CreditsPriceIDs: []string{"price_credits_a"},
    CreditsPerUnit:  40,
})

mod := billing.New(billing.Deps{
    Provider:     provider,
    Bus:          eventbus.New(),
    Customers:    myCustomerStore,             // your impl
    EventRepo:    repo.NewBillingEventRepo(db),
    UserResolver: myUserResolver,              // your impl
    GetUserID:    myGinUserIDExtractor,        // ties auth to billing routes
})

// Public webhook endpoint (no auth!)
r.POST("/stripe/webhook", mod.Handler.HandleWebhook)

// Authenticated routes
user := r.Group("/api/v1", requireAuth)
user.POST("/stripe/checkout/session", mod.Handler.CreateCheckoutSession)
user.POST("/stripe/subscription/cancel", mod.Handler.CancelSubscription)
user.POST("/stripe/subscription/reactivate", mod.Handler.ReactivateSubscription)
user.GET("/stripe/subscription", mod.Handler.GetSubscription)
user.GET("/stripe/invoices", mod.Handler.ListInvoices)
```

## Checkout metadata pass-through

Callers can attach metadata to the Checkout Session via the request body:

```js
fetch('/api/v1/stripe/checkout/session', {
  body: JSON.stringify({
    plan: 'pro', interval: 'monthly',
    success_url: '...', cancel_url: '...',
    metadata: { referral: window.Rewardful?.referral }  // Rewardful
  })
})
```

Reserved keys (`user_id`, `email`, `plan`, `interval`, `product_type`,
`price_id`, `quantity`) are written by the billing layer itself and
**always win** over caller metadata — frontend can't spoof them. See
`domain.ReservedMetadataKeys` and `domain.IsReservedMetadataKey`.

## Idempotent webhooks

`adapter/repo` persists every webhook event id; replays are detected and
short-circuited before listeners run. Listeners can therefore assume
each delivery fires at most once per process.

## Testing

```bash
go test -race ./...
```

Coverage: stripe 76.1%, eventbus 100%, app 61.7%, domain 75%.

## Changelog

See [CHANGELOG.md](./CHANGELOG.md).
