package service

import (
	bizmodel "github.com/gkk/stall-location/backend/internal/model"
)

const (
	IdentityCustomer    = bizmodel.IdentityCustomer
	IdentityApplication = bizmodel.IdentityApplication
	IdentityMerchant    = bizmodel.IdentityMerchant

	StatusActive   = bizmodel.StatusActive
	StatusDisabled = bizmodel.StatusDisabled

	VerifyUnverified = bizmodel.VerifyUnverified
	VerifyPending    = bizmodel.VerifyPending
	VerifyVerified   = bizmodel.VerifyVerified
	VerifyRejected   = bizmodel.VerifyRejected

	ProductStatusOnSale  = bizmodel.ProductStatusOnSale
	ProductStatusOffSale = bizmodel.ProductStatusOffSale

	StatusEnded = bizmodel.StatusEnded

	OrderPendingAccept = bizmodel.OrderPendingAccept
	OrderAccepted      = bizmodel.OrderAccepted
	OrderPreparing     = bizmodel.OrderPreparing
	OrderReady         = bizmodel.OrderReady
	OrderCompleted     = bizmodel.OrderCompleted
	OrderRejected      = bizmodel.OrderRejected
	OrderCanceled      = bizmodel.OrderCanceled
	OrderExpired       = bizmodel.OrderExpired

	PaymentUnpaid    = bizmodel.PaymentUnpaid
	PaymentPaying    = bizmodel.PaymentPaying
	PaymentPaid      = bizmodel.PaymentPaid
	PaymentRefunding = bizmodel.PaymentRefunding
	PaymentRefunded  = bizmodel.PaymentRefunded
	PaymentFailed    = bizmodel.PaymentFailed

	ApplicationPending   = bizmodel.ApplicationPending
	ApplicationReviewing = bizmodel.ApplicationReviewing
	ApplicationApproved  = bizmodel.ApplicationApproved
	ApplicationRejected  = bizmodel.ApplicationRejected

	FeedbackSourceCustomer = bizmodel.FeedbackSourceCustomer
	FeedbackSourceMerchant = bizmodel.FeedbackSourceMerchant
	FeedbackStatusPending  = bizmodel.FeedbackStatusPending
	FeedbackStatusHandling = bizmodel.FeedbackStatusHandling
	FeedbackStatusResolved = bizmodel.FeedbackStatusResolved
	FeedbackStatusClosed   = bizmodel.FeedbackStatusClosed
)

type Merchant = bizmodel.Merchant
type PublicMerchant = bizmodel.PublicMerchant
type ProductSummary = bizmodel.ProductSummary
type Customer = bizmodel.User
type Product = bizmodel.Product
type ProductItem = bizmodel.ProductItem
type PublicProduct = bizmodel.PublicProduct
type StallSession = bizmodel.StallSession
type PublicStallSession = bizmodel.PublicStallSession
type Preorder = bizmodel.Preorder
type PreorderItem = bizmodel.PreorderItem
type Favorite = bizmodel.Favorite
type Application = bizmodel.Application
type ApplicationItem = bizmodel.ApplicationItem
type Feedback = bizmodel.Feedback
type FeedbackItem = bizmodel.FeedbackItem
type FeedbackCreate = bizmodel.FeedbackCreate
