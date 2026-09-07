package event

const (
	KindRefundUpdated  Kind = "payment.refund_updated"
	KindChargeRefunded Kind = "payment.charge_refunded"
	KindDisputeUpdated Kind = "payment.dispute_updated"
)

type RefundUpdated struct {
	ProviderEventType       string
	ProviderRefundID        string
	ProviderChargeID        string
	ProviderPaymentIntentID string
	Status                  string
	Reason                  string
	FailureReason           string
	Amount                  int64
	Currency                string
}

type ChargeRefunded struct {
	ProviderChargeID        string
	ProviderPaymentIntentID string
	ProviderCustomerID      string
	Amount                  int64
	AmountRefunded          int64
	FullyRefunded           bool
	Currency                string
}

type DisputeUpdated struct {
	ProviderEventType       string
	ProviderDisputeID       string
	ProviderChargeID        string
	ProviderPaymentIntentID string
	Status                  string
	Reason                  string
	Amount                  int64
	Currency                string
}
