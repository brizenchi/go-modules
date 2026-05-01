package saascore

import (
	"context"
	"strings"
)

type contextKey string

const referralCodeKey contextKey = "saascore.referral_code"

func withReferralCode(ctx context.Context, code string) context.Context {
	code = strings.TrimSpace(code)
	if code == "" {
		return ctx
	}
	return context.WithValue(ctx, referralCodeKey, code)
}

func referralCode(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	value, _ := ctx.Value(referralCodeKey).(string)
	return strings.TrimSpace(value)
}
