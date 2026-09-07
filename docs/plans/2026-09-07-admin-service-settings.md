# Admin service settings

The website owner can configure Resend and Stripe at `/admin/integrations` without editing provider environment variables. This is separate from the customer workspace and public brand settings.

## Scope and decisions

- Store provider credentials as plaintext in the database, as explicitly requested. Never return secret values through the API, persist them in browser storage, or include them in SQL/application/audit logs.
- Keep database connection, administrator login, JWT, domains and infrastructure configuration in deployment environment variables.
- Each provider can use database settings or fall back to its original deployment configuration. Database settings take precedence only for that provider.
- Save with administrator authorization, bounded input, field allowlists, version checks and audited idempotent updates. Omitted secrets retain their effective values; explicit clearing requires a disabled provider.
- Existing modules bind configuration at startup. Save for the next backend restart, showing saved source, current source, current enabled state and pending restart. No automatic process restart or request-time provider replacement.
- The isolated local preview permits form persistence tests but never activates external mail/payment services. Its data is temporary.
- Validate configuration structure and completeness locally. Saving does not verify a key with the remote provider or send email/create purchases.

## Implementation

1. Add a host-owned service configuration store, safe status views, overrides and migration artifact.
2. Register administrator GET/PATCH endpoints and audit updates without secret values.
3. Split initial deployment validation from effective configuration validation. Load database overrides before module migration/construction, so database-enabled billing receives its tables and event listeners.
4. Add bilingual admin navigation/forms with retained-key semantics, version-conflict handling, restart status and environment fallback.
5. Update setup documentation and run temporary SQLite tests, frontend tests/lint/build and local browser verification.

No migrations or repair jobs will be executed against the configured database. No deployment, push, real email or payment is part of this task.

## Verification completed

- Frontend: 109 tests passed, lint and production build passed (35 generated pages). The final bundle includes the narrow-screen header and distinct credential-clear labels.
- Go race tests passed for serviceconfig, operations, bootstrap, preview and HTTP middleware. Coverage includes database provider overrides before module creation, effective deployment validation, restricted Stripe key formats, plaintext persistence without secret responses/logs, permission rejection, version/idempotency handling, restart status and local preview isolation.
- Browser: administrator login; Resend save, blank-key retention and environment reset; Stripe missing-field errors and successful test-configuration save; password fields clear after save; Chinese desktop/375px and English 320px layouts, with no horizontal page overflow.
- Only generated dummy credentials were used in the temporary preview. The preview was restarted after tests to remove those temporary settings, and the final Chinese admin page was left open. No configured database, actual email or payment service was used.
