#!/usr/bin/env bash
# Rehearses docs/MIGRATE_FROM_SAASCORE.md on a throwaway PostgreSQL: creates the
# saascore schema with the legacy code itself, loads seed.sql, runs every
# upgrade step against the current template and compares what the upgraded
# application returns with expected.tsv.
#
# Needs git, go, curl, jq and PostgreSQL server binaries (initdb, pg_ctl, psql;
# set PG_BIN when they are not on PATH). The first run downloads the legacy
# module's dependencies. Everything lives in a temporary directory that is
# removed afterwards; KEEP=1 keeps it (logs, JSON reports, snapshots).
#
#   scripts/saascore-rehearsal/run.sh
set -euo pipefail

ROOT=$(git -C "$(dirname "$0")" rev-parse --show-toplevel)
HERE=$ROOT/scripts/saascore-rehearsal
GUIDE=$ROOT/docs/migrate-from-saascore
LEGACY_REV=${LEGACY_REV:-d6a572f}
APP_PORT=${APP_PORT:-18080}
JWT_SECRET=sk_test_not-a-real-jwt-secret-0000000000
SERVICE=rehearsal
# /tmp keeps the PostgreSQL socket path under the 103-byte limit on macOS.
WORK=$(mktemp -d /tmp/saascore-rehearsal.XXXXXX)
APP_PID=""
FAILURES=0

find_pg_bin() {
	if [ -n "${PG_BIN:-}" ]; then echo "$PG_BIN"; return; fi
	# initdb alone is not enough: Homebrew's libpq ships it without the server.
	for dir in "$(dirname "$(command -v initdb 2>/dev/null || echo /nonexistent/initdb)")" \
		/opt/homebrew/opt/postgresql@*/bin /usr/local/opt/postgresql@*/bin /usr/lib/postgresql/*/bin; do
		if [ -x "$dir/initdb" ] && [ -x "$dir/postgres" ]; then echo "$dir"; return; fi
	done
	echo "PostgreSQL server binaries not found; set PG_BIN" >&2
	exit 2
}
PG_BIN=$(find_pg_bin)
export PGHOST=$WORK/pg PGPORT=${PGPORT:-55432} PGUSER=postgres PGDATABASE=app
DSN="host=$PGHOST port=$PGPORT user=postgres dbname=app sslmode=disable"

step() { printf '\n==> %s\n' "$*"; }
fail() { echo "FAIL: $*"; FAILURES=$((FAILURES + 1)); }
sql() { "$PG_BIN/psql" -X -q -v ON_ERROR_STOP=1 "$@"; }

start_app() {
	(cd "$WORK" && exec env \
		CONFIG="$WORK/config.yaml" APP_ENV=dev APP_SERVER_NAME="$SERVICE" APP_SERVER_PORT="$APP_PORT" APP_LOG_FORMAT=text \
		APP_DB_HOST="$PGHOST" APP_DB_PORT="$PGPORT" APP_DB_USER=postgres APP_DB_PASSWORD=unused APP_DB_NAME=app APP_DB_SSL_MODE=disable \
		APP_AUTH_USER_JWT_SECRET="$JWT_SECRET" APP_AUTH_EMAIL_DEBUG=true APP_EMAIL_PROVIDER=log \
		APP_BILLING_ENABLED=true APP_BILLING_STRIPE_SECRET_KEY=sk_test_not-a-real-key \
		APP_BILLING_STRIPE_WEBHOOK_SECRET=whsec_not-a-real-secret \
		APP_BILLING_STRIPE_PRICES_STARTER_MONTHLY=price_test_starter_monthly \
		APP_BILLING_STRIPE_PRICES_PRO_MONTHLY=price_test_pro_monthly \
		"$WORK/quickstart" >>"$WORK/app.log" 2>&1) &
	APP_PID=$!
	for _ in $(seq 1 60); do
		if curl -sf "http://127.0.0.1:$APP_PORT/health" >/dev/null; then return; fi
		if ! kill -0 "$APP_PID" 2>/dev/null; then
			tail -20 "$WORK/app.log"
			echo "application exited while starting" >&2
			exit 1
		fi
		sleep 1
	done
	echo "application did not become healthy" >&2
	exit 1
}

stop_app() {
	if [ -n "$APP_PID" ] && kill -0 "$APP_PID" 2>/dev/null; then
		kill "$APP_PID"
		wait "$APP_PID" 2>/dev/null || true
	fi
	APP_PID=""
}

cleanup() {
	stop_app
	"$PG_BIN/pg_ctl" -D "$WORK/pg/data" -m fast stop >/dev/null 2>&1 || true
	if [ "${KEEP:-0}" = 1 ]; then echo "work directory kept: $WORK"; else rm -rf "$WORK"; fi
}
trap cleanup EXIT

api() { # METHOD PATH [TOKEN] [JSON]
	local args=(-s -X "$1" "http://127.0.0.1:$APP_PORT/api/v1$2" -H 'Content-Type: application/json')
	if [ -n "${3:-}" ]; then args+=(-H "Authorization: Bearer $3"); fi
	if [ -n "${4:-}" ]; then args+=(-d "$4"); fi
	curl "${args[@]}"
}

login() { # EMAIL -> token, through the e-mail code flow (debug mode returns the code)
	local code
	code=$(api POST /auth/send-code "" "{\"email\":\"$1\"}" | jq -r '.data.debug_code // empty')
	if [ -z "$code" ]; then echo "no debug code for $1" >&2; return 1; fi
	api POST /auth/verify-code "" "{\"email\":\"$1\",\"code\":\"$code\"}" | jq -r '.data.token // empty'
}

oldside() { (cd "$WORK/legacy" && GOWORK=off GOFLAGS=-mod=mod go run ./cmd/oldside/main.go "$@" 2>>"$WORK/oldside.log"); }
billing() { "$WORK/legacy-billing-migrate" --dsn "$DSN" --json "$@"; }

step "1/8 temporary PostgreSQL ($PG_BIN, socket $PGHOST)"
"$PG_BIN/initdb" -D "$WORK/pg/data" -U postgres --auth=trust -E UTF8 >/dev/null
"$PG_BIN/pg_ctl" -D "$WORK/pg/data" -o "-p $PGPORT -k $PGHOST -c listen_addresses=''" -l "$WORK/postgres.log" -w start >/dev/null
sql -d postgres -c 'CREATE DATABASE app'

step "2/8 saascore schema from $LEGACY_REV, sample data, a token signed by saascore"
mkdir -p "$WORK/legacy"
git -C "$ROOT" archive "$LEGACY_REV" | tar -x -C "$WORK/legacy"
mkdir -p "$WORK/legacy/cmd/oldside"
cp "$HERE/oldside/main.go" "$WORK/legacy/cmd/oldside/main.go"
oldside schema -dsn "$DSN" -secret "$JWT_SECRET" -service "$SERVICE"
sql -f "$HERE/seed.sql"
LEGACY_TOKEN=$(oldside token -secret "$JWT_SECRET" -service "$SERVICE" \
	-user-id 00000000-0000-4000-8000-0000000000a1 -email alice@example.com)
sql -At -F $'\t' -c "SELECT email, id, plan, credits FROM users ORDER BY email" >"$WORK/legacy_users.tsv"
echo "$(wc -l <"$WORK/legacy_users.tsv" | tr -d ' ') legacy users loaded"

step "3/8 guide step 2: snapshot"
sql -At -f "$GUIDE/snapshot.sql" >"$WORK/before.txt"
cat "$WORK/before.txt"

step "4/8 guide step 3: start the new application once so it creates its tables"
(cd "$ROOT/templates/quickstart" && go build -o "$WORK/quickstart" ./cmd/quickstart)
cp "$ROOT/templates/quickstart/deploy/config.yaml.example" "$WORK/config.yaml"
start_app
stop_app
echo "AutoMigrate finished on the legacy schema"

step "5/8 guide step 4: Stripe columns into billing tables"
(cd "$ROOT" && go build -o "$WORK/legacy-billing-migrate" ./cmd/legacy-billing-migrate)
# check exits 2 while issues remain; the JSON report is still written.
billing --mode check >"$WORK/billing_check_before.json" || true
issues=$(jq '.issues | length' "$WORK/billing_check_before.json")
echo "check before backfill: $issues issues"
billing --mode backfill | tee "$WORK/billing_backfill.json"
billing --mode check >"$WORK/billing_check_after.json" || true
issues=$(jq '.issues | length' "$WORK/billing_check_after.json")
echo "check after backfill: $issues issues"
# Expected leftovers: saascore kept the subscription id in billing_subscriptions
# when a subscription was canceled but cleared it on users. Both sides say
# canceled, so nothing needs fixing; anything else is a real difference.
while read -r id; do
	[ -n "$id" ] || continue
	state=$(sql -At -c "SELECT u.billing_status || '/' || COALESCE(s.status, '') FROM users u
		LEFT JOIN billing_subscriptions s ON s.user_id = u.id AND s.provider = 'stripe' WHERE u.id = '$id'")
	if [ "$state" = canceled/canceled ]; then
		echo "ignored: $id differs only in fields of a canceled subscription"
	else
		fail "billing check issue for $id ($state)"
	fi
done < <(jq -r '.issues[]?.user_id' "$WORK/billing_check_after.json" | sort -u)

step "6/8 guide steps 5-7: identities, credit ledger, manual membership"
sql -f "$GUIDE/user_identities.sql"
sql -f "$GUIDE/credit_ledger.sql"
sql -f "$HERE/overrides.sql"
echo "rerunning both files must change nothing"
sql -f "$GUIDE/user_identities.sql"
sql -f "$GUIDE/credit_ledger.sql"

step "7/8 guide step 8: verify before resuming writes"
sql -At -f "$GUIDE/snapshot.sql" >"$WORK/after.txt"
if diff -u "$WORK/before.txt" "$WORK/after.txt"; then echo "snapshot unchanged"; else fail "legacy data changed"; fi
sql -At -F ' | ' -f "$GUIDE/verify.sql" >"$WORK/verify.txt"
cat "$WORK/verify.txt"
problems=$(awk '/^== 5\./{exit} !/^== /' "$WORK/verify.txt")
if [ -n "$problems" ]; then fail "verify.sql sections 1-4 are not empty"; fi
# Expected: ivy's last payment failed. saascore left users.plan = pro; the
# template grants no access while payment_failed (no grace period by default).
plan_changes=$(awk -F' [|] ' 'f{print $1} /^== 5\./{f=1}' "$WORK/verify.txt" | sort | tr '\n' ' ')
if [ "$plan_changes" != "ivy@example.com " ]; then fail "unexpected plan changes: $plan_changes"; fi

step "8/8 guide step 9: start the application and sign in as every sample user"
start_app
status=$(curl -s -o /dev/null -w '%{http_code}' -H "Authorization: Bearer $LEGACY_TOKEN" \
	"http://127.0.0.1:$APP_PORT/api/v1/account/profile")
if [ "$status" = 401 ]; then
	echo "token signed by saascore -> HTTP 401: every user signs in again"
else
	fail "token signed by saascore returned HTTP $status, expected 401"
fi

token_of() { cat "$WORK/token.$1"; }
printf '\n%-19s %-9s %-9s %-9s %-7s %-7s %-6s %s\n' email plan_old plan_new expected cr_old cr_new role same_account
while IFS=$'\t' read -r email want_plan want_credits want_role; do
	IFS=$'\t' read -r _ old_id old_plan old_credits < <(awk -F'\t' -v e="$email" '$1 == e' "$WORK/legacy_users.tsv")
	token=$(login "$email")
	echo "$token" >"$WORK/token.$email"
	profile=$(api GET /account/profile "$token")
	new_id=$(jq -r '.data.id' <<<"$profile")
	role=$(jq -r '.data.role' <<<"$profile")
	plan=$(api GET /account/membership "$token" | jq -r '.data.plan')
	credits=$(api GET /credits "$token" | jq -r '.data.balance')
	same=no
	if [ "$new_id" = "$old_id" ]; then same=yes; fi
	printf '%-19s %-9s %-9s %-9s %-7s %-7s %-6s %s\n' "$email" "$old_plan" "$plan" "$want_plan" "$old_credits" "$credits" "$role" "$same"
	if [ "$same" != yes ] || [ "$plan" != "$want_plan" ] || [ "$credits" != "$want_credits" ] || [ "$role" != "$want_role" ]; then
		fail "$email: got plan=$plan credits=$credits role=$role same_account=$same"
	fi
done < <(tail -n +2 "$HERE/expected.tsv")

echo
code=$(api GET /referral/code "$(token_of erin@example.com)")
if grep -q INVERIN0001 <<<"$code"; then echo "erin keeps referral code INVERIN0001"; else fail "erin's referral code: $code"; fi
opening=$(api GET /credits/transactions "$(token_of carol@example.com)" | jq -c '[.data.list[] | {kind, amount}]')
if [ "$opening" = '[{"kind":"opening","amount":500}]' ]; then echo "carol's history: one opening entry of 500"; else fail "carol's history: $opening"; fi
grant=$(api POST /admin/credits/grants "$(token_of root@example.com)" \
	'{"user_id":"00000000-0000-4000-8000-0000000000a3","amount":5,"reason":"rehearsal grant","idempotency_key":"rehearsal-grant-1"}')
balance=$(api GET /credits "$(token_of carol@example.com)" | jq -r '.data.balance')
if [ "$balance" = 505 ]; then echo "admin grant after the upgrade works: carol now has 505"; else fail "grant: $grant, balance $balance"; fi
identity=$(sql -At -c "SELECT provider || ':' || subject FROM user_identities WHERE user_id = '00000000-0000-4000-8000-0000000000a4'")
if [ "$identity" = google:google-sub-dave-0001 ]; then echo "dave's Google identity is linked"; else fail "dave's identity: $identity"; fi

echo
if [ "$FAILURES" -eq 0 ]; then
	echo "REHEARSAL PASSED"
else
	echo "REHEARSAL FAILED: $FAILURES problem(s); rerun with KEEP=1 to inspect $WORK"
	exit 1
fi
