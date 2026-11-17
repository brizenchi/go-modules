-- Links legacy OAuth logins to user_identities for a database created by
-- stacks/saascore (go-modules d6a572f or earlier). See docs/MIGRATE_FROM_SAASCORE.md.
--
-- saascore kept one external identity per user in users.provider /
-- users.provider_subject and matched OAuth logins by subject first. The
-- template matches by user_identities first and falls back to email, so
-- without this file a user whose Google email changed after signup would get
-- a new, empty account. Email-code users (provider 'email') need no row.
--
-- Safe to rerun: existing links are kept and conflicts are skipped. Run while
-- application writes are stopped; it does not modify users.
BEGIN;
LOCK TABLE users IN SHARE ROW EXCLUSIVE MODE;
SET LOCAL client_min_messages = warning; -- hide "already exists, skipping"

CREATE TABLE IF NOT EXISTS user_identities (
  id bigserial PRIMARY KEY,
  user_id varchar(36) NOT NULL,
  provider varchar(32) NOT NULL,
  subject varchar(255) NOT NULL,
  created_at timestamptz,
  updated_at timestamptz
);
CREATE UNIQUE INDEX IF NOT EXISTS uniq_user_identity_provider ON user_identities(user_id, provider);
CREATE UNIQUE INDEX IF NOT EXISTS uniq_provider_subject ON user_identities(provider, subject);
CREATE INDEX IF NOT EXISTS idx_user_identities_user_id ON user_identities(user_id);
SET LOCAL client_min_messages = notice;

DO $$
DECLARE
  candidates bigint;
  linked bigint;
BEGIN
  SELECT count(*) INTO candidates FROM users
  WHERE COALESCE(TRIM(provider), '') NOT IN ('', 'email') AND COALESCE(TRIM(provider_subject), '') <> '';

  INSERT INTO user_identities(user_id, provider, subject, created_at, updated_at)
  SELECT id, TRIM(provider), TRIM(provider_subject), COALESCE(created_at, now()), now()
  FROM users
  WHERE COALESCE(TRIM(provider), '') NOT IN ('', 'email') AND COALESCE(TRIM(provider_subject), '') <> ''
  ON CONFLICT DO NOTHING;

  SELECT count(*) INTO linked FROM users u
  JOIN user_identities i ON i.user_id = u.id AND i.provider = TRIM(u.provider) AND i.subject = TRIM(u.provider_subject)
  WHERE COALESCE(TRIM(u.provider), '') NOT IN ('', 'email') AND COALESCE(TRIM(u.provider_subject), '') <> '';

  RAISE NOTICE 'legacy oauth users: %, linked: %', candidates, linked;
  IF linked <> candidates THEN
    RAISE WARNING '% legacy oauth users could not be linked (subject already linked to another user, or the user already has another identity for that provider); see verify.sql', candidates - linked;
  END IF;
END $$;
COMMIT;
