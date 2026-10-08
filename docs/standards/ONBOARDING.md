# Onboarding

## Day one

### 1. Install the common tools
```bash
brew install git gh lefthook gitleaks uv
curl -fsSL https://raw.githubusercontent.com/brizenchi/keel/main/install.sh | sh   # the keel command (optional; the project also ships .keel/bin/keel)
```
Go: the version in `go.mod` (`brew install go`).
Node: the version in `node_projects` in `.keel/answers.yml` (default 22); the package manager follows the lock file.

### 2. Clone and install the hooks
```bash
git clone git@github.com:brizenchi/go-modules.git
cd go-modules
keel doctor               # check that the tools are installed
keel hooks                # install the pre-commit checks: secret scan, formatting
```

### 3. Run the project
Follow [PROJECT.md](./PROJECT.md#local-development).

## First week: reading

1. [PROJECT.md](./PROJECT.md): this project's architecture, layout, deployment and conventions.
2. The [standards overview](./README.md), especially:
   - [GIT_WORKFLOW.md](./GIT_WORKFLOW.md): branches, commit messages, pull requests;
   - [CODE_STYLE.md](./CODE_STYLE.md) and the guide for your language;
   - [TESTING.md](./TESTING.md) and [API_STANDARD.md](./API_STANDARD.md).

## Your first pull request

1. `git switch -c fix/<scope>-<summary>`.
2. Change the code, add tests, and run the checks for the parts you changed (listed in the PR template).
3. `git commit -m "fix(<scope>): <summary>"`.
4. Push and open a pull request; the title follows the same format as commit messages.
5. Address review comments; squash-merge once CI passes.

## AI assistants

`AGENTS.md` (read by Codex, Cursor and others) and `CLAUDE.md` (read by Claude Code)
summarise these standards, so assistants follow them automatically. You are still
responsible for reviewing and testing what they produce. See [AI_ASSISTANTS.md](./AI_ASSISTANTS.md).

## Access

Ask a maintainer for the access you need (repository write access, read-only
access to monitoring, test accounts for third-party services). Production secrets
are never handed to individuals; local development uses test keys.
