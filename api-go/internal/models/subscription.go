package models

import "time"

// SubscriptionStatus mirrors the Rails integer enum for subscriptions.status.
type SubscriptionStatus int

const (
	SubscriptionStatusPending    SubscriptionStatus = 0
	SubscriptionStatusActive     SubscriptionStatus = 1
	SubscriptionStatusTerminated SubscriptionStatus = 2
	SubscriptionStatusCanceled   SubscriptionStatus = 3
)

// SubscriptionStatusFromString converts a string to SubscriptionStatus.
func SubscriptionStatusFromString(s string) (SubscriptionStatus, bool) {
	m := map[string]SubscriptionStatus{
		"pending":    SubscriptionStatusPending,
		"active":     SubscriptionStatusActive,
		"terminated": SubscriptionStatusTerminated,
		"canceled":   SubscriptionStatusCanceled,
	}
	v, ok := m[s]
	return v, ok
}

// SubscriptionStatusToString returns the string representation of a SubscriptionStatus.
func SubscriptionStatusToString(s SubscriptionStatus) string {
	m := map[SubscriptionStatus]string{
		SubscriptionStatusPending:    "pending",
		SubscriptionStatusActive:     "active",
		SubscriptionStatusTerminated: "terminated",
		SubscriptionStatusCanceled:   "canceled",
	}
	if str, ok := m[s]; ok {
		return str
	}
	return "pending"
}

// BillingTime mirrors the Rails integer enum for subscriptions.billing_time.
type BillingTime int

const (
	BillingTimeCalendar    BillingTime = 0
	BillingTimeAnniversary BillingTime = 1
)

// BillingTimeFromString converts a string to BillingTime.
func BillingTimeFromString(s string) (BillingTime, bool) {
	m := map[string]BillingTime{
		"calendar":    BillingTimeCalendar,
		"anniversary": BillingTimeAnniversary,
	}
	v, ok := m[s]
	return v, ok
}

// BillingTimeToString returns the string representation of a BillingTime.
func BillingTimeToString(b BillingTime) string {
	if b == BillingTimeAnniversary {
		return "anniversary"
	}
	return "calendar"
}

// Subscription maps to the subscriptions table.
type Subscription struct {
	BaseModel
	OrganizationID         string             `gorm:"column:organization_id;not null;index"`
	CustomerID             string             `gorm:"column:customer_id;not null;index"`
	PlanID                 string             `gorm:"column:plan_id;not null;index"`
	PreviousSubscriptionID *string            `gorm:"column:previous_subscription_id;index"`
	ExternalID             string             `gorm:"column:external_id;not null;index"`
	Name                   *string            `gorm:"column:name"`
	Status                 SubscriptionStatus `gorm:"column:status;not null;default:0"`
	BillingTime            BillingTime        `gorm:"column:billing_time;not null;default:0"`
	SubscriptionAt         *time.Time         `gorm:"column:subscription_at"`
	StartedAt              *time.Time         `gorm:"column:started_at"`
	EndingAt               *time.Time         `gorm:"column:ending_at"`
	CanceledAt             *time.Time         `gorm:"column:canceled_at"`
	TerminatedAt           *time.Time         `gorm:"column:terminated_at"`
	TrialEndedAt           *time.Time         `gorm:"column:trial_ended_at"`
	// Associations loaded on demand.
	Customer *Customer `gorm:"foreignKey:CustomerID"`
	Plan     *Plan     `gorm:"foreignKey:PlanID"`
}

func (Subscription) TableName() string { return "subscriptions" }
