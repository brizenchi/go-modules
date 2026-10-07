#!/usr/bin/env bash
# PostToolUse: format files the assistant just edited, so formatting never
# reaches review. See docs/standards/AI_ASSISTANTS.md.
set -euo pipefail
source "$(dirname "$0")/lib.sh"

file=$(json_field '.tool_input.file_path' <<<"$(cat)")
[ -n "$file" ] && [ -f "$file" ] || exit 0

case "$file" in
  *.go) gofmt -s -w "$file" ;;
esac
exit 0
