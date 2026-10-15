# Customer console simplification

- Exactly three tabs in the administrator's customer dialog: Overview, Payment records, Credit changes. Overview shows registration time, effective membership, available credits and compact subscription facts. Only explicit Adjust membership / Adjust credits actions open an inline editor; records tabs are read-only. Existing membership/credit services, audit, validation and stable retry keys remain in use.
- Fix sibling reconciliation keys: the former credit editor and history shared the same key and could leave duplicate/wrong forms after tab changes. Regression must exercise repeated switches in a real React DOM.
- Credit changes are one ledger, with translated source, signed amount, historical balance, reason, operator and correlation reference. Referral, signup, top-up, recurring allowance and campaign sources are supported without creating another reward tab or fabricating unimplemented grants.
- Use one native accessible Select component for admin filters and forms: consistent height, padding, chevron, focus and disabled states, with mobile sizing. No custom combobox dependency.
- Persist newly opened checkout sessions through an optional additive billing observer, wired only by the template composition root. Store a safe allowlisted snapshot, not URLs/keys/request bodies. Merge with verified provider events for administrator history; unpaid intent must never count as paid or alter customer purchase/entitlement behavior. Retries/session reuse are deduplicated. Historical abandoned sessions that were never recorded cannot be reconstructed.
- Tests use disposable databases, fake providers and offline browser fixtures. No existing service restart, configured database migration, production write, commit, push or deployment.

## Verification

- Reproduced the former editor remaining under Invitations in a real React DOM before the fix; repeated tab switches now pass with one explicit editor and unique IDs.
- `scripts/check-customer-dialog.mjs` renders the real dialog, table, overview and Select in isolated offline Chrome: 20 English/Chinese, desktop/mobile fixtures pass. Screenshots were inspected for filters, overview, adjustment form and unified ledger.
- Frontend `npm run verify`: 198 tests, lint, production build and 72 public-content checks passed. Backend template race/vet, shared-module race/vet/purity/lint, and detached template-copy verification passed. Provider metadata tests cover order amount, environment and original creation/expiry times.
- Checkout history tests cover unpaid visibility, exact customer isolation, checkout/invoice deduplication, verified-only customer/reporting projection, and provider-confirmed expiry without downgrading a paid order. All database tests use disposable SQLite fixtures.
