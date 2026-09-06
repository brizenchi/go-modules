import assert from "node:assert/strict";
import { test } from "node:test";
import { resolveAuthMethods, type AuthMethodPolicy } from "../lib/auth-methods";
import type { CapabilitiesView } from "../lib/api";

const baseCapabilities: CapabilitiesView = {
  auth: { email_enabled: true, oauth_providers: ["google", "github"] },
  account: { enabled: true },
  billing: {
    enabled: false,
    provider: "stripe",
    offers: { subscriptions: [], lifetime: false, credits: false }
  },
  referral: { enabled: true }
};

const unrestrictedPolicy: AuthMethodPolicy = {
  emailConfigured: false,
  emailEnabled: true,
  oauthProvidersConfigured: false,
  oauthProviders: []
};

test("backend auth capabilities are authoritative when env is unset", () => {
  assert.deepEqual(resolveAuthMethods(baseCapabilities, unrestrictedPolicy), {
    emailEnabled: true,
    oauthProviders: ["google", "github"],
    legacyFallback: false
  });
});

test("frontend env can narrow but cannot add backend login methods", () => {
  assert.deepEqual(resolveAuthMethods(
    {
      ...baseCapabilities,
      auth: { email_enabled: false, oauth_providers: ["google"] }
    },
    {
      emailConfigured: true,
      emailEnabled: true,
      oauthProvidersConfigured: true,
      oauthProviders: ["github", "google"]
    }
  ), {
    emailEnabled: false,
    oauthProviders: ["google"],
    legacyFallback: false
  });

  assert.deepEqual(resolveAuthMethods(baseCapabilities, {
    emailConfigured: true,
    emailEnabled: false,
    oauthProvidersConfigured: true,
    oauthProviders: ["github"]
  }), {
    emailEnabled: false,
    oauthProviders: ["github"],
    legacyFallback: false
  });
});

test("older capability payloads use explicit env compatibility without a fake OAuth default", () => {
  const { auth: _auth, ...legacyCapabilities } = baseCapabilities;
  assert.deepEqual(resolveAuthMethods(legacyCapabilities, unrestrictedPolicy), {
    emailEnabled: true,
    oauthProviders: [],
    legacyFallback: true
  });
});
