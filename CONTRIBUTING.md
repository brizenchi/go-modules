# Contributing to go-modules

Thanks for your interest. This monorepo holds the org's reusable Go
packages. Every change should hold the line on three things: **purity**
(no project-specific imports leak in), **stability** (additive changes
within a major), and **discoverability** (each package's README +
CHANGELOG stays accurate).

## Repo layout

```
foundation/<name>/   pure infra: stdlib + one-or-two common libs only
modules/<name>/      DDD-layered business packages
docs/                host integration and migration guides
templates/quickstart/ runnable boot example consuming all modules
scripts/             one-off helpers (extraction, migration)
```

This repo is one Go module rooted at
`github.com/brizenchi/go-modules`. Packages live under
`foundation/*`, `modules/*`, `stacks/*`, and `templates/quickstart`.
They ship together under one repo tag such as `v0.3.0`.

## Local setup

```bash
git clone git@github.com:brizenchi/go-modules.git
cd go-modules
make test          # all modules, no -race
make test-race     # all modules with -race
make purity-check  # ensure no host-app imports leaked
make fmt           # gofmt the repo
make tidy          # go mod tidy at the repo root
```

Because the repo is a single Go module, cross-package changes work
without `replace` directives — edit `modules/auth` and
`modules/billing` together and root `go test` sees the changes.

## Design principles

### 1. Purity

`pkg/<module>/` MUST NOT import any project-specific package. Allowed
deps: stdlib + a small set of common libraries (`gin`, `gorm`,
`stripe-go/v76`, `golang-jwt/v5`, `go-redis/v9`, `viper`, `mapstructure`).

Project-specific glue (e.g. wiring viper config to a package's `Config`
struct, mapping the host's user table to `port.UserStore`) lives in the
**host project**, not here. The reference bootstrap wiring lives in
`templates/quickstart/cmd/quickstart`.

`make purity-check` enforces this — CI fails on violations.

### 2. Ports + adapters

Business packages expose a `port/` package with the interfaces they
depend on, and one or more `adapter/` implementations. Adding a new
provider for an existing package = new adapter + tests. **Don't**
modify the port to fit a single provider's quirks — use an adapter
or extend `domain/` if the concept is universal.

### 3. Domain events

Cross-package integration goes through `event/` packages and an
`EventBus` port. Hosts subscribe in their boot code. Never import
another business module directly; route through the bus.

### 4. Additive evolution

Within a major version (`v0.x.y`):

- Adding new fields, methods, types: ✅
- Adding new exported functions: ✅
- Renaming, removing, changing signatures of exported symbols: ❌

Breaking changes require a major bump (`v1.x.y`) with a migration note
in the CHANGELOG.

## Adding a new package

1. Create `<tier>/<name>/` (e.g. `foundation/retry/` or `modules/sms/`)
2. Write package doc on the entry-point file (`<name>.go`)
3. Write `README.md` (use any existing package as a template)
4. Write `CHANGELOG.md` with an "Unreleased" section
5. Add tests; aim for 70% coverage at minimum
6. Run `make purity-check` to catch accidental imports
7. Add the new path to the top-level `README.md` when it is part of the public surface

Do not add a nested `go.mod` or `go.sum`. New code should join the root
module.

Foundation packages should be tiny and self-contained. Business
packages follow the DDD layering used by `modules/auth/`,
`modules/billing/`, `modules/email/`, `modules/referral/`.

## Pull request checklist

- [ ] `make test-race` passes
- [ ] `make fmt` clean
- [ ] `make purity-check` clean
- [ ] CHANGELOG updated under "Unreleased"
- [ ] README updated if public API changed
- [ ] Test coverage didn't drop (CI shows repo-wide coverage — adding tests
      is welcome even when not strictly required)
- [ ] If adding a new export, it has a doc comment

## Releasing

See [VERSIONING.md](./VERSIONING.md) for the full policy. Quick steps:

```bash
# 1. Move "Unreleased" entries in the touched package CHANGELOG.md
#    files under a new version + date heading
# 2. Commit on main
git tag v<x.y.z>
git push origin v<x.y.z>
```

Tags push to GitHub → Go proxy picks the version up within minutes.

## Code style

- Go 1.25 (see the root `go.mod`).
- `slog` for logging — never `fmt.Println` or `log.Print*` in library code.
- Errors: prefer `fmt.Errorf("...: %w", err)` for wrapping; use sentinel
  errors (e.g. `ErrInvalidInput`) for branchable categories.
- Tests use `_test.go` files; testify is allowed but stdlib `testing` is
  preferred for small modules.
- Keep doc comments single-purpose and at the top of exported symbols.

## Reporting issues

Open a GitHub issue with:

- Package + version (e.g. `modules/auth @ v0.3.0`)
- Minimal reproducer (Go playground link or short snippet)
- Expected vs actual behavior

Security issues — email the maintainers directly, do not open a public issue.
