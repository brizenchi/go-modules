# referral

> Portable, schema-owning C2C referral module: code generation, attribution, activation events.

[![Go Reference](https://pkg.go.dev/badge/github.com/brizenchi/go-modules/modules/referral.svg)](https://pkg.go.dev/github.com/brizenchi/go-modules/modules/referral)

Owns its own database schema (`referral_codes`, `referrals`) — auto-migrated
at boot. Hosts wire it to their auth + billing flows via two integration
calls and an event listener.

For affiliate / B2B partner programs, prefer
[Rewardful](https://rewardful.com/) (front-end script + Stripe metadata).
This module is for **user-to-user** "share with a friend" referrals.

## Install

```bash
go get github.com/brizenchi/go-modules/modules/referral
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
    "context"

    billingevent "github.com/brizenchi/go-modules/modules/billing/event"
    referraleventbus "github.com/brizenchi/go-modules/modules/referral/adapter/eventbus"
    referralhttp "github.com/brizenchi/go-modules/modules/referral/http"
    "github.com/brizenchi/go-modules/modules/referral"
    "github.com/brizenchi/go-modules/modules/referral/adapter/codegen"
    "github.com/brizenchi/go-modules/modules/referral/adapter/gormrepo"
    "github.com/brizenchi/go-modules/modules/referral/event"
)

// Auto-migrate the module's tables.
if err := db.AutoMigrate(gormrepo.AutoMigrateModels()...); err != nil {
    log.Fatal(err)
}

mod := referral.New(referral.Deps{
    Codes:     gormrepo.NewCodeRepo(db),
    Referrals: gormrepo.NewReferralRepo(db),
    Generator: codegen.NewRandom("INV", 8),
    Bus:       referraleventbus.NewInProc(),
    GetUserID: myGinUserIDExtractor,
    BaseLink:  "https://app.example.com/invite?ref=",
})

user := r.Group("/api/v1", requireAuth)

referralhttp.Mount(mod.Handler, user)
```

## Two integration calls (host wires)

```go
// 1. After signup — only if your host registration flow captured a referral code.
func onUserSignedUp(ctx context.Context, userID, referralCode string) error {
    if referralCode == "" {
        return nil
    }
    _, err := referralMod.Attribute.AttributeReferral(ctx, userID, referralCode)
    return err
}

// 2. After the first qualifying paid event — typically subscription activation.
billingMod.Subscribe(billingevent.KindSubscriptionActivated, func(ctx context.Context, env billingevent.Envelope) error {
    _, err := referralMod.Attribute.ActivateReferral(ctx, env.UserID, 100)
    return err
})

// 3. Reward payout hook.
referralMod.Subscribe(event.KindReferralActivated, func(ctx context.Context, env event.Envelope) error {
    activated, _ := env.Payload.(event.ReferralActivated)
    return creditWallet(ctx, activated.Referral.ReferrerID, activated.Referral.RewardCredits)
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
