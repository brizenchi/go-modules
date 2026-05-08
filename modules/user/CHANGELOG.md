# Changelog

## Unreleased

- document the shift toward billing-owned persistence tables
- mark `adapter/billingstore` as a deprecated compatibility layer
- clarify that legacy Stripe/billing columns on `users` are transitional
  projection fields, not the long-term billing source of truth

## v0.1.0

- add standard reusable `users` module
- add GORM-backed standard user schema and repository helpers
- add auth adapters for `auth.port.UserStore` and `RoleResolver`
- add billing adapters for `billing.port.CustomerStore` and `UserResolver`
- add subscription summary sync helpers for standard user fields
