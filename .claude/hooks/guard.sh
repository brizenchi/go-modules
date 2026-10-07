#!/usr/bin/env bash
# PreToolUse(Bash): refuse commands that bypass the repository's safety
# checks. Exit code 2 blocks the command and shows the reason to the assistant.
# See docs/standards/GIT_WORKFLOW.md and docs/standards/AI_ASSISTANTS.md.
set -euo pipefail
# shellcheck source-path=SCRIPTDIR source=lib.sh
source "$(dirname "$0")/lib.sh"

cmd=$(json_field '.tool_input.command' <<<"$(cat)")
[ -n "$cmd" ] || exit 0

block() { echo "Blocked by .claude/hooks/guard.sh: $1" >&2; exit 2; }

if grep -Eq -- '(^|[[:space:]])--no-verify([[:space:]]|$)' <<<"$cmd"; then
  block "--no-verify skips the commit hooks (format, commit message, secret scan). Fix the reported problem instead."
fi
if grep -Eq -- 'git[[:space:]]+push([^|;&]*)[[:space:]](--force|--force-with-lease|-f)([[:space:]=]|$)' <<<"$cmd"; then
  block "force push rewrites shared history. Ask the user to do it explicitly if it is really needed."
fi
if grep -Eq -- 'git[[:space:]]+add([^|;&]*)[[:space:]](-f|--force)[[:space:]][^|;&]*\.env' <<<"$cmd"; then
  block "force-adding .env files would commit secrets."
fi
exit 0
