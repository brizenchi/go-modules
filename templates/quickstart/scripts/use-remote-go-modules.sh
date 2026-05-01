#!/usr/bin/env bash

set -euo pipefail

modules=(
  github.com/brizenchi/go-modules/foundation/config
  github.com/brizenchi/go-modules/foundation/ginx
  github.com/brizenchi/go-modules/foundation/httpresp
  github.com/brizenchi/go-modules/foundation/pgx
  github.com/brizenchi/go-modules/foundation/slog
  github.com/brizenchi/go-modules/foundation/tracing
  github.com/brizenchi/go-modules/modules/auth
  github.com/brizenchi/go-modules/modules/billing
  github.com/brizenchi/go-modules/modules/email
  github.com/brizenchi/go-modules/modules/referral
  github.com/brizenchi/go-modules/modules/user
  github.com/brizenchi/go-modules/stacks/saascore
)

for module in "${modules[@]}"; do
  go mod edit -dropreplace "$module" || true
done

if ! go mod tidy; then
  cat <<'EOF'
Failed to switch to published GitHub module versions.

Most likely cause:
- one or more required go-modules tags have not been published yet

Restore local iteration by keeping the existing replace directives, or
publish the missing module tags before retrying.
EOF
  exit 1
fi

echo "Switched go.mod back to published GitHub module versions."
