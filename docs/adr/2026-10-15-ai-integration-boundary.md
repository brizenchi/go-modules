# AI integration belongs to infrastructure; conversations belong to products

Status: accepted by the user on 2026-10-15.

The template is presented to site owners. Its customer account area supplies common account, subscription, credit and referral services. A model connection must not implicitly create a consumer chat product.

Decision: extract the existing compatible completion transport into `foundation/aix`, with a small backend-template configuration factory. It accepts explicit messages and returns a completion; it has no user identity, database, conversation history, quota or credit policy. Do not register a default public AI capability endpoint or authenticated chat endpoint. Remove the default chat page and its browser client. Products inject the provider into their own features and define authorization, conversation storage, charging and retries there.

Retain notes/export and image-upload code as clearly labelled examples, reachable from the public examples directory and their guides. Exclude them from the default account overview. Keep the public template website and independent administrator interface. Recapture homepage screenshots after this change.

The alternative of hiding the chat link would leave an implicit billable public API and product-specific persistence behind. Deleting AI entirely would discard the requested integration. A stateless shared adapter keeps the reusable part with little operational cost. It defaults off, requires the injected HTTP client, propagates context, refuses redirects and never returns upstream bodies in errors. It performs no retries of its own.

No data is deleted and no applied migration is edited. Previously created demo chat tables are left untouched; they are no longer migrated or used. No new dependency or Go module is needed.

Verification: failing route/navigation regressions first; fake-transport tests for request validation, cancellation, response limits and safe failures; backend and shared-package race/vet/purity checks; frontend verify and browser checks in both languages and on mobile.

Completed validation: root `make fmt`, `make test-race`, `make vet`, `make purity-check` and golangci-lint passed. Backend `go test -race ./...`, `go vet ./...` and golangci-lint passed. Frontend isolated `npm run verify` passed (170 tests; 76 public pages), followed by a successful rebuild/content check after replacing the screenshot assets. Browser checks confirmed the bilingual account services, read-only account dialog and the public homepage's AI/example boundaries. Provider tests use an injected fake transport; no real model call was performed.
