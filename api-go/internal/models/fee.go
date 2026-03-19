package models

import "time"

// FeeType maps to Rails integer enum for fee_type.
type FeeType int

const (
	FeeTypeCharge      FeeType = 0
	FeeTypeAddOn       FeeType = 1
	FeeTypeSubscription FeeType = 2
	FeeTypeCredit      FeeType = 3
	FeeTypeCommitment  FeeType = 4
	FeeTypeFixedCharge FeeType = 5
)

// FeePaymentStatus maps to Rails integer enum for payment_status.
type FeePaymentStatus int

const (
	FeePaymentStatusPending   FeePaymentStatus = 0
	FeePaymentStatusSucceeded FeePaymentStatus = 1
	FeePaymentStatusFailed    FeePaymentStatus = 2
	FeePaymentStatusRefunded  FeePaymentStatus = 3
)

// Fee maps to the existing Rails `fees` table.
type Fee struct {
	BaseModel
	OrganizationID                  string           `gorm:"column:organization_id;not null;index"`
	BillingEntityID                 string           `gorm:"column:billing_entity_id;not null"`
	InvoiceID                       *string          `gorm:"column:invoice_id;index"`
	ChargeID                        *string          `gorm:"column:charge_id"`
	SubscriptionID                  *string          `gorm:"column:subscription_id;index"`
	AppliedAddOnID                  *string          `gorm:"column:applied_add_on_id"`
	AddOnID                         *string          `gorm:"column:add_on_id"`
	ChargeFilterID                  *string          `gorm:"column:charge_filter_id"`
	InvoiceableType                 *string          `gorm:"column:invoiceable_type"`
	InvoiceableID                   *string          `gorm:"column:invoiceable_id"`
	TrueUpParentFeeID               *string          `gorm:"column:true_up_parent_fee_id"`
	FixedChargeID                   *string          `gorm:"column:fixed_charge_id"`
	FeeType                         *FeeType         `gorm:"column:fee_type"`
	PaymentStatus                   FeePaymentStatus `gorm:"column:payment_status;not null;default:0"`
	AmountCents                     int64            `gorm:"column:amount_cents;not null;default:0"`
	AmountCurrency                  string           `gorm:"column:amount_currency;not null;default:''"`
	TaxesAmountCents                int64            `gorm:"column:taxes_amount_cents;not null;default:0"`
	TaxesRate                       float64          `gorm:"column:taxes_rate;not null;default:0"`
	UnitAmountCents                 int64            `gorm:"column:unit_amount_cents;not null;default:0"`
	Units                           float64          `gorm:"column:units;not null;default:0"`
	TotalAggregatedUnits            *float64         `gorm:"column:total_aggregated_units"`
	EventsCount                     *int             `gorm:"column:events_count"`
	PayInAdvance                    bool             `gorm:"column:pay_in_advance;not null;default:false"`
	PayInAdvanceEventID             *string          `gorm:"column:pay_in_advance_event_id"`
	PayInAdvanceEventTransactionID  *string          `gorm:"column:pay_in_advance_event_transaction_id"`
	Description                     *string          `gorm:"column:description"`
	InvoiceDisplayName              *string          `gorm:"column:invoice_display_name"`
	DeletedAt                       *time.Time       `gorm:"column:deleted_at;index"`
	SucceededAt                     *time.Time       `gorm:"column:succeeded_at"`
	FailedAt                        *time.Time       `gorm:"column:failed_at"`
	RefundedAt                      *time.Time       `gorm:"column:refunded_at"`
}

func (Fee) TableName() string { return "fees" }
