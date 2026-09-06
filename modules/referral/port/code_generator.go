package port

// CodeGenerator produces a referral code for a given user.
//
// Two reference implementations are provided:
//
//	deterministic — derives the code from user_id and is stable across
//	                restarts. Truncation can collide, so callers still need
//	                a repository uniqueness constraint.
//	random        — generates a random alphanumeric code; the caller
//	                retries on storage collision. Recommended for public
//	                codes and requires a repository uniqueness constraint.
type CodeGenerator interface {
	Generate(userID string) string
}
