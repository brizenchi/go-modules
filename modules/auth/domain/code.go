package domain

import "time"

// VerificationCode is a one-time email verification code (passwordless flow).
type VerificationCode struct {
	Email        string
	Code         string
	ExpiresAt    time.Time
	AttemptCount int
}

// ExchangeCode is a one-time short-lived code exchanged for a Token.
//
// Used in OAuth callbacks: the provider's callback URL receives an
// auth_code, the auth module persists it, redirects the browser to the
// frontend, and the frontend POSTs the code to /auth/exchange-token to
// receive the JWT. This avoids putting the JWT in a URL fragment.
type ExchangeCode struct {
	Code        string
	UserID      string
	Provider    Provider
	IsNew       bool
	BindingHash string
	ExpiresAt   time.Time
}

// OAuthFlow is the server-side half of OAuth's browser binding. Only hashes
// are persisted: StateHash identifies the signed provider state and
// BindingHash identifies the random HttpOnly cookie delivered to the browser
// that started the flow.
type OAuthFlow struct {
	StateHash   string
	Provider    Provider
	BindingHash string
	// VerifierChallenge is SHA-256 of a browser-held verifier, encoded with
	// unpadded base64url. It binds the final exchange-code step to that browser.
	VerifierChallenge string
	ExpiresAt         time.Time
}

// CodeIssueRequest captures the inputs for SendCode use case.
type CodeIssueRequest struct {
	Email string
}

// CodeIssueResult reports rate-limit status to the caller.
type CodeIssueResult struct {
	Email     string
	ExpiresAt time.Time
	DebugCode string // populated only when running in debug mode
}
