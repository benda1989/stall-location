package model

const (
	IdentityCustomer    = "customer"
	IdentityApplication = "application"
	IdentityMerchant    = "merchant"

	StatusActive   = "active"
	StatusDisabled = "disabled"
	StatusEnded    = "ended"

	VerifyUnverified = "unverified"
	VerifyPending    = "pending"
	VerifyVerified   = "verified"
	VerifyRejected   = "rejected"

	ProductStatusOnSale  = "on_sale"
	ProductStatusOffSale = "off_sale"

	OrderPendingAccept = "pending_accept"
	OrderAccepted      = "accepted"
	OrderPreparing     = "preparing"
	OrderReady         = "ready"
	OrderCompleted     = "completed"
	OrderRejected      = "rejected"
	OrderCanceled      = "canceled"
	OrderExpired       = "expired"

	PaymentUnpaid    = "unpaid"
	PaymentPaying    = "paying"
	PaymentPaid      = "paid"
	PaymentRefunding = "refunding"
	PaymentRefunded  = "refunded"
	PaymentFailed    = "failed"

	ApplicationPending   = "pending"
	ApplicationReviewing = "reviewing"
	ApplicationApproved  = "approved"
	ApplicationRejected  = "rejected"

	FeedbackSourceCustomer = "customer"
	FeedbackSourceMerchant = "merchant"
	FeedbackStatusPending  = "pending"
	FeedbackStatusHandling = "handling"
	FeedbackStatusResolved = "resolved"
	FeedbackStatusClosed   = "closed"
)
