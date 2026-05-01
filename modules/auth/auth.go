// Package auth is a portable, provider-agnostic authentication module.
//
// Layering (mirrors modules/billing and modules/email):
//
//	domain/   pure types: Identity, Token, OAuthProfile, errors
//	event/    domain events (UserSignedUp, UserLoggedIn)
//	port/     interfaces: IdentityProvider, EmailCodeIssuer/Verifier,
//	          TokenSigner, WSTicketSigner, UserStore, RoleResolver,
//	          ExchangeCodeStore, EventBus
//	adapter/  concrete implementations
//	  jwt/         HS256 token + WS ticket signer
//	  google/      Google OAuth IdentityProvider
//	  emailcode/   passwordless email-code provider (uses modules/email)
//	  redisstore/  CodeRateLimitStore on Redis
//	  memstore/    in-memory CodeRateLimitStore + ExchangeCodeStore (dev/tests)
//	  gormstore/   GORM-backed ExchangeCodeStore
//	  eventbus/    in-process synchronous bus
//	app/      use cases: SendCode, VerifyCode, StartOAuth, OAuthCallback,
//	          ExchangeToken, RefreshToken, IssueWSTicket
//	http/     Gin handlers, UserAuthMiddleware, Mount()
//	auth.go   Module wiring
//
// The host project provides:
//  1. port.UserStore         — read/write the host's user table
//  2. port.RoleResolver      — assign roles to identities
//  3. port.ExchangeCodeStore — persist OAuth callback codes
//  4. event listeners        — react to UserSignedUp / UserLoggedIn
//
// Auth never imports project-specific models.
package auth

// The Module struct, Deps and New() constructor live in module.go to
// keep this file as the package's overview.
