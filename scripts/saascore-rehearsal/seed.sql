-- Sample data for the saascore upgrade rehearsal, written the way saascore
-- (d6a572f) stored it. Loaded into a schema created by oldside/main.go.
-- Every e-mail address is fake; every provider id is a test id.
--
--   alice  Stripe pro, written to users AND billing tables (saascore dual-write)
--   bob    Stripe starter, canceling at period end, users columns only (older data)
--   carol  free, 500 credits
--   dave   signed up with Google (provider/provider_subject on users)
--   erin   invited frank; the referral is activated with a reward
--   frank  invited by erin
--   grace  premium granted by hand (users.plan only, no Stripe data)
--   henry  Stripe lifetime purchase, users columns only
--   ivy    Stripe pro whose last payment failed (dual-write)
--   jack   Stripe subscription canceled (dual-write)
--   root   admin role
BEGIN;

INSERT INTO users (id, email, email_verified, username, provider, provider_subject, role, plan, credits,
  stripe_customer_id, stripe_subscription_id, stripe_price_id, stripe_product_id, billing_status,
  billing_period_start, billing_period_end, cancel_effective_at, created_at, updated_at) VALUES
('00000000-0000-4000-8000-0000000000a1', 'alice@example.com', true, 'alice', 'email', 'alice@example.com', 'user', 'pro', 120,
  'cus_test_alice', 'sub_test_alice', 'price_test_pro_monthly', 'prod_test_pro', 'active',
  now() - interval '10 days', now() + interval '20 days', NULL, now() - interval '200 days', now() - interval '10 days'),
('00000000-0000-4000-8000-0000000000a2', 'bob@example.com', true, 'bob', 'email', 'bob@example.com', 'user', 'starter', 0,
  'cus_test_bob', 'sub_test_bob', 'price_test_starter_monthly', 'prod_test_starter', 'canceling',
  now() - interval '5 days', now() + interval '25 days', now() + interval '25 days', now() - interval '300 days', now() - interval '5 days'),
('00000000-0000-4000-8000-0000000000a3', 'carol@example.com', true, 'carol', 'email', 'carol@example.com', 'user', 'free', 500,
  NULL, NULL, NULL, NULL, '', NULL, NULL, NULL, now() - interval '100 days', now() - interval '1 day'),
('00000000-0000-4000-8000-0000000000a4', 'dave@example.com', true, 'Dave', 'google', 'google-sub-dave-0001', 'user', 'free', 30,
  NULL, NULL, NULL, NULL, '', NULL, NULL, NULL, now() - interval '90 days', now() - interval '2 days'),
('00000000-0000-4000-8000-0000000000a5', 'erin@example.com', true, 'erin', 'email', 'erin@example.com', 'user', 'free', 70,
  NULL, NULL, NULL, NULL, '', NULL, NULL, NULL, now() - interval '80 days', now() - interval '3 days'),
('00000000-0000-4000-8000-0000000000a6', 'frank@example.com', true, 'frank', 'email', 'frank@example.com', 'user', 'free', 10,
  NULL, NULL, NULL, NULL, '', NULL, NULL, NULL, now() - interval '60 days', now() - interval '3 days'),
('00000000-0000-4000-8000-0000000000a7', 'grace@example.com', true, 'grace', 'email', 'grace@example.com', 'user', 'premium', 0,
  NULL, NULL, NULL, NULL, '', NULL, NULL, NULL, now() - interval '50 days', now() - interval '4 days'),
('00000000-0000-4000-8000-0000000000a8', 'henry@example.com', true, 'henry', 'email', 'henry@example.com', 'user', 'lifetime', 0,
  'cus_test_henry', '', 'price_test_lifetime', 'prod_test_lifetime', 'active',
  NULL, NULL, NULL, now() - interval '400 days', now() - interval '390 days'),
('00000000-0000-4000-8000-0000000000a9', 'ivy@example.com', true, 'ivy', 'email', 'ivy@example.com', 'user', 'pro', 15,
  'cus_test_ivy', 'sub_test_ivy', 'price_test_pro_monthly', 'prod_test_pro', 'payment_failed',
  now() - interval '2 days', now() + interval '28 days', NULL, now() - interval '150 days', now() - interval '1 day'),
('00000000-0000-4000-8000-0000000000b0', 'jack@example.com', true, 'jack', 'email', 'jack@example.com', 'user', 'free', 0,
  'cus_test_jack', '', '', '', 'canceled',
  NULL, NULL, NULL, now() - interval '250 days', now() - interval '20 days'),
('00000000-0000-4000-8000-0000000000b1', 'root@example.com', true, 'root', 'email', 'root@example.com', 'admin', 'free', 0,
  NULL, NULL, NULL, NULL, '', NULL, NULL, NULL, now() - interval '500 days', now() - interval '1 day');

-- saascore mirrored newer subscription events into billing-owned tables.
INSERT INTO billing_customers (user_id, provider, provider_customer_id, created_at, updated_at) VALUES
('00000000-0000-4000-8000-0000000000a1', 'stripe', 'cus_test_alice', now() - interval '40 days', now() - interval '40 days'),
('00000000-0000-4000-8000-0000000000a9', 'stripe', 'cus_test_ivy', now() - interval '40 days', now() - interval '40 days'),
('00000000-0000-4000-8000-0000000000b0', 'stripe', 'cus_test_jack', now() - interval '40 days', now() - interval '40 days');

INSERT INTO billing_subscriptions (user_id, provider, provider_customer_id, provider_subscription_id, provider_price_id,
  provider_product_id, product_type, plan, billing_interval, status, cancel_at_period_end, period_start, period_end,
  cancel_effective_at, raw_snapshot_json, created_at, updated_at) VALUES
('00000000-0000-4000-8000-0000000000a1', 'stripe', 'cus_test_alice', 'sub_test_alice', 'price_test_pro_monthly',
  'prod_test_pro', 'subscription', 'pro', 'monthly', 'active', false, now() - interval '10 days', now() + interval '20 days',
  NULL, '{}', now() - interval '40 days', now() - interval '10 days'),
('00000000-0000-4000-8000-0000000000a9', 'stripe', 'cus_test_ivy', 'sub_test_ivy', 'price_test_pro_monthly',
  'prod_test_pro', 'subscription', 'pro', 'monthly', 'payment_failed', false, now() - interval '2 days', now() + interval '28 days',
  NULL, '{}', now() - interval '40 days', now() - interval '1 day'),
-- onSubscriptionCanceled kept the subscription id here but cleared it on users.
('00000000-0000-4000-8000-0000000000b0', 'stripe', 'cus_test_jack', 'sub_test_jack', '',
  '', '', 'free', '', 'canceled', false, NULL, NULL,
  NULL, '{}', now() - interval '40 days', now() - interval '20 days');

INSERT INTO billing_events (user_id, provider, stripe_event_id, event_type, payload, processed, processed_at, created_at, updated_at) VALUES
('00000000-0000-4000-8000-0000000000a1', 'stripe', 'evt_test_alice_renewal', 'invoice.paid', '{}', true, now() - interval '10 days', now() - interval '10 days', now() - interval '10 days');

INSERT INTO referral_codes (user_id, value, created_at) VALUES
('00000000-0000-4000-8000-0000000000a5', 'INVERIN0001', now() - interval '80 days');

INSERT INTO referrals (code, referrer_id, referee_id, status, activated_at, expires_at, reward_credits, created_at, updated_at) VALUES
('INVERIN0001', '00000000-0000-4000-8000-0000000000a5', '00000000-0000-4000-8000-0000000000a6', 'activated',
  now() - interval '30 days', NULL, 20, now() - interval '60 days', now() - interval '30 days');

COMMIT;
