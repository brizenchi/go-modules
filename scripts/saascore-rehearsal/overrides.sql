-- The operator decision for the rehearsal's one manual grant (step 7 of the
-- guide): grace keeps premium until a far-future date.
BEGIN;
INSERT INTO membership_overrides(user_id, plan, expires_at, updated_at) VALUES
  ('00000000-0000-4000-8000-0000000000a7', 'premium', '2099-12-31T00:00:00Z', now())
ON CONFLICT (user_id) DO NOTHING;
COMMIT;
