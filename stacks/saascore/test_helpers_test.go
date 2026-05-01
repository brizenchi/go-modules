package saascore

import (
	"time"

	billingdomain "github.com/brizenchi/go-modules/modules/billing/domain"
)

func subscriptionSnapshot(customerID, subID, priceID, plan, status string, end time.Time) billingdomain.SubscriptionSnapshot {
	return billingdomain.SubscriptionSnapshot{
		ProviderCustomerID:     customerID,
		ProviderSubscriptionID: subID,
		ProviderPriceID:        priceID,
		Plan:                   billingdomain.PlanType(plan),
		Status:                 billingdomain.SubscriptionStatus(status),
		PeriodEnd:              &end,
	}
}
