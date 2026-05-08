# SaaSCore Guide

Use this guide when starting a new backend that should match the
standard shared SaaS shape.

## Use `stacks/saascore` when

Your projects intentionally share:

- the same `users` table shape
- the same JWT auth model
- the same billing + referral lifecycle
- the same referral registration and activation flow

If those assumptions are not true, do not force `saascore`. Use the
lower-level `modules/*` packages directly instead.

## What you get

`stacks/saascore` already composes:

- `modules/user`
- `modules/auth`
- `modules/email`
- `modules/billing`
- `modules/referral`

It also already wires:

- signup -> shared user creation/linking
- signup -> free-plan initialization
- optional `referral_code` attribution
- billing events -> billing-owned state sync
- billing events -> user summary projection sync
- Stripe subscription activation -> referral activation
- shared JWT middleware for billing and referral user routes

## Current billing boundary

The platform boundary is transitioning to:

- identity and auth summary in `users`
- provider linkage in `billing_customers`
- current commercial snapshot in `billing_subscriptions`
- webhook audit/idempotency in `billing_events`

`stacks/saascore` still keeps legacy billing columns on `users`
up-to-date for compatibility, but new integrations should treat those
columns as deprecated projections rather than source-of-truth billing
state.

## New backend checklist

1. Copy `templates/quickstart`
2. Initialize your own module
   - `go mod init github.com/yourname/yournewservice`
   - `go get github.com/brizenchi/go-modules@latest`
   - `go mod tidy`
3. Seed config files
   - `cp .env.example .env`
   - `cp deploy/config.yaml.example deploy/config.yaml`
4. Set the minimum required config
   - DB config
   - `auth.user_jwt_secret`
5. Verify structure
   - `go test ./...`
   - `go build ./...`
6. Run locally
   - `set -a; source .env; set +a`
   - `go run ./cmd/quickstart`
7. Replace only host-specific business hooks

## What you should change

- project/module path
- env/config values
- reward and entitlement hooks
- branding and product-specific routes

## What you should not rewrite

- JWT signing and verification
- email-code login flow
- Google OAuth flow
- Stripe checkout/webhook parsing
- billing event idempotency
- referral repositories and HTTP handlers

## Minimum config to boot

Backend requires at minimum:

- `db.host`
- `db.user`
- `db.password`
- `db.name`
- `auth.user_jwt_secret`

Optional integrations:

- email
  - `email.provider=brevo|resend`
  - `email.brevo.api_key`
  - `email.brevo.sender_email`
  - `email.resend.api_key`
  - `email.resend.sender_email`
- Google OAuth
  - `auth.google.client_id`
  - `auth.google.client_secret`
  - `auth.google.redirect_url`
  - `auth.google.state_secret`
- Stripe
  - `billing.stripe.secret_key`
  - `billing.stripe.webhook_secret`

## Local verification before handoff

Verify all of these before telling another team the starter is ready:

1. `go test ./...`
2. `go build ./...`
3. email-code login works
4. Google OAuth works when configured
5. JWT-protected routes reject missing bearer tokens
6. Stripe webhook reaches the backend from a public tunnel or domain
7. referral signup with `?ref=` activates after paid subscription

## Production checklist

- set a strong `auth.user_jwt_secret`
- set real email sender identity
- set public backend OAuth callback URLs
- set public frontend success/cancel/login/invite URLs
- verify the Stripe webhook secret in the deployed environment
- keep host reward and entitlement hooks idempotent
