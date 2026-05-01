package port

// CodeGenerator produces a referral code for a given user.
//
// Two reference implementations are provided:
//
//	deterministic — derives the code from user_id (no DB collision risk
//	                because user_id is unique). Stable across restarts.
//	                Convenient when you don't want to persist codes.
//	random        — generates a random alphanumeric code; the caller
//	                retries on storage collision. Looks more "real" but
//	                requires a code repo to enforce uniqueness.
type CodeGenerator interface {
	Generate(userID string) string
}
