# go-modules

Reusable Go modules for shared SaaS backends.

This repo is for teams that do not want every new project to rebuild the
same auth, billing, email, referral, logging, and HTTP wiring from
scratch.

## Repo layout

```text
foundation/   infrastructure-only packages
modules/      reusable business modules
stacks/       opinionated multi-module compositions
templates/    runnable backend/frontend starters
docs/         integration and onboarding guides
```

Rules:

- `foundation/*` has no business concepts.
- `modules/*` has business concepts, but no host-app coupling.
- `stacks/*` is for standard compositions such as a shared SaaS backend.
- `templates/*` is for copyable host shells that prove the stack works.

## Start here

Choose one path:

1. New backend using the shared SaaS model
   Read [docs/SAASCORE_GUIDE.md](./docs/SAASCORE_GUIDE.md)
2. Existing backend adopting one reusable module
   Read [docs/INTEGRATION.md](./docs/INTEGRATION.md), then the module README
3. New frontend for the shared backend contract
   Read [templates/quickstart-nextjs/README.md](./templates/quickstart-nextjs/README.md)
4. Template overview
   Read [templates/README.md](./templates/README.md)
5. Shared config and bootstrap standard
   Read [docs/CONFIG_STANDARD.md](./docs/CONFIG_STANDARD.md)

## Standard paths

### `foundation/*`

Stable project-agnostic helpers:

- `foundation/slog`
- `foundation/jwt`
- `foundation/ginx`
- `foundation/httpresp`
- `foundation/config`
- `foundation/httpx`
- `foundation/tracing`
- `foundation/resilience`
- `foundation/pgx`
- `foundation/rdx`
- `foundation/randx`
- `foundation/ossx`

### `modules/*`

Reusable business modules:

- `modules/auth`
- `modules/billing`
- `modules/email`
- `modules/user`
- `modules/referral`

Planned but not yet shipped as production-ready modules:

- `modules/sms`
- `modules/llm`
- `modules/marketing`

### `stacks/*`

Use `stacks/saascore` when multiple projects intentionally share:

- the same `users` table shape
- the same JWT auth model
- the same Stripe customer/subscription linkage
- the same referral flow

If those assumptions are not true, compose `modules/*` directly instead.

## New project workflow

For the standard shared SaaS shape:

1. Copy `templates/quickstart`
2. Copy `templates/quickstart-nextjs` if you also need the browser shell
3. Follow [docs/SAASCORE_GUIDE.md](./docs/SAASCORE_GUIDE.md)
4. Replace only env/config values and host business hooks

For a custom host shape:

1. Read [docs/INTEGRATION.md](./docs/INTEGRATION.md)
2. Keep your own user table and route tree
3. Implement only the required ports around the module you are adopting

## Local development

This repo uses `go.work` so local modules resolve to local paths during
development.

```bash
make test
make test-race
make tidy-check
make fmt-check
make purity-check
make lint
make vuln
```

## Versioning

This is a multi-module repo. Tags must use the module path prefix:

```bash
git tag foundation/slog/v1.0.0
git tag modules/auth/v1.0.0
git tag stacks/saascore/v1.0.0
```

Read [VERSIONING.md](./VERSIONING.md) before publishing tags or making
breaking changes.

## Contributing

Read [CONTRIBUTING.md](./CONTRIBUTING.md).
