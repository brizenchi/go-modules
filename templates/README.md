# templates

Runnable starter templates for teams standardizing on `go-modules`.

## Choose a template

### Backend

Use [`quickstart`](./quickstart/) when your backend should match the
shared `saascore` model.

Read in this order:

1. [docs/SAASCORE_GUIDE.md](../docs/SAASCORE_GUIDE.md)
2. [quickstart/README.md](./quickstart/README.md)

Verify:

```bash
cd templates/quickstart
go test ./...
go build ./...
```

### Frontend

Use [`quickstart-nextjs`](./quickstart-nextjs/) when you need a
copyable SaaS frontend shell for the shared backend contract.

Already included:

- public home, pricing, and docs routes
- multilingual top navigation
- breadcrumbs and docs table of contents
- login flow and signed-in avatar menu
- account, billing, and referral management pages

Read:

1. [quickstart-nextjs/README.md](./quickstart-nextjs/README.md)

Verify:

```bash
cd templates/quickstart-nextjs
npm ci
npm test
npm run verify
```

## Recommended flow

1. Copy the backend template
2. Copy the frontend template if needed
3. Set env/config values first
4. Verify backend and frontend builds
5. Run the manual auth, billing, and referral checks in the leaf template READMEs

These templates are reference starters, not deployment guarantees. Real
OAuth, Stripe webhook, and referral activation still require a live
runtime environment.
