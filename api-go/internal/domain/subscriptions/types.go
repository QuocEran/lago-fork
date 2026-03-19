package subscriptions

import "time"

// SubscriptionStatus represents the lifecycle status of a subscription in the domain.
// Values match models.SubscriptionStatus for easy mapping.
type SubscriptionStatus int

const (
	SubscriptionStatusPending    SubscriptionStatus = 0
	SubscriptionStatusActive     SubscriptionStatus = 1
	SubscriptionStatusTerminated SubscriptionStatus = 2
	SubscriptionStatusCanceled   SubscriptionStatus = 3
)

// SubscriptionState holds the fields required for subscription state machine logic.
type SubscriptionState struct {
	Status       SubscriptionStatus
	StartedAt   *time.Time
	CanceledAt  *time.Time
	TerminatedAt *time.Time
}
