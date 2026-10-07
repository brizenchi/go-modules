# AGENTS.md — templates/quickstart-nextjs (frontend)

Full rules: `docs/standards/FRONTEND.md`. Root `AGENTS.md` still applies.

- All backend calls go through `lib/api.ts` (`apiRequest` or its wrappers); components never call `fetch`.
- Keep request/response types in `lib/api.ts` in sync with the backend in the same change.
- Show users messages derived from `ApiError.status` / `ApiError.reason` and translated with
  `t({ en, zh })`; never render `ApiError.message` directly. Show `ApiError.requestId` for 5xx.
- Logic lives in `lib/` as pure functions with tests in `tests/<module>.test.ts` (`node:test`).
- Environment access only through `lib/env.ts`; `NEXT_PUBLIC_*` values are public — no secrets.
- Session handling only through `lib/auth.ts`.
- No `dangerouslySetInnerHTML` with user content; redirects only to same-site paths.
- Before finishing: `npm run verify` (tests, lint, build, content checks).
