-- Post-migration checks for a saascore database upgraded to the template.
-- Run with psql after every step in docs/MIGRATE_FROM_SAASCORE.md has finished
-- and BEFORE application writes resume:
--
--   psql "$DATABASE_URL" -f docs/migrate-from-saascore/verify.sql
--
-- Read-only. Each section prints the rows that need attention; an empty
-- section is a pass. Legacy users columns (plan, stripe_*, provider) are still
-- present because AutoMigrate never drops columns, which is what makes these
-- comparisons possible.

\echo '== 1. users whose credits were not moved to the ledger (expect none)'
SELECT id, email, credits, credits_version
FROM users WHERE credits_version <> 1
ORDER BY email LIMIT 100;

\echo '== 2. users.credits differs from the ledger balance (expect none)'
SELECT u.id, u.email, u.credits AS legacy_credits, COALESCE(l.remaining, 0) AS ledger_credits
FROM users u
LEFT JOIN (SELECT user_id, sum(remaining) AS remaining FROM user_credit_lots GROUP BY user_id) l ON l.user_id = u.id
WHERE u.credits <> COALESCE(l.remaining, 0)
ORDER BY u.email LIMIT 100;

\echo '== 3. Stripe customers missing from billing_customers (expect none; rerun legacy-billing-migrate --mode backfill)'
SELECT u.id, u.email, u.stripe_customer_id
FROM users u
WHERE COALESCE(u.stripe_customer_id, '') <> ''
  AND NOT EXISTS (
    SELECT 1 FROM billing_customers c
    WHERE c.user_id = u.id AND c.provider = 'stripe' AND c.provider_customer_id = u.stripe_customer_id)
ORDER BY u.email LIMIT 100;

\echo '== 4. OAuth users without a user_identities link (expect none; see user_identities.sql warnings)'
SELECT u.id, u.email, u.provider, u.provider_subject
FROM users u
WHERE COALESCE(TRIM(u.provider), '') NOT IN ('', 'email') AND COALESCE(TRIM(u.provider_subject), '') <> ''
  AND NOT EXISTS (
    SELECT 1 FROM user_identities i
    WHERE i.user_id = u.id AND i.provider = TRIM(u.provider) AND i.subject = TRIM(u.provider_subject))
ORDER BY u.email LIMIT 100;

\echo '== 5. legacy users.plan differs from the plan the template will grant (review every row)'
-- Mirrors the template defaults in internal/feature/membership (Evaluate and
-- Resolve). If you changed Evaluate, change this query the same way.
WITH sub AS (
  SELECT DISTINCT ON (user_id) user_id, plan, status, period_end, cancel_effective_at,
    CASE WHEN cancel_effective_at IS NOT NULL AND (period_end IS NULL OR cancel_effective_at < period_end)
      THEN cancel_effective_at ELSE period_end END AS valid_until
  FROM billing_subscriptions WHERE provider = 'stripe'
  ORDER BY user_id, id
),
billing AS (
  SELECT u.id AS user_id,
    CASE
      WHEN s.user_id IS NULL OR s.status NOT IN ('active', 'trialing', 'canceling') THEN 'free'
      WHEN s.plan = 'lifetime' AND s.status = 'active' THEN 'lifetime'
      WHEN s.plan NOT IN ('starter', 'pro', 'premium') THEN 'free'
      WHEN s.valid_until IS NULL OR now() >= s.valid_until THEN 'free'
      ELSE s.plan
    END AS plan
  FROM users u LEFT JOIN sub s ON s.user_id = u.id
),
effective AS (
  SELECT b.user_id,
    CASE
      WHEN o.user_id IS NULL OR (o.expires_at IS NOT NULL AND now() >= o.expires_at) THEN b.plan
      WHEN o.plan IN ('free', 'lifetime') THEN o.plan
      WHEN o.plan IN ('starter', 'pro', 'premium') AND o.expires_at IS NOT NULL THEN o.plan
      ELSE b.plan
    END AS plan
  FROM billing b LEFT JOIN membership_overrides o ON o.user_id = b.user_id
)
SELECT u.email, u.plan AS legacy_plan, e.plan AS new_plan,
  s.status AS billing_status, s.valid_until AS billing_valid_until,
  CASE
    WHEN s.user_id IS NULL AND COALESCE(u.stripe_customer_id, '') = '' THEN 'no Stripe data: manual grant? add membership_overrides'
    WHEN s.user_id IS NULL THEN 'no billing_subscriptions row: run legacy-billing-migrate --mode backfill'
    WHEN s.status NOT IN ('active', 'trialing', 'canceling') THEN 'billing status ' || s.status || ' grants no access by default'
    WHEN s.valid_until IS NULL OR now() >= s.valid_until THEN 'paid period ended or unknown'
    ELSE 'review'
  END AS hint
FROM users u
JOIN effective e ON e.user_id = u.id
LEFT JOIN sub s ON s.user_id = u.id
WHERE COALESCE(NULLIF(TRIM(LOWER(u.plan)), ''), 'free') <> e.plan
ORDER BY u.email LIMIT 500;
