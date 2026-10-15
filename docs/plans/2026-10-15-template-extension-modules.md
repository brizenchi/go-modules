# Template Extension Modules Implementation Plan

> Boundary correction (2026-10-15): AI is now a stateless shared adapter without a default chat product. The accepted [AI integration decision](../adr/2026-10-15-ai-integration-boundary.md) supersedes the chat UI/API/storage portions of this earlier plan. Notes and uploads remain developer examples outside the default account center.

**Goal:** Add independently implemented reusable table/form components, optional analytics/support adapters, and a scoped AI extension, excluding new payment providers.

**Architecture:** Share typed React primitives and small tested logic modules. Use documented provider contracts, never the reference project's source code. External integrations default off and use explicit public IDs; provider secrets remain server-only. Preserve all pending customer UI changes.

1. Implement DataTable, pagination and sorting state helpers; migrate credit history.
2. Implement FormField and submit action feedback; migrate the note editor.
3. Implement optional page analytics and on-demand isolated support chat, with safe URL/ID validation and bilingual docs.
4. Resolve AI scope; implement only the agreed/default capability with injected HTTP, auth, cancellation and tests.
5. Verify changed backend/frontend packages; inspect actual desktop/mobile views. No commit, push, or deployment.
