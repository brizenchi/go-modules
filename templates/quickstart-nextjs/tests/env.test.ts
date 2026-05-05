import assert from "node:assert/strict";
import { test } from "node:test";

const ENV_KEYS = [
  "NEXT_PUBLIC_APP_NAME",
  "NEXT_PUBLIC_APP_URL",
  "NEXT_PUBLIC_API_BASE_URL",
  "NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY",
  "NEXT_PUBLIC_DEFAULT_PLAN",
  "NEXT_PUBLIC_DEFAULT_INTERVAL",
  "NEXT_PUBLIC_DEFAULT_CREDITS_QUANTITY",
  "NEXT_PUBLIC_DEFAULT_TOPUP_AMOUNT_USD",
  "NEXT_PUBLIC_CREDITS_PRICE_ID",
  "NEXT_PUBLIC_STRIPE_SUCCESS_PATH",
  "NEXT_PUBLIC_STRIPE_CANCEL_PATH"
] as const;

function loadEnvModule(overrides: Partial<Record<(typeof ENV_KEYS)[number], string>>) {
  for (const key of ENV_KEYS) {
    delete process.env[key];
  }
  for (const [key, value] of Object.entries(overrides)) {
    if (value !== undefined) {
      process.env[key] = value;
    }
  }

  const modPath = require.resolve("../lib/env");
  delete require.cache[modPath];
  return require("../lib/env") as typeof import("../lib/env");
}

test("env defaults are stable and normalized", () => {
  const env = loadEnvModule({});

  assert.equal(env.appEnv.appName, "Clawmesh Quickstart Frontend");
  assert.equal(env.appEnv.appUrl, "http://localhost:3000");
  assert.equal(env.appEnv.apiBaseUrl, "http://localhost:8080/api/v1");
  assert.equal(env.appEnv.stripePublishableKey, "");
  assert.equal(env.appEnv.defaultPlan, "pro");
  assert.equal(env.appEnv.defaultInterval, "monthly");
  assert.equal(env.appEnv.defaultCreditsQuantity, 1);
  assert.equal(env.appEnv.defaultTopUpAmountUSD, 25);
  assert.equal(env.appEnv.creditsPriceId, "");
  assert.equal(env.appEnv.stripeSuccessPath, "/billing?checkout=success");
  assert.equal(env.appEnv.stripeCancelPath, "/billing?checkout=cancelled");
  assert.equal(env.apiUrl("auth/send-code"), "http://localhost:8080/api/v1/auth/send-code");
  assert.equal(env.appUrl("login"), "http://localhost:3000/login");
});

test("env trims origins, normalizes paths, and parses positive ints", () => {
  const env = loadEnvModule({
    NEXT_PUBLIC_APP_NAME: "  Demo App  ",
    NEXT_PUBLIC_APP_URL: "https://app.example.com///",
    NEXT_PUBLIC_API_BASE_URL: "https://api.example.com/api/v1/",
    NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY: " pk_test_123 ",
    NEXT_PUBLIC_DEFAULT_PLAN: " lifetime ",
    NEXT_PUBLIC_DEFAULT_INTERVAL: " yearly ",
    NEXT_PUBLIC_DEFAULT_CREDITS_QUANTITY: "3",
    NEXT_PUBLIC_DEFAULT_TOPUP_AMOUNT_USD: "49.5",
    NEXT_PUBLIC_CREDITS_PRICE_ID: " price_123 ",
    NEXT_PUBLIC_STRIPE_SUCCESS_PATH: "billing/success",
    NEXT_PUBLIC_STRIPE_CANCEL_PATH: "/billing/cancel"
  });

  assert.equal(env.appEnv.appName, "Demo App");
  assert.equal(env.appEnv.appUrl, "https://app.example.com");
  assert.equal(env.appEnv.apiBaseUrl, "https://api.example.com/api/v1");
  assert.equal(env.appEnv.stripePublishableKey, "pk_test_123");
  assert.equal(env.appEnv.defaultPlan, "lifetime");
  assert.equal(env.appEnv.defaultInterval, "yearly");
  assert.equal(env.appEnv.defaultCreditsQuantity, 3);
  assert.equal(env.appEnv.defaultTopUpAmountUSD, 49.5);
  assert.equal(env.appEnv.creditsPriceId, "price_123");
  assert.equal(env.appEnv.stripeSuccessPath, "/billing/success");
  assert.equal(env.appEnv.stripeCancelPath, "/billing/cancel");
  assert.equal(env.apiUrl("/stripe/subscription"), "https://api.example.com/api/v1/stripe/subscription");
  assert.equal(env.appUrl("/billing"), "https://app.example.com/billing");
});

test("invalid credits quantity falls back to default", () => {
  const env = loadEnvModule({
    NEXT_PUBLIC_DEFAULT_CREDITS_QUANTITY: "0"
  });

  assert.equal(env.appEnv.defaultCreditsQuantity, 1);
});

test("invalid top-up amount falls back to default", () => {
  const env = loadEnvModule({
    NEXT_PUBLIC_DEFAULT_TOPUP_AMOUNT_USD: "0"
  });

  assert.equal(env.appEnv.defaultTopUpAmountUSD, 25);
});
