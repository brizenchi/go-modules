# AI coding assistants

The standards reach AI assistants (Claude Code, Codex, Cursor, …) in three layers:
**rule files** say what to follow, **on-demand procedures** describe how to do specific
tasks, and **automated checks** make sure non-compliant code cannot land even when an
assistant ignores the text.

```text
Standards (for people)        docs/standards/*.md
        │ distilled into
Rule files (always loaded)    AGENTS.md          ← read by Codex, Cursor and others
                              CLAUDE.md          ← read by Claude Code; contains @AGENTS.md
        │
Procedures (on demand)        .claude/skills/<name>/SKILL.md (added by the project)
        │
Automated checks (enforced)   .claude/settings.json → lefthook → CI
```

## How AGENTS.md is organised

| Part | Owner | Notes |
| --- | --- | --- |
| From the top to "Project-specific rules" | keel | refreshed by `keel update` |
| "Project-specific rules" | **this project** | architecture, layout, commands; kept on update |

Subdirectories may have their own `AGENTS.md` (plus a `CLAUDE.md` containing only
`@AGENTS.md`) with rules specific to that directory.

Writing guidelines:

- **Short**: keep the root `AGENTS.md` around 100 lines; every rule costs context in every session.
- **Imperative and actionable**: "must …", with file paths and commands.
- **Conclusions plus links**: the details live in `docs/standards/`.

## Automated checks

| Layer | What it does | Applies to |
| --- | --- | --- |
| Claude Code hooks (`.claude/settings.json`) | formats files after edits (Go, Python, Prettier-based frontends); blocks `--no-verify`, `git push --force` and force-adding `.env` | Claude Code |
| Git hooks (`lefthook.yml`) | secret scan, formatting | every person and every assistant, once installed |
| CI | everything | every change; cannot be skipped |

**Rules that really matter need an automated check**: an assistant may ignore text, but it cannot get past CI.

## Skills (optional)

Write complete procedures (adding an endpoint, a database migration, a third-party
integration, …) as `.claude/skills/<name>/SKILL.md`; Claude Code loads them on demand.
Other tools can read the same files — list their paths in the project section of `AGENTS.md`.

## Checking that the rules are loaded

- Claude Code: run `/memory` to see the loaded `CLAUDE.md` and the `AGENTS.md` it imports.
- Any tool: ask "what is this project's commit message format?" and check the answer.

## Maintenance

- Shared rules: change and release them in keel, then run `keel update` in each project.
- This project's rules: edit the project section of `AGENTS.md` and [PROJECT.md](./PROJECT.md).
- When an assistant keeps making the same mistake: add a check to CI if a tool can catch it
  (to the Git hooks only if it must be stopped before the commit), otherwise add an explicit rule to `AGENTS.md`.
- Personal preferences go in `~/.claude/CLAUDE.md` or `.claude/settings.local.json`, not in the repository.
