# Versioning policy

This is a multi-module Go repo. Each module is versioned independently
via tag prefixes.

## Tags

```
foundation/slog/v1.0.0
foundation/jwt/v1.0.0
foundation/httpresp/v1.0.0
foundation/ginx/v1.0.0
foundation/config/v1.0.0
foundation/pgx/v1.0.0
foundation/rdx/v1.0.0
auth/v1.0.0
billing/v1.0.0
email/v1.0.0
referral/v1.0.0
```

`git tag <module>/vX.Y.Z` (matches the directory path). DO NOT use a
top-level `vX.Y.Z` tag — Go's module proxy treats it as ambiguous in a
multi-module repo.

## SemVer

| Change | Bump | Example |
|---|---|---|
| Bug fix, perf, internal refactor | Patch | `v1.0.0 → v1.0.1` |
| New API surface (additive) | Minor | `v1.0.0 → v1.1.0` |
| Removing/renaming exported names, changing signatures | Major | `v1.0.0 → v2.0.0` |

## Major version requirements (Go modules)

For v2+, the module path itself MUST include the major:

```
require github.com/brizenchi/go-modules/auth/v2 v2.0.0
```

That means major bumps require:
1. Renaming the directory: `auth/` → `auth/v2/` (or use a v2 subdirectory).
2. Updating the `module` line in go.mod.
3. Updating import paths inside the module.

Avoid this if possible — prefer additive minor releases.

## Foundation policy

Foundation modules are public API for everything that depends on them.
Breaking changes in foundation cascade to every business module.

Rule: **breaking changes in foundation require a deprecation period of
at least one minor version.**

Process:
1. Add the new API; mark the old one with `// Deprecated: ...` comment.
2. Release a minor (e.g. `foundation/slog/v1.2.0`).
3. Update business modules to use the new API.
4. After at least one release, remove the old API in the next major.

## Business module policy

Business modules can release majors more freely. Each business module
is consumed by exactly the projects that opt-in, not by other modules
in this repo, so blast radius is bounded.

## Release process

```bash
# 1. Make sure tests pass for the module:
cd auth && go test ./...

# 2. Update CHANGELOG.md inside the module.

# 3. Tag from the main branch:
git tag auth/v1.1.0
git push origin auth/v1.1.0

# 4. The Go module proxy picks it up within minutes.
```
