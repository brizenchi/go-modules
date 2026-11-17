-- Fingerprint of the data the saascore upgrade must not change. Run it on the
-- legacy database before step 3 and again after the last SQL step, both while
-- application writes are stopped, then compare:
--
--   psql "$DATABASE_URL" -At -f docs/migrate-from-saascore/snapshot.sql > before.txt
--   ... upgrade steps ...
--   psql "$DATABASE_URL" -At -f docs/migrate-from-saascore/snapshot.sql > after.txt
--   diff before.txt after.txt     # no output means nothing changed
--
-- Read-only. Only legacy columns are hashed, so new columns do not count as changes.

SELECT 'users', count(*), md5(COALESCE(string_agg(concat_ws('|',
  id, email, role, plan, credits, COALESCE(stripe_customer_id, ''), COALESCE(provider, ''), COALESCE(provider_subject, '')),
  ',' ORDER BY id), ''))
FROM users;

SELECT 'credits_total', COALESCE(sum(credits), 0) FROM users;

SELECT 'referral_codes', count(*), md5(COALESCE(string_agg(concat_ws('|', user_id, value), ',' ORDER BY user_id), ''))
FROM referral_codes;

SELECT 'referrals', count(*), md5(COALESCE(string_agg(concat_ws('|',
  id, code, referrer_id, referee_id, status, activated_at, expires_at, reward_credits), ',' ORDER BY id), ''))
FROM referrals;

SELECT 'billing_events', count(*) FROM billing_events;
