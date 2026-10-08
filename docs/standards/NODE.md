# Node / TypeScript

General rules: [CODE_STYLE.md](./CODE_STYLE.md).

## Tools

| Tool | Checks | Where it runs |
| --- | --- | --- |
| TypeScript `strict` | types | the `build` or `typecheck` script |
| ESLint | static analysis | the `lint` script, CI `node-* / check` |
| Prettier (if the project uses it) | formatting | editor, the AI assistant's format hook |
| Test runner | unit tests | the `test` script, CI `node-* / check` |
| Dependency audit | vulnerable production dependencies | CI `node-* / audit` (does not block merging) |

CI runs the `package.json` scripts configured in `node_projects` in `.keel/answers.yml`, in
order, skipping scripts that do not exist. The package manager is detected from the lock
file (`package-lock.json`, `pnpm-lock.yaml`, `yarn.lock`); **the lock file must be committed**.

## Code

- TypeScript strict mode; no `any` — use `unknown` and narrow it.
- Logic that does not depend on React or another framework lives in plain modules as pure functions, so it can be tested on its own.
- File names are kebab-case: `sign-in-panel.tsx`.
- User-facing text is kept in one place; do not hard-code it in components when the product is translated.

## Calling the backend

- **Every backend request goes through one shared client module**; components never call `fetch` directly.
- The client builds URLs, attaches credentials, unwraps the response shape, handles 401, and throws an
  error carrying `status`, `reason` (the stable error code) and `requestId` (the `X-Request-ID` response header).
- Request and response types stay in sync with the backend and change in the same pull request as the endpoint (see [API_STANDARD.md](./API_STANDARD.md#api-contracts)).

### Showing errors
- What users see is derived from `status` or `reason`; **never show the server's error message**.
- **Show the request ID for 5xx errors**, so users can report it and developers can find it in logs and traces.
- Network errors say the server cannot be reached and offer a retry; 401 leads to sign-in; 403 says access is denied.

### Writes
- Disable the submit button while a request is in flight.
- Operations that must be idempotent generate an `Idempotency-Key` and **reuse it on retry**.
- Never retry non-idempotent writes automatically.

## Authentication and environment variables

- Session state is read and written in one module; tokens never appear in URLs, logs or error reports.
- Variables readable in the browser (`NEXT_PUBLIC_*`, `VITE_*`) **never contain secrets**.
- Environment variables are read and validated in one module; components never read `process.env`.

## Security

- Never render user input with `dangerouslySetInnerHTML`.
- Redirects after sign-in or payment allow only paths on your own site.
- External links use `rel="noopener noreferrer"`.

## Accessibility and performance

- Form controls have an associated `label`; error messages use `role="alert"`; clickable elements are `button` or `a`.
- Images have `alt` text; public pages render on the server, with client components only where interaction needs them.

## Testing

- Pure logic — the API client, state calculations, formatting — must be tested.
- Replace `fetch` when testing request logic; never call the real backend.
