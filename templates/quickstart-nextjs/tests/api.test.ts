import assert from "node:assert/strict";
import { afterEach, test } from "node:test";

const ENV_KEYS = [
  "NEXT_PUBLIC_APP_URL",
  "NEXT_PUBLIC_API_BASE_URL",
  "NEXT_PUBLIC_APP_NAME",
  "NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY",
  "NEXT_PUBLIC_DEFAULT_PLAN",
  "NEXT_PUBLIC_DEFAULT_INTERVAL",
  "NEXT_PUBLIC_DEFAULT_CREDITS_QUANTITY",
  "NEXT_PUBLIC_DEFAULT_TOPUP_AMOUNT_USD",
  "NEXT_PUBLIC_CREDITS_PRICE_ID",
  "NEXT_PUBLIC_STRIPE_SUCCESS_PATH",
  "NEXT_PUBLIC_STRIPE_CANCEL_PATH"
] as const;

const originalFetch = globalThis.fetch;

function loadApiModule(overrides: Partial<Record<(typeof ENV_KEYS)[number], string>> = {}) {
  for (const key of ENV_KEYS) {
    delete process.env[key];
  }
  process.env.NEXT_PUBLIC_API_BASE_URL = "https://api.example.com/api/v1/";
  for (const [key, value] of Object.entries(overrides)) {
    if (value !== undefined) {
      process.env[key] = value;
    }
  }

  for (const mod of ["../lib/api", "../lib/env", "../lib/auth"]) {
    const modPath = require.resolve(mod);
    delete require.cache[modPath];
  }

  return require("../lib/api") as typeof import("../lib/api");
}

function jsonResponse(status: number, body: unknown) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "content-type": "application/json" }
  });
}

afterEach(() => {
  globalThis.fetch = originalFetch;
});

test("apiRequest builds auth and json headers against normalized api base url", async () => {
  const api = loadApiModule();
  let requestURL = "";
  let requestInit: RequestInit | undefined;

  globalThis.fetch = (async (input: string | URL | Request, init?: RequestInit) => {
    requestURL = typeof input === "string" ? input : String(input);
    requestInit = init;
    return jsonResponse(200, {
      code: 200,
      msg: "ok",
      data: { ok: true }
    });
  }) as typeof fetch;

  const data = await api.apiRequest<{ ok: boolean }>("/demo", {
    method: "POST",
    authToken: "jwt-token",
    json: { hello: "world" }
  });

  const headers = new Headers(requestInit?.headers);

  assert.deepEqual(data, { ok: true });
  assert.equal(requestURL, "https://api.example.com/api/v1/demo");
  assert.equal(headers.get("Accept"), "application/json");
  assert.equal(headers.get("Content-Type"), "application/json");
  assert.equal(headers.get("Authorization"), "Bearer jwt-token");
  assert.equal(requestInit?.body, JSON.stringify({ hello: "world" }));
});

test("apiRequest throws ApiError on transport or envelope failure", async () => {
  const api = loadApiModule();

  globalThis.fetch = (async () =>
    jsonResponse(200, {
      code: 4001,
      msg: "provider disabled",
      data: { reason: "stripe_disabled" }
    })) as typeof fetch;

  await assert.rejects(
    api.apiRequest("/stripe/checkout/session"),
    (error: unknown) => {
      assert.ok(error instanceof api.ApiError);
      assert.equal(error.status, 200);
      assert.equal(error.code, 4001);
      assert.equal(error.message, "provider disabled");
      return true;
    }
  );
});

test("verifyCode forwards a trimmed referral code", async () => {
  const api = loadApiModule();
  let payload: unknown;

  globalThis.fetch = (async (_input: string | URL | Request, init?: RequestInit) => {
    payload = JSON.parse(String(init?.body));
    return jsonResponse(200, {
      code: 200,
      msg: "ok",
      data: {
        token: "jwt-token",
        expires_at: "2030-01-01T00:00:00Z",
        user: { id: "user_1", email: "user@example.com" }
      }
    });
  }) as typeof fetch;

  await api.verifyCode("user@example.com", "123456", " REF123 ");

  assert.deepEqual(payload, {
    email: "user@example.com",
    code: "123456",
    referral_code: "REF123"
  });
});

test("createTopUpPaymentIntent hits the custom top-up endpoint", async () => {
  const api = loadApiModule();
  let requestURL = "";
  let requestBody: unknown;

  globalThis.fetch = (async (input: string | URL | Request, init?: RequestInit) => {
    requestURL = typeof input === "string" ? input : String(input);
    requestBody = init?.body ? JSON.parse(String(init.body)) : null;
    return jsonResponse(200, {
      code: 200,
      msg: "ok",
      data: {
        payment_intent_id: "pi_123",
        client_secret: "pi_123_secret_456",
        amount_cents: 2500,
        amount_usd: 25,
        currency: "usd",
        credits: 2500
      }
    });
  }) as typeof fetch;

  const result = await api.createTopUpPaymentIntent("jwt-token", {
    amount: 25,
    metadata: { referral_code: "INV-123" }
  });

  assert.equal(requestURL, "https://api.example.com/api/v1/stripe/topup/payment-intent");
  assert.deepEqual(requestBody, {
    amount: 25,
    metadata: { referral_code: "INV-123" }
  });
  assert.deepEqual(result, {
    payment_intent_id: "pi_123",
    client_secret: "pi_123_secret_456",
    amount_cents: 2500,
    amount_usd: 25,
    currency: "usd",
    credits: 2500
  });
});

test("getSubscription maps provider payment fields into frontend shape", async () => {
  const api = loadApiModule();

  globalThis.fetch = (async () =>
    jsonResponse(200, {
      code: 200,
      msg: "ok",
      data: {
        plan: "pro",
        status: "active",
        billing_cycle: "monthly",
        current_period_end: "2030-02-01T00:00:00Z",
        cancel_at_period_end: false,
        payment_method: {
          Brand: "visa",
          Last4: "4242",
          ExpMonth: 1,
          ExpYear: 2030
        }
      }
    })) as typeof fetch;

  const subscription = await api.getSubscription("jwt-token");

  assert.deepEqual(subscription, {
    plan: "pro",
    status: "active",
    billing_cycle: "monthly",
    current_period_end: "2030-02-01T00:00:00Z",
    cancel_at_period_end: false,
    payment_method: {
      brand: "visa",
      last4: "4242",
      exp_month: 1,
      exp_year: 2030
    }
  });
});

test("preview/change subscription and billing portal hit the new billing endpoints", async () => {
  const api = loadApiModule();
  const calls: Array<{ url: string; body: unknown }> = [];

  globalThis.fetch = (async (input: string | URL | Request, init?: RequestInit) => {
    const url = typeof input === "string" ? input : String(input);
    const body = init?.body ? JSON.parse(String(init.body)) : null;
    calls.push({ url, body });

    if (url.endsWith("/stripe/subscription/preview")) {
      return jsonResponse(200, {
        code: 200,
        msg: "ok",
        data: {
          currency: "usd",
          amount_due_now: 30,
          current_period_end: "2030-02-01T00:00:00Z",
          next_billing_at: "2030-02-01T00:00:00Z",
          target_plan: "pro",
          target_interval: "monthly",
          change_mode: "immediate_prorated",
          immediate_charge: true,
          effective_at_period_end: false,
          message: "preview ready"
        }
      });
    }

    if (url.endsWith("/stripe/subscription/change")) {
      return jsonResponse(200, {
        code: 200,
        msg: "ok",
        data: {
          status: "active",
          plan: "pro",
          billing_cycle: "monthly",
          change_mode: "immediate_prorated",
          provider_subscription_id: "sub_1",
          message: "subscription changed"
        }
      });
    }

    return jsonResponse(200, {
      code: 200,
      msg: "ok",
      data: {
        url: "https://billing.stripe.test/session_123"
      }
    });
  }) as typeof fetch;

  const preview = await api.previewSubscriptionChange("jwt-token", {
    plan: "pro",
    interval: "monthly"
  });
  const change = await api.changeSubscription("jwt-token", {
    plan: "pro",
    interval: "monthly",
    change_mode: "immediate_prorated"
  });
  const portal = await api.createBillingPortalSession("jwt-token", "https://app.example.com/billing");

  assert.equal(calls[0]?.url, "https://api.example.com/api/v1/stripe/subscription/preview");
  assert.deepEqual(calls[0]?.body, {
    plan: "pro",
    interval: "monthly"
  });
  assert.equal(calls[1]?.url, "https://api.example.com/api/v1/stripe/subscription/change");
  assert.deepEqual(calls[1]?.body, {
    plan: "pro",
    interval: "monthly",
    change_mode: "immediate_prorated"
  });
  assert.equal(calls[2]?.url, "https://api.example.com/api/v1/stripe/portal/session");
  assert.deepEqual(calls[2]?.body, {
    return_url: "https://app.example.com/billing"
  });
  assert.equal(preview.change_mode, "immediate_prorated");
  assert.equal(change.message, "subscription changed");
  assert.equal(portal.url, "https://billing.stripe.test/session_123");
});

test("invoice, referral, and user helpers map backend payloads correctly", async () => {
  const api = loadApiModule();
  let callIndex = 0;

  globalThis.fetch = (async () => {
    callIndex += 1;
    if (callIndex === 1) {
      return jsonResponse(200, {
        code: 200,
        msg: "ok",
        data: {
          items: [
            {
              ID: "inv_1",
              AmountUSD: 29,
              Status: "paid",
              Period: "2026-05",
              PDFURL: "https://example.com/invoice.pdf",
              CreatedAt: "2030-01-01T00:00:00Z"
            }
          ],
          total: 1,
          page: 1,
          limit: 10
        }
      });
    }
    if (callIndex === 2) {
      return jsonResponse(200, {
        code: 200,
        msg: "ok",
        data: {
          TotalReferred: 3,
          Activated: 2,
          Pending: 1,
          TotalRewardCredits: 80
        }
      });
    }
    return jsonResponse(200, {
      code: 200,
      msg: "ok",
      data: {
        items: [
          {
            ID: 10,
            Code: "REF123",
            ReferrerID: "user_ref",
            RefereeID: "user_new",
            Status: "activated",
            ActivatedAt: "2030-01-05T00:00:00Z",
            ExpiresAt: null,
            RewardCredits: 50,
            CreatedAt: "2030-01-01T00:00:00Z",
            UpdatedAt: "2030-01-05T00:00:00Z"
          }
        ],
        total: 1,
        page: 1,
        limit: 20
      }
    });
  }) as typeof fetch;

  const invoices = await api.listInvoices("jwt-token");
  const stats = await api.getReferralStats("jwt-token");
  const referrals = await api.listReferrals("jwt-token");

  assert.deepEqual(invoices, {
    items: [
      {
        id: "inv_1",
        amount_usd: 29,
        status: "paid",
        period: "2026-05",
        pdf_url: "https://example.com/invoice.pdf",
        created_at: "2030-01-01T00:00:00Z"
      }
    ],
    total: 1,
    page: 1,
    limit: 10
  });

  assert.deepEqual(stats, {
    total_referred: 3,
    activated: 2,
    pending: 1,
    total_reward_credits: 80
  });

  assert.deepEqual(referrals, {
    items: [
      {
        id: 10,
        code: "REF123",
        referrer_id: "user_ref",
        referee_id: "user_new",
        status: "activated",
        activated_at: "2030-01-05T00:00:00Z",
        expires_at: null,
        reward_credits: 50,
        created_at: "2030-01-01T00:00:00Z",
        updated_at: "2030-01-05T00:00:00Z"
      }
    ],
    total: 1,
    page: 1,
    limit: 20
  });

  assert.equal(api.userLabel({ id: "user_1", email: "user@example.com", username: "alice" }), "alice");
  assert.equal(api.userLabel({ id: "user_2", email: "user@example.com" }), "user@example.com");
  assert.equal(api.userLabel({ id: "user_3", email: "" }), "user_3");
  assert.equal(api.userLabel(null), "anonymous");
});
