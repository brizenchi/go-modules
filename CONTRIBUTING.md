# Contributing to go-modules

Thanks for your interest. This monorepo holds the org's reusable Go
modules. Every change should hold the line on three things: **purity**
(no project-specific imports leak in), **stability** (additive changes
within a major), and **discoverability** (each module's README +
CHANGELOG stays accurate).

## Repo layout

```
foundation/<name>/   pure infra: stdlib + one-or-two common libs only
modules/<name>/      DDD-layered business modules
docs/                host integration and migration guides
templates/quickstart/ runnable boot example consuming all modules
scripts/             one-off helpers (extraction, migration)
```

Every directory listed in `go.work` is its own Go module with its own
`go.mod`. Modules are tagged independently as `<name>/v<x.y.z>` (e.g.
`modules/auth/v0.2.1`).

## Local setup

```bash
git clone git@github.com:brizenchi/go-modules.git
cd go-modules
make test          # all modules, no -race
make test-race     # all modules with -race
make purity-check  # ensure no host-app imports leaked
make fmt           # gofmt every module
make tidy          # go mod tidy every module
```

`go.work` makes cross-module changes work without `replace` directives —
edit `modules/auth` and `modules/billing` together and `go test` sees
the changes.

## Design principles

### 1. Purity

`pkg/<module>/` MUST NOT import any project-specific package. Allowed
deps: stdlib + a small set of common libraries (`gin`, `gorm`,
`stripe-go/v76`, `golang-jwt/v5`, `go-redis/v9`, `viper`, `mapstructure`).

Project-specific glue (e.g. wiring viper config to a module's `Config`
struct, mapping the host's user table to `port.UserStore`) lives in the
**host project**, not here. The host adapter pattern is shown in
`templates/quickstart/internal/*_glue/`.

`make purity-check` enforces this — CI fails on violations.

### 2. Ports + adapters

Business modules expose a `port/` package with the interfaces they
depend on, and one or more `adapter/` implementations. Adding a new
provider for an existing module = new adapter + tests. **Don't**
modify the port to fit a single provider's quirks — use an adapter
or extend `domain/` if the concept is universal.

### 3. Domain events

Cross-module integration goes through `event/` packages and an
`EventBus` port. Hosts subscribe in their boot code. Never import
another business module directly; route through the bus.

### 4. Additive evolution

Within a major version (`v0.x.y`):

- Adding new fields, methods, types: ✅
- Adding new exported functions: ✅
- Renaming, removing, changing signatures of exported symbols: ❌

Breaking changes require a major bump (`v1.x.y`) with a migration note
in the CHANGELOG.

## Adding a new module

1. Create `<tier>/<name>/` (e.g. `foundation/retry/` or `modules/sms/`)
2. `go mod init github.com/brizenchi/go-modules/<tier>/<name>`
3. Add the module path to `go.work`
4. Write package doc on the entry-point file (`<name>.go`)
5. Write `README.md` (use any existing module as a template)
6. Write `CHANGELOG.md` with an "Unreleased" section
7. Add tests; aim for 70% coverage at minimum
8. Run `make purity-check` to catch accidental imports
9. Add the new path to the `MODULES` list in `Makefile`
10. Add the new path to `.github/workflows/ci.yml` matrix

If one module depends on another module in this repo, keep both:

- a semver `require github.com/brizenchi/go-modules/modules/<module> vX.Y.Z`
- a local `replace github.com/brizenchi/go-modules/modules/<module> => ../<module>`

That combination keeps external consumers on proper versions while letting
`go mod tidy` pass inside the monorepo before the dependency tag exists.

Foundation modules should be tiny and self-contained. Business modules
follow the DDD layering used by `modules/auth/`, `modules/billing/`,
`modules/email/`, `modules/referral/`.

If you add a new module under `foundation/`, `modules/`, or `stacks/`,
add it to:

- `go.work`
- the `MODULES` list in `Makefile`
- the matrices in `.github/workflows/ci.yml`
- the top-level `README.md`

## Pull request checklist

- [ ] `make test-race` passes
- [ ] `make fmt` clean
- [ ] `make purity-check` clean
- [ ] CHANGELOG updated under "Unreleased"
- [ ] README updated if public API changed
- [ ] Test coverage didn't drop (CI shows per-module % — adding tests
      is welcome even when not strictly required)
- [ ] If adding a new export, it has a doc comment

## Releasing

See [VERSIONING.md](./VERSIONING.md) for the full policy. Quick steps:

```bash
# 1. Move "Unreleased" entries in <module>/CHANGELOG.md under a new
#    version + date heading
# 2. Commit on main
git tag <module>/v<x.y.z>
git push origin <module>/v<x.y.z>
```

Tags push to GitHub → Go proxy picks the version up within minutes.

## Code style

- Go 1.25 (see `go.work`).
- `slog` for logging — never `fmt.Println` or `log.Print*` in library code.
- Errors: prefer `fmt.Errorf("...: %w", err)` for wrapping; use sentinel
  errors (e.g. `ErrInvalidInput`) for branchable categories.
- Tests use `_test.go` files; testify is allowed but stdlib `testing` is
  preferred for small modules.
- Keep doc comments single-purpose and at the top of exported symbols.

## Reporting issues

Open a GitHub issue with:

- Module + version (e.g. `auth v0.1.1`)
- Minimal reproducer (Go playground link or short snippet)
- Expected vs actual behavior

Security issues — email the maintainers directly, do not open a public issue.
