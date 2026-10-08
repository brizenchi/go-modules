# Git workflow

## Branches

`main` is the only long-lived branch and is always releasable.

| Branch | Name | Purpose |
| --- | --- | --- |
| Feature | `feat/<scope>-<summary>` | new functionality, e.g. `feat/billing-annual-plan` |
| Fix | `fix/<scope>-<summary>` | bug fix, e.g. `fix/auth-token-expiry` |
| Other | `chore/`, `docs/`, `refactor/`, `test/`, `ci/` | same as the commit type |
| Hotfix | `hotfix/<summary>` | production incident, see below |

- Branch names use lowercase letters, digits and `-` only.
- One branch does one thing and lives no longer than about three days.
- Every change reaches `main` through a pull request (see "Branch protection").

## Commit messages: Conventional Commits

```text
<type>(<scope>): <subject>

<body: why and how, optional>

<footer: BREAKING CHANGE / Refs #123, optional>
```

| type | Meaning | In the changelog |
| --- | --- | --- |
| `feat` | new functionality | yes |
| `fix` | bug fix | yes |
| `perf` | performance improvement | yes |
| `refactor` | restructuring without behaviour change | no |
| `docs` | documentation only | no |
| `test` | tests only | no |
| `build` | dependencies, build system | no |
| `ci` | CI configuration | no |
| `chore` | anything else | no |
| `revert` | reverts an earlier commit | yes |

The **scope** is the module or directory the change is in; common scopes in this project: `tracing`, `ginx`, `slog`, `httpx`, `pgx`, `rdx`, `auth`, `billing`, `email`, `referral`, `quickstart`, `nextjs`, `deploy`, `docs`, `ci`. When a change spans several modules, use the most affected one or leave the scope out.

Rules:

- The subject is in the imperative mood, has no trailing period and is at most 72 characters.
- No uninformative messages such as `update` or `fix bug`.
- Breaking changes add `!` after the type and a `BREAKING CHANGE: <how to migrate>` footer.
- One commit does one thing; formatting, renames and behaviour changes are separate commits.

Examples:

```text
feat(api): add pagination to invoice list
fix(auth): reject expired refresh tokens
docs(standards): clarify migration rollback
feat(auth)!: require PKCE for OAuth login

BREAKING CHANGE: clients must send code_verifier on the token exchange.
```

CI's `commits / pr-title` check validates pull request titles. Check a message locally
with `keel lint-commit`.

## Pull requests

1. Branch from the latest `main`; push and open a pull request when done.
2. **The title must be a valid commit message**: pull requests are squash-merged and the title becomes the commit.
3. Fill in the template: what changed, why, how it was verified, risk and how to roll back.
4. Merge only when CI passes and the required approvals are in (see [CODE_REVIEW.md](./CODE_REVIEW.md)).
5. Always **Squash and merge**; the branch is deleted automatically.

Size: aim for under 400 changed lines (excluding generated code and fixtures). Split
large features into pull requests that can merge independently, and hide unfinished
parts behind a configuration flag.

## Rewriting history

- **Never force-push `main` or any pushed branch that others use.**
- Amend, rebase and squash only commits that have not been pushed.
- If a secret was pushed: **revoke and rotate it immediately**, then follow
  [SECURITY_STANDARD.md](./SECURITY_STANDARD.md#leaked-secrets). Rewriting history does not un-leak a secret.

## Hotfixes

1. Branch `hotfix/<summary>` from `main` and fix only the problem itself.
2. Title the pull request `fix(<scope>): ...`; one reviewer may approve it quickly.
3. After merging, verify the release and add the missing tests within 24 hours, together with a postmortem (see [INCIDENT_RESPONSE.md](./INCIDENT_RESPONSE.md)).

If a fix is not quick, roll back first: `git revert` the bad commit through a pull request,
or use your deployment platform's rollback.

## Branch protection

`keel github` (needs `gh` and repository admin rights) creates a ruleset named `keel`:

- the default branch cannot be deleted or force-pushed, and history stays linear;
- changes merge through pull requests only, squash only, with all review threads resolved;
- the checks listed in `.keel/required-checks.txt` must pass;
- one approval is required by default; `--solo` drops the approval for single-maintainer
  repositories (pull requests and CI are still required);
- repository admins may bypass in emergencies; `--no-admin-bypass` turns that off.

Run it again after changing the rules; it is idempotent.
