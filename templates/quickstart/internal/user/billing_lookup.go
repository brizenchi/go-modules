package user

import (
	"context"

	billingport "github.com/brizenchi/go-modules/modules/billing/port"
)

// BillingLookup exposes only ID and email to billing. Adding fields to User
// never changes the shared billing module.
type BillingLookup struct {
	users *Repository
}

func NewBillingLookup(users *Repository) *BillingLookup { return &BillingLookup{users: users} }

func (l *BillingLookup) FindBillingAccount(ctx context.Context, userID string) (billingport.Account, error) {
	user, err := l.users.FindByID(ctx, userID)
	if err != nil {
		return billingport.Account{}, err
	}
	return billingport.Account{UserID: user.ID, Email: user.Email}, nil
}

func (l *BillingLookup) FindUserIDByEmail(ctx context.Context, email string) (string, error) {
	user, err := l.users.FindByEmail(ctx, email)
	if err != nil {
		return "", err
	}
	return user.ID, nil
}

var _ billingport.AccountLookup = (*BillingLookup)(nil)
