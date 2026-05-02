# saascore

> Standard shared SaaS backend composition for projects that reuse the same user/auth/billing/referral model.

[![Go Reference](https://pkg.go.dev/badge/github.com/brizenchi/go-modules/stacks/saascore.svg)](https://pkg.go.dev/github.com/brizenchi/go-modules/stacks/saascore)

Use this stack when multiple projects intentionally share:

- the same `users` table shape
- the same auth model
- the same Stripe customer/subscription linkage
- the same referral attribution and activation flow

This is the canonical "full backend starter" layer in `go-modules`.
Business repositories should consume this package rather than re-creating
the same glue.

## What it includes

- `modules/user` schema migration
- `modules/auth` wired to the shared user repo
- email-code auth with Brevo provider templates or Resend/local fallback delivery
- optional Google OAuth
- `modules/billing` wired to the shared user repo
- billing webhook idempotency persistence
- `modules/referral` wiring
- standard cross-module listeners for signup, billing sync, and referral activation
- standard Gin route mounting helpers

## What stays in the host app

- env loading and config file ownership
- router root and infrastructure middleware
- reward payout semantics
- app-specific routes and domain logic
- any non-standard user fields or role policy

## Install

```bash
go get github.com/brizenchi/go-modules/stacks/saascore
```

## Quick start

```go
stack, err := saascore.New(
    db,
    cfg,
    saascore.HostHooks{
        OnReferralActivated: func(ctx context.Context, ev saascore.ReferralActivatedEvent) error {
            return usersRepo.AddCredits(ctx, ev.Referral.ReferrerID, ev.Referral.RewardCredits)
        },
    },
    saascore.PolicyHooks{
        ResolveReferralReward: func(ctx context.Context, in saascore.ReferralRewardPolicyInput) (int, error) {
            return 50, nil
        },
    },
)
if err != nil {
	log.Fatal(err)
}

publicGroup := r.Group("/api/v1")
userGroup := r.Group("/api/v1")
userGroup.Use(stack.RequireUser())
stack.Mount(publicGroup, userGroup)
```

## Extension model

- `HostHooks`
  - host business callbacks for standard lifecycle events
- `PolicyHooks`
  - host-controlled business rules that the shared flow needs at runtime

Typical use:

- `OnUserSignedUp`
  - create workspace, team, tenant, wallet
- `OnSubscriptionActivated`
  - grant project-specific quota or entitlement
- `OnReferralActivated`
  - perform the actual reward payout
- `ResolveReferralReward`
  - decide the reward amount per project

See [`templates/quickstart`](../../templates/quickstart/) for the
complete runnable host shell and [`docs/INTEGRATION.md`](../../docs/INTEGRATION.md)
for the integration contract.

For a full project-onboarding walkthrough, config checklist, host hook
rules, and frontend pairing guidance, read
[`docs/SAASCORE_GUIDE.md`](../../docs/SAASCORE_GUIDE.md).
