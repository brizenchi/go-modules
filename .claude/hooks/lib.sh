# shellcheck shell=bash
# Shared helper for Claude Code hooks: read one field from the hook JSON on stdin.
# Usage: value=$(json_field '.tool_input.file_path' <<<"$input")
json_field() {
  if command -v jq >/dev/null 2>&1; then
    jq -r "$1 // empty"
  elif command -v python3 >/dev/null 2>&1; then
    python3 -c '
import json, sys
data = json.load(sys.stdin)
for key in sys.argv[1].lstrip(".").split("."):
    data = data.get(key) if isinstance(data, dict) else None
print(data or "")' "$1"
  fi
}
