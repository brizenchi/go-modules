# Security

How to report a vulnerability: [SECURITY.md](../../SECURITY.md) in the repository root.

## Input validation

- All external input is untrusted: request bodies, query parameters, headers, webhooks, third-party API responses.
- Validate format, length and range at the boundary; strings have a maximum length, enum values come from an allow-list.
- Validation failures return 400 without echoing internal parser errors.

## Authorization

- Authentication happens once, in a route group or middleware; handlers never parse tokens themselves.
- **Check ownership on every access**: queries by ID include the owner (`WHERE id = ? AND user_id = ?`); hiding a button in the UI is not access control.
- Another user's resource returns 404, not 403.
- Sensitive admin actions write an audit log (who, when, what, values before and after).

## Authentication

- Separate secrets for separate purposes (session signing, OAuth state, webhooks), at least 32 bytes each.
- Passwords are stored with bcrypt, scrypt or argon2; failed logins do not reveal whether the user exists.
- Login and verification-code endpoints are rate-limited and cap failed attempts.
- Signing out invalidates the token.

## Injection

| Kind | Rule |
| --- | --- |
| SQL injection | parameterised queries only; dynamic sort columns come from an allow-list |
| XSS | rely on the framework's escaping; never render user-supplied HTML; escape user input in email templates |
| Command injection | never run shell commands built from user input |
| Path traversal | never use user input as a file path; name uploads with generated IDs |
| **SSRF** | the server calls only configured third-party hosts and **never** arbitrary user-supplied URLs; when unavoidable, check the scheme, reject private addresses and set a timeout |
| Open redirect | redirect targets must be on an allow-list of your own URLs |

## Webhooks

- **Verify the signature before doing anything**; a bad signature returns 400.
- Process idempotently by event ID.
- Do not trust amounts or user IDs in the payload; use the provider's API or your own records.

## Secret management

| Rule | Details |
| --- | --- |
| Where secrets live | the deployment platform's environment variables, a secret manager, or an uncommitted local `.env` |
| Never | in Git, in code or images, in logs, in chat |
| Test data | obviously fake values such as `sk_test_not-a-real-key`, so secret scanners stay quiet |
| Least privilege | third-party tokens get only the scopes they need |
| Isolation | separate secrets per environment and per customer |
| Rotation | when someone leaves, on suspicion of a leak, and yearly |
| Scanning | the gitleaks pre-commit hook and CI's `secrets / gitleaks`; GitHub push protection (enabled by `keel github`) |

### Leaked secrets
1. **Revoke the secret at the provider immediately**, create a new one and update every environment.
   This is the step that matters: rewriting Git history does not un-leak a secret.
2. Check the provider's usage logs for unexpected calls during the exposure.
3. If the commit was not pushed, rewrite it locally; if it was pushed, treat the secret as leaked.
4. Record it and write a postmortem (see [INCIDENT_RESPONSE.md](./INCIDENT_RESPONSE.md)).

## Logging and privacy

- Never log passwords, tokens, verification codes, full card numbers or `Authorization` headers; the logging setup redacts common sensitive fields.
- SQL logs omit parameter values; access logs omit query strings.
- Log personal data only when needed to debug, and never full request or response bodies.

## HTTP security

- Security headers: `Strict-Transport-Security` (set once at the edge), `X-Content-Type-Options: nosniff`, and `X-Frame-Options` or a CSP.
- Cookies are `Secure`, `HttpOnly` and `SameSite`.
- CORS allows only listed origins.
- Uploads: limit size and type, validate the content, store them in object storage, never inside the application directory.

## Dependency security

- CI scans dependencies for vulnerabilities (Go `govulncheck`, Node `audit`, Python `pip-audit`); high-severity findings are handled within a week.
- Dependabot proposes updates weekly; Dependabot security alerts are enabled by `keel github`.
- Before adding a dependency, check that it is maintained, its license allows commercial use, and it has no known vulnerabilities.
