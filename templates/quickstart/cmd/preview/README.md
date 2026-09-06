# Local acceptance preview

Run from `templates/quickstart`:

```sh
export APP_AUTH_ADMIN_PASSWORD='replace-with-a-local-test-password'
go run ./cmd/preview --port 18081 --frontend http://localhost:3100
```

Set `NEXT_PUBLIC_API_BASE_URL=http://127.0.0.1:18081/api/v1` and
`NEXT_PUBLIC_APP_URL=http://localhost:3100` when starting the frontend development
server (or before its production build). Start that frontend on port 3100.
The command always
binds to `127.0.0.1`; it accepts only a loopback HTTP frontend origin.

Sign in:

- Administrator: open `/admin`, enter `preview-admin@example.test` and the password
  you supplied in `APP_AUTH_ADMIN_PASSWORD` (12–72 bytes). There is no default password.
- Ordinary user: `preview-user@example.test`, through the ordinary email-code dialog.

The preview admin address is fixed; `APP_AUTH_ADMIN_EMAIL` does not change it.
Without `APP_AUTH_ADMIN_PASSWORD`, password login is disabled. Email-code login
remains available for both fixture accounts through the ordinary sign-in dialog.

Both start with 50 credits and a sample note. Email codes appear in the terminal
and in the API's debug response. Additional addresses can register locally and
receive 50 signup credits. Invite a fresh address to verify signup attribution;
existing accounts cannot attach an invitation on a later login.

This is a **throwaway local fixture**, not a deployment mode. It constructs all
provider configuration in code and never reads `.env`, deployment YAML, configured DB
connections, payment keys, email credentials, or object-storage credentials.
The only environment variable it consumes is `APP_AUTH_ADMIN_PASSWORD` for the
local administrator login; its value is never printed.
The real authentication, referral, account, notes, credits, settings, operator,
and private upload handlers run against a newly created temporary SQLite file.
Email uses the log sender; OAuth and billing are disabled. No payment provider
or email network request is made. Subscription/order screens therefore show
their disabled/empty states; this does not verify a Stripe payment integration.

Each run generates a new JWT secret. Ctrl+C shuts down the server, closes SQLite,
and removes the temporary database and private uploads. Force-killing the process
may leave its clearly named `quickstart-local-preview-*` temporary directory.
