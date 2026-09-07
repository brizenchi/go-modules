#!/usr/bin/env bash
set -euo pipefail

version=${1:-local}
script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repo_root=$(CDPATH= cd -- "$script_dir/.." && pwd)
temp_root=$(mktemp -d "${TMPDIR:-/tmp}/quickstart-release.XXXXXX")
trap 'rm -rf "$temp_root"' EXIT

mkdir -p "$temp_root/backend"
(
  cd "$repo_root/templates/quickstart"
  shopt -s dotglob
  backend_entries=()
  for entry in *; do
    case "$entry" in .env|.cache|quickstart) continue ;; esac
    backend_entries+=("$entry")
  done
  tar -cf - "${backend_entries[@]}"
) | tar -C "$temp_root/backend" -xf -

cd "$temp_root/backend"
if [ "$version" = "local" ]; then
  GOWORK=off go mod edit -replace="github.com/brizenchi/go-modules=$repo_root"
  GOWORK=off CGO_ENABLED=0 GOOS=linux go build -mod=readonly -ldflags="-s -w" -o "$temp_root/quickstart-linux" ./cmd/quickstart
else
  GOWORK=off go mod edit -dropreplace=github.com/brizenchi/go-modules 2>/dev/null || true
  GOWORK=off go mod edit -require="github.com/brizenchi/go-modules@$version"
fi

GOWORK=off go mod tidy
GOWORK=off go mod download
GOWORK=off go test ./...
GOWORK=off go vet ./...
GOWORK=off go build ./...
GOWORK=off CGO_ENABLED=0 GOOS=linux go build -mod=readonly -ldflags="-s -w" -o "$temp_root/quickstart-linux" ./cmd/quickstart

echo "quickstart detached-copy verification passed (go-modules=$version)"
