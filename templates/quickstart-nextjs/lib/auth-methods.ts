import type { CapabilitiesView, OAuthProvider } from "./api";

export type AuthMethodPolicy = {
  emailConfigured: boolean;
  emailEnabled: boolean;
  oauthProvidersConfigured: boolean;
  oauthProviders: OAuthProvider[];
};

export type AvailableAuthMethods = {
  emailEnabled: boolean;
  oauthProviders: OAuthProvider[];
  legacyFallback: boolean;
};

const knownOAuthProviders = new Set<OAuthProvider>(["google", "github"]);

/**
 * Resolve login methods with the API as the source of truth. Build-time env
 * flags may narrow that set, but they can never advertise a provider the API
 * did not report. Older APIs without `auth` retain the explicit env behavior.
 */
export function resolveAuthMethods(
  capabilities: CapabilitiesView,
  policy: AuthMethodPolicy
): AvailableAuthMethods {
  if (!capabilities.auth) {
    return {
      emailEnabled: policy.emailEnabled,
      oauthProviders: policy.oauthProviders,
      legacyFallback: true
    };
  }

  const allowlistedProviders = policy.oauthProvidersConfigured
    ? new Set(policy.oauthProviders)
    : null;
  const oauthProviders = capabilities.auth.oauth_providers.filter(
    (provider): provider is OAuthProvider =>
      knownOAuthProviders.has(provider)
      && (allowlistedProviders === null || allowlistedProviders.has(provider))
  );

  return {
    emailEnabled: capabilities.auth.email_enabled
      && (!policy.emailConfigured || policy.emailEnabled),
    oauthProviders,
    legacyFallback: false
  };
}
