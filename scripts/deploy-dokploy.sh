#!/usr/bin/env bash
# Triggers Dokploy deployments for the commit that just passed CI.
#
# Env:
#   DOKPLOY_URL              https://dokploy.example.com (no trailing slash needed)
#   DOKPLOY_API_KEY          Dokploy → Settings → Profile → API/CLI token
#   DOKPLOY_APPLICATION_IDS  comma-separated application ids (from the app URL)
#   GITHUB_SHA, GITHUB_REF_NAME, GITHUB_REPOSITORY  set by GitHub Actions
#
# Dokploy deploys the branch head, not a specific commit. If main moved on
# while this run was in CI, deploying now would ship a commit that has not
# passed CI yet, so this run steps aside and the newer run deploys instead.
set -euo pipefail

if [ -z "${DOKPLOY_URL:-}" ] || [ -z "${DOKPLOY_API_KEY:-}" ] || [ -z "${DOKPLOY_APPLICATION_IDS:-}" ]; then
  echo "::warning::Dokploy secrets are not configured; skipping deploy (see docs/standards/DEPLOYMENT.md)."
  exit 0
fi

branch=${GITHUB_REF_NAME:-main}
head=$(git ls-remote "https://github.com/${GITHUB_REPOSITORY}.git" "refs/heads/${branch}" | cut -f1)
if [ -n "$head" ] && [ "$head" != "${GITHUB_SHA}" ]; then
  echo "::notice::${branch} moved to ${head:0:7}; skipping deploy of ${GITHUB_SHA:0:7} (the newer run deploys after its CI)."
  exit 0
fi

base=${DOKPLOY_URL%/}
IFS=',' read -r -a apps <<<"$DOKPLOY_APPLICATION_IDS"
for raw in "${apps[@]}"; do
  app=$(printf '%s' "$raw" | tr -d '[:space:]')
  [ -n "$app" ] || continue
  echo "Deploying application ${app} at ${GITHUB_SHA:0:7}…"
  body=$(printf '{"applicationId":"%s"}' "$app")
  status=$(curl -sS -o /tmp/dokploy-response -w '%{http_code}' -X POST "${base}/api/application.deploy" \
    -H 'accept: application/json' -H 'Content-Type: application/json' \
    -H "x-api-key: ${DOKPLOY_API_KEY}" -d "$body")
  if [ "${status}" -lt 200 ] || [ "${status}" -ge 300 ]; then
    echo "::error::Dokploy returned HTTP ${status} for application ${app}: $(head -c 300 /tmp/dokploy-response)"
    exit 1
  fi
done
echo "Deploy triggered. Watch progress in Dokploy, then run the post-deploy checks."
