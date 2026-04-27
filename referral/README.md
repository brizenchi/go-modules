# referral

> Portable, schema-owning C2C referral module: code generation, attribution, activation events.

[![Go Reference](https://pkg.go.dev/badge/github.com/brizenchi/go-modules/referral.svg)](https://pkg.go.dev/github.com/brizenchi/go-modules/referral)

Owns its own database schema (`referral_codes`, `referrals`) — auto-migrated
at boot. Hosts wire it to their auth + billing flows via two integration
calls and an event listener.

For affiliate / B2B partner programs, prefer
[Rewardful](https://rewardful.com/) (front-end script + Stripe metadata).
This module is for **user-to-user** "share with a friend" referrals.

## Install

```bash
go get github.com/brizenchi/go-modules/referral
```

## Layering

```
domain/   Code, Referral, Status, Stats, errors
event/    ReferralRegistered, ReferralActivated
port/     CodeRepository, ReferralRepository, CodeGenerator, EventBus
adapter/
  gormrepo/   GORM impl + AutoMigrateModels()
  codegen/    deterministic + random CodeGenerator
  eventbus/   in-process synchronous bus
app/      CodeService, AttributeService, QueryService
http/     Gin handlers + Mount()
```

## Quick start

```go
import (
    "github.com/brizenchi/go-modules/referral"
    "github.com/brizenchi/go-modules/referral/adapter/codegen"
    "github.com/brizenchi/go-modules/referral/adapter/eventbus"
    "github.com/brizenchi/go-modules/referral/adapter/gormrepo"
)

// Auto-migrate the module's tables.
if err := gormrepo.AutoMigrateModels(db); err != nil { log.Fatal(err) }

mod := referral.New(referral.Deps{
    Codes:     gormrepo.NewCodeRepo(db),
    Referrals: gormrepo.NewReferralRepo(db),
    CodeGen:   codegen.NewRandom(8),  // 8-char codes
    Bus:       eventbus.New(),
    GetUserID: myGinUserIDExtractor,
    CodeTTL:   0, // 0 = no expiry
})

user := r.Group("/api/v1", requireAuth)
user.GET("/referral/code", mod.Handler.GetMyCode)
user.GET("/referral/list", mod.Handler.ListMyReferrals)
user.GET("/referral/stats", mod.Handler.GetMyStats)
```

## Two integration calls (host wires)

```go
// 1. After signup — referee supplied a code at registration
authMod.Subscribe(authevent.KindUserSignedUp, func(ctx context.Context, ev authevent.Event) error {
    if ev.ReferralCode == "" { return nil }
    return referralMod.Attribute.AttributeReferral(ctx, ev.UserID, ev.ReferralCode)
})

// 2. After "qualified" event — typically subscription activation
billingMod.Subscribe(billingevent.KindSubscriptionActivated, func(ctx context.Context, ev billingevent.Event) error {
    return referralMod.Attribute.ActivateReferral(ctx, ev.UserID, /*reward credits*/ 100)
})
```

## State machine

```
Code: ACTIVE → REVOKED
Referral: PENDING → ATTRIBUTED → ACTIVATED
                                ↓
                              REJECTED (refund / fraud)
```

The `ReferralActivated` event is your reward hook — listen and grant
credits / discount / extension to either or both parties.

## Testing

```bash
go test -race ./...
```

Coverage: app 78%, codegen 84%, eventbus 83.3%.

## Changelog

See [CHANGELOG.md](./CHANGELOG.md).
