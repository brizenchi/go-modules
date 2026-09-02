#!/usr/bin/env bash
set -euo pipefail

usage() {
  echo "usage: $0 <destination> <go-module> <app-name> <go-modules-version>" >&2
  echo "example: $0 ../my-saas github.com/me/my-saas my-saas v0.3.0" >&2
}

if [ "$#" -ne 4 ]; then
  usage
  exit 2
fi

destination=$1
go_module=$2
app_name=$3
go_modules_version=$4
script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repo_root=$(CDPATH= cd -- "$script_dir/.." && pwd)

if [ -e "$destination" ]; then
  echo "destination already exists: $destination" >&2
  exit 1
fi
if [[ ! "$go_module" =~ ^[A-Za-z0-9._~/-]+$ ]]; then
  echo "invalid Go module path: $go_module" >&2
  exit 1
fi
if [[ ! "$app_name" =~ ^[a-z0-9][a-z0-9-]*$ ]]; then
  echo "app-name must be a lowercase DNS-style slug" >&2
  exit 1
fi
if [[ ! "$go_modules_version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+([-.+][A-Za-z0-9.-]+)?$ ]]; then
  echo "go-modules-version must be a published semantic version such as v0.3.0" >&2
  exit 1
fi

mkdir -p "$destination/backend" "$destination/frontend"
tar -C "$repo_root/templates/quickstart" \
  --exclude='.env' --exclude='.cache' --exclude='quickstart' \
  -cf - . | tar -C "$destination/backend" -xf -
tar -C "$repo_root/templates/quickstart-nextjs" \
  --exclude='.env.local' --exclude='.next' --exclude='node_modules' \
  -cf - . | tar -C "$destination/frontend" -xf -

old_module='github.com/brizenchi/quickstart-template'
find "$destination/backend" -type f \( -name '*.go' -o -name 'go.mod' \) \
  -exec env OLD_MODULE="$old_module" NEW_MODULE="$go_module" \
  perl -pi -e 's/\Q$ENV{OLD_MODULE}\E/$ENV{NEW_MODULE}/g' {} +

old_app='quickstart'
find "$destination/backend" "$destination/frontend" -type f \
  \( -name '*.go' -o -name '*.yaml' -o -name '*.example' -o -name '*.json' -o -name '*.md' -o -name '*.ts' -o -name '*.tsx' -o -name '*.mjs' \) \
  -exec env OLD_APP="$old_app" NEW_APP="$app_name" \
  perl -pi -e 's/\Q$ENV{OLD_APP}\E/$ENV{NEW_APP}/g' {} +

(
  cd "$destination/backend"
  GOWORK=off go mod edit -module "$go_module"
  GOWORK=off go mod edit -require="github.com/brizenchi/go-modules@$go_modules_version"
  GOWORK=off go mod tidy
)

echo "created $app_name in $destination"
echo "next:"
echo "  cp $destination/backend/deploy/config.yaml.example $destination/backend/deploy/config.yaml"
echo "  cp $destination/backend/.env.example $destination/backend/.env"
echo "  cp $destination/frontend/.env.example $destination/frontend/.env.local"
echo "  cd $destination/backend && GOWORK=off go test ./..."
echo "  cd $destination/frontend && npm install && npm run verify"
