#!/usr/bin/env bash
# PostToolUse: format files the assistant just edited, so formatting never
# reaches review. Formatters are used only when installed.
# See docs/standards/AI_ASSISTANTS.md.
set -euo pipefail
# shellcheck source-path=SCRIPTDIR source=lib.sh
source "$(dirname "$0")/lib.sh"

file=$(json_field '.tool_input.file_path' <<<"$(cat)")
[ -n "$file" ] && [ -f "$file" ] || exit 0

case "$file" in
  *.go)
    command -v gofmt >/dev/null 2>&1 && gofmt -s -w "$file" ;;
  *.py)
    if command -v ruff >/dev/null 2>&1; then ruff format --quiet "$file"
    elif command -v uvx >/dev/null 2>&1; then uvx --quiet ruff format --quiet "$file"; fi ;;
  *.ts|*.tsx|*.js|*.jsx|*.mjs|*.cjs|*.css|*.scss|*.json)
    # Only when the project uses Prettier (local install), never a global guess.
    dir=$(dirname "$file")
    while [ "$dir" != "/" ] && [ ! -f "$dir/package.json" ]; do dir=$(dirname "$dir"); done
    if [ -x "$dir/node_modules/.bin/prettier" ]; then "$dir/node_modules/.bin/prettier" --write --log-level warn "$file"; fi ;;
esac
exit 0
