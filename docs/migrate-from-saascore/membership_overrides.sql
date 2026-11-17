-- Manual memberships for a saascore database upgraded to the template.
-- See docs/MIGRATE_FROM_SAASCORE.md, step 7.
--
-- saascore read access from users.plan, so an operator could grant a plan by
-- editing that column. The template derives access from billing_subscriptions
-- and lets operators override it in membership_overrides. Users granted a plan
-- by hand therefore lose it unless you add an override.
--
-- Template rules (internal/feature/membership.Resolve):
--   lifetime            permanent when expires_at is NULL
--   starter/pro/premium require expires_at; a NULL expiry is ignored
--   free                removes access
--
-- 1. List candidates: paid legacy plan, no Stripe data at all.
SELECT id, email, plan, created_at
FROM users
WHERE COALESCE(NULLIF(TRIM(LOWER(plan)), ''), 'free') <> 'free'
  AND COALESCE(stripe_customer_id, '') = ''
  AND COALESCE(stripe_subscription_id, '') = ''
  AND NOT EXISTS (SELECT 1 FROM billing_subscriptions s WHERE s.user_id = users.id)
ORDER BY email LIMIT 500;

-- 2. Decide each row with the product owner, then insert explicitly, e.g.:
--
-- BEGIN;
-- INSERT INTO membership_overrides(user_id, plan, expires_at, updated_at) VALUES
--   ('<user id>', 'pro',      '2027-12-31T00:00:00Z', now()),  -- time-limited grant
--   ('<user id>', 'lifetime', NULL,                   now())   -- permanent grant
-- ON CONFLICT (user_id) DO NOTHING;
-- COMMIT;
--
-- Once the application is live, prefer the admin UI (PATCH
-- /api/v1/admin/users/:id/membership), which also writes an audit record.
