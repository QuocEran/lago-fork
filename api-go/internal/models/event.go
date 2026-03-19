package models

import "time"

// Event stores raw usage events sent by API clients.
// Uses SoftDeleteModel for deleted_at so GORM scopes exclude soft-deleted rows by default.
type Event struct {
	SoftDeleteModel
	OrganizationID          string     `gorm:"column:organization_id;not null;index"`
	CustomerID              *string    `gorm:"column:customer_id"`
	TransactionID           string     `gorm:"column:transaction_id;not null;index"`
	Code                    string     `gorm:"column:code;not null"`
	Properties              JSONBMap   `gorm:"column:properties;type:jsonb"`
	Timestamp               *time.Time `gorm:"column:timestamp"`
	Metadata                JSONBMap   `gorm:"column:metadata;type:jsonb"`
	SubscriptionID          *string    `gorm:"column:subscription_id"`
	ExternalCustomerID      *string    `gorm:"column:external_customer_id"`
	ExternalSubscriptionID  *string    `gorm:"column:external_subscription_id"`
	PreciseTotalAmountCents *string    `gorm:"column:precise_total_amount_cents;type:numeric(40,15)"`
}

func (Event) TableName() string { return "events" }
