# Versioning Policy

This is a single-module Go repo. One repo tag versions every package
under `github.com/brizenchi/go-modules`.

## Current package paths

```text
foundation/config
foundation/ginx
foundation/httpresp
foundation/httpx
foundation/jwt
foundation/ossx
foundation/pgx
foundation/randx
foundation/rdx
foundation/resilience
foundation/slog
foundation/tracing
modules/auth
modules/billing
modules/email
modules/referral
modules/user
stacks/saascore
```

Tag format:

```bash
git tag v0.3.0
git push origin v0.3.0
```

Do not create per-package tags such as `modules/auth/v0.3.0`. With a
single root module, Go tooling expects one shared repo tag.

Consumers can pin either the root module or a package path at the same
repo tag:

```bash
go get github.com/brizenchi/go-modules@v0.3.0
go get github.com/brizenchi/go-modules/modules/auth@v0.3.0
```

## SemVer rules

| Change | Bump |
|---|---|
| Bug fix, perf, internal refactor | Patch |
| New API surface, additive config, new helpers | Minor |
| Removed or changed exported API | Major |

## `v0.x` policy

The repo is still pre-`v1.0.0`.

Rule:

- while the repo is on `v0.x`, breaking changes may still happen in a
  minor release
- every breaking change still must be called out in the touched
  package `CHANGELOG.md` files
- once a module reaches `v1.0.0`, normal SemVer compatibility rules
  apply

## `v2+` Go module rule

For `v2+`, the root module path itself must include the major version:

```go
require github.com/brizenchi/go-modules/v2 v2.0.0
```

That means a major release requires:

1. moving the root module to a `v2/` path
2. updating the module path in the root `go.mod`
3. updating imports across packages and consumers, for example
   `github.com/brizenchi/go-modules/v2/foundation/ginx`

Avoid major bumps unless the API break is worth the migration cost.

## Foundation policy

`foundation/*` is the lowest shared API layer. Breaking changes there
fan out into every dependent module.

Rule:

- after `v1.0.0`, breaking foundation changes require at least one minor
  release of deprecation first

Expected flow:

1. add the replacement API
2. mark the old API with `// Deprecated: ...`
3. release a minor version
4. update dependent modules
5. remove the old API in the next major release

## Business and stack policy

`modules/*` and `stacks/*` can evolve faster than foundation, but they
still need:

- scoped CHANGELOG entries
- explicit upgrade notes when host app wiring changes
- coordination inside the shared repo release tag

## Release checklist

```bash
go test ./...

# update touched CHANGELOG.md files

git tag v0.3.0
git push origin v0.3.0
```
