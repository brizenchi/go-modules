<!--
PR title = final squash commit: type(scope): subject
e.g. fix(api): reject expired tokens — see docs/standards/GIT_WORKFLOW.md
-->

## What and why

<!-- Context, problem, approach. Related issue: Closes #123 -->

## How it was verified

<!-- Tests run, manual steps, screenshots or logs -->
- [ ] `make fmt && make test-race && make purity-check`
- [ ] `cd templates/quickstart && go test ./...` (when the template changed)
- [ ] `cd templates/quickstart-nextjs && npm run verify` (when the frontend changed)

## Impact and risk

- [ ] Public API of `foundation/*` or `modules/*` changed: additive only, and the package `CHANGELOG.md` is updated
- [ ] API changed: callers' types or clients (frontend, other services) are updated in this pull request
- [ ] Database changed: a new migration was added, no applied migration was edited, and it can be rolled back
- [ ] New configuration: example configuration and docs are updated, and every environment is set
- [ ] New dependency: the reason is given below

<!-- How to roll back; release ordering to watch out for -->

## Checklist

- [ ] Meets the [code review checklist](../docs/standards/CODE_REVIEW.md#checklist)
- [ ] No secrets committed; fake secrets in tests are obviously fake (`*_not-a-real-key`)
