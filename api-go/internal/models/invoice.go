package models

import "time"

// InvoiceStatus maps to the Rails integer enum for invoice status.
type InvoiceStatus int

const (
	InvoiceStatusDraft      InvoiceStatus = 0
	InvoiceStatusFinalized  InvoiceStatus = 1
	InvoiceStatusVoided     InvoiceStatus = 2
	InvoiceStatusGenerating InvoiceStatus = 3
	InvoiceStatusFailed     InvoiceStatus = 4
)

// InvoicePaymentStatus maps to the Rails integer enum for invoice payment_status.
type InvoicePaymentStatus int

const (
	InvoicePaymentStatusPending   InvoicePaymentStatus = 0
	InvoicePaymentStatusSucceeded InvoicePaymentStatus = 1
	InvoicePaymentStatusFailed    InvoicePaymentStatus = 2
)

// InvoiceType maps to the Rails integer enum for invoice_type.
type InvoiceType int

const (
	InvoiceTypeSubscription       InvoiceType = 0
	InvoiceTypeAddOn              InvoiceType = 1
	InvoiceTypeCredit             InvoiceType = 2
	InvoiceTypeOneOff             InvoiceType = 3
	InvoiceTypeAdvanceCharges     InvoiceType = 4
	InvoiceTypeProgressiveBilling InvoiceType = 5
)

// Invoice maps to the existing Rails `invoices` table.
// No migration is required — the table already exists.
type Invoice struct {
	BaseModel
	OrganizationID     string               `gorm:"column:organization_id;not null;index"`
	BillingEntityID    string               `gorm:"column:billing_entity_id;not null"`
	CustomerID         *string              `gorm:"column:customer_id"`
	Status             InvoiceStatus        `gorm:"column:status;not null;default:1"`
	PaymentStatus      InvoicePaymentStatus `gorm:"column:payment_status;not null;default:0"`
	InvoiceType        InvoiceType          `gorm:"column:invoice_type;not null;default:0"`
	Number             string               `gorm:"column:number;not null;default:''"`
	SequentialID       *int                 `gorm:"column:sequential_id"`
	OrgSequentialID    int                  `gorm:"column:organization_sequential_id;not null;default:0"`
	Currency           string               `gorm:"column:currency"`
	TaxesRate          float64              `gorm:"column:taxes_rate;not null;default:0"`
	FeesAmountCents    int64                `gorm:"column:fees_amount_cents;not null;default:0"`
	TaxesAmountCents   int64                `gorm:"column:taxes_amount_cents;not null;default:0"`
	CouponsAmountCents int64                `gorm:"column:coupons_amount_cents;not null;default:0"`
	CreditNotesAmountCents      int64   `gorm:"column:credit_notes_amount_cents;not null;default:0"`
	PrepaidCreditAmountCents    int64   `gorm:"column:prepaid_credit_amount_cents;not null;default:0"`
	SubTotalExcludingTaxesCents int64   `gorm:"column:sub_total_excluding_taxes_amount_cents;not null;default:0"`
	SubTotalIncludingTaxesCents int64   `gorm:"column:sub_total_including_taxes_amount_cents;not null;default:0"`
	TotalAmountCents            int64   `gorm:"column:total_amount_cents;not null;default:0"`
	NegativeAmountCents         int64   `gorm:"column:negative_amount_cents;not null;default:0"`
	NetPaymentTerm              int     `gorm:"column:net_payment_term;not null;default:0"`
	VersionNumber               int     `gorm:"column:version_number;not null;default:4"`
	Timezone                    string  `gorm:"column:timezone;not null;default:'UTC'"`
	IssuingDate                 *time.Time `gorm:"column:issuing_date"`
	PaymentDueDate              *time.Time `gorm:"column:payment_due_date"`
	FinalizedAt                 *time.Time `gorm:"column:finalized_at"`
	VoidedAt                    *time.Time `gorm:"column:voided_at"`
	PaymentOverdue              bool    `gorm:"column:payment_overdue;not null;default:false"`
	ReadyForPaymentProcessing   bool    `gorm:"column:ready_for_payment_processing;not null;default:true"`
	ReadyToBeRefreshed          bool    `gorm:"column:ready_to_be_refreshed;not null;default:false"`
	SkipCharges                 bool    `gorm:"column:skip_charges;not null;default:false"`
	SelfBilled                  bool    `gorm:"column:self_billed;not null;default:false"`
	PaymentAttempts             int     `gorm:"column:payment_attempts;not null;default:0"`
	PaymentDisputeLostAt        *time.Time `gorm:"column:payment_dispute_lost_at"`
}

func (Invoice) TableName() string { return "invoices" }
