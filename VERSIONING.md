# Versioning Policy

This is a multi-module Go repo. Every module is tagged independently
using its directory path as the tag prefix.

## Current module paths

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
git tag foundation/slog/v0.1.0
git tag modules/auth/v0.1.0
git tag stacks/saascore/v0.1.0
```

Do not create a top-level `vX.Y.Z` tag. In a multi-module repo, Go's
module tooling needs the module-path prefix.

## SemVer rules

| Change | Bump |
|---|---|
| Bug fix, perf, internal refactor | Patch |
| New API surface, additive config, new helpers | Minor |
| Removed or changed exported API | Major |

## `v0.x` policy

Some modules are still pre-`v1.0.0`.

Rule:

- while a module is on `v0.x`, breaking changes may still happen in a
  minor release
- every breaking change still must be called out in that module's
  `CHANGELOG.md`
- once a module reaches `v1.0.0`, normal SemVer compatibility rules
  apply

## `v2+` Go module rule

For `v2+`, the module path itself must include the major version:

```go
require github.com/brizenchi/go-modules/modules/auth/v2 v2.0.0
```

That means a major release requires:

1. moving to a `v2/` subdirectory or equivalent path
2. updating the module path in `go.mod`
3. updating imports inside the module and its consumers

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
- tag names that match the real module path

## Release checklist

```bash
cd modules/auth
go test ./...

# update CHANGELOG.md

git tag modules/auth/v0.1.1
git push origin modules/auth/v0.1.1
```
