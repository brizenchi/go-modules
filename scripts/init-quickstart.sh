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
(
  cd "$repo_root/templates/quickstart"
  shopt -s dotglob
  backend_entries=()
  for entry in *; do
    case "$entry" in .env|.cache|quickstart) continue ;; esac
    backend_entries+=("$entry")
  done
  tar -cf - "${backend_entries[@]}"
) | tar -C "$destination/backend" -xf -
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
# Renaming the module and app changes import order and alignment.
gofmt -s -w "$destination/backend"

# Engineering standards come from dev-standards (installed with Copier, so the
# project can `uvx copier update` later). Override the source for local testing
# with DEV_STANDARDS_SRC=/path/to/dev-standards.
standards_src=${DEV_STANDARDS_SRC:-gh:brizenchi/dev-standards}
github_repo=$app_name
case "$go_module" in github.com/*/*) github_repo=$(printf '%s' "$go_module" | cut -d/ -f2-3) ;; esac
case "$github_repo" in */*) ;; *) github_repo="brizenchi/$app_name" ;; esac
git -C "$destination" init -q
if command -v uvx >/dev/null 2>&1; then copier=(uvx --quiet copier)
elif command -v pipx >/dev/null 2>&1; then copier=(pipx run copier)
else echo "error: needs uv (https://docs.astral.sh/uv/) or pipx to install dev-standards" >&2; exit 1; fi
"${copier[@]}" copy --quiet --defaults \
  --data "project_name=$app_name" --data "github_repo=$github_repo" \
  --data 'languages=["go","node"]' \
  --data 'go_modules=[{"dir":"backend"}]' \
  --data 'node_projects=[{"dir":"frontend","scripts":"verify"}]' \
  --data 'commit_scopes=["api","web","deploy","docs","ci"]' \
  --data ci_mode=reusable --data ci_caller_job=standards \
  "$standards_src" "$destination" >/dev/null

# Quickstart-specific additions on top of the standards: project CI with the
# deploy gate, project docs, AI skills and the project section of AGENTS.md.
(
  cd "$repo_root/templates/project-extras"
  tar --exclude=AGENTS.project.md -cf - .
) | tar -C "$destination" -xf -
mkdir -p "$destination/scripts" "$destination/.claude"
cp -R "$repo_root/docs/adr" "$destination/docs/"
cp "$repo_root/docs/ARCHITECTURE.md" "$repo_root/docs/CONFIG_STANDARD.md" "$repo_root/docs/OBSERVABILITY.md" \
  "$repo_root/docs/SETUP_ZH.md" "$repo_root/docs/DEPLOYMENT.md" "$destination/docs/"
cp "$repo_root/scripts/deploy-dokploy.sh" "$destination/scripts/"
cp -R "$repo_root/.claude/skills" "$destination/.claude/"
printf 'observability-config\n' >> "$destination/.github/required-checks.txt"
DEST_AGENTS="$destination/AGENTS.md" EXTRA="$repo_root/templates/project-extras/AGENTS.project.md" python3 - <<'PY'
import os
path, extra = os.environ["DEST_AGENTS"], open(os.environ["EXTRA"]).read()
text = open(path).read()
marker = "## Project-specific rules"
text = text[: text.index(marker)] + extra
open(path, "w").write(text)
PY
# Repository paths become the project's layout.
find "$destination/docs" "$destination/.claude/skills" -type f -name '*.md' \
  -exec perl -pi -e 's{templates/quickstart-nextjs}{frontend}g; s{templates/quickstart}{backend}g; s{ && make purity-check}{}g' {} +

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
echo "  cd $destination && lefthook install   # standards: docs/standards; AI rules: AGENTS.md / CLAUDE.md"
echo "  gh auth login && .standards/bin/setup-github --dry-run   # GitHub rulesets, squash-only, secret scanning"
echo "  cd $destination/backend && GOWORK=off go test ./..."
echo "  cd $destination/frontend && npm install && npm run verify"
