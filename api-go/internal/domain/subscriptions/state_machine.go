package subscriptions

import (
	"errors"
	"time"

	"github.com/getlago/lago/api-go/internal/models"
)

var (
	ErrInvalidTransition   = errors.New("invalid_subscription_status_transition")
	ErrAlreadyTerminated   = errors.New("subscription_already_terminated")
	ErrAlreadyCanceled     = errors.New("subscription_already_canceled")
	ErrAlreadyActive       = errors.New("subscription_already_active")
)

// CanActivate reports whether the subscription may transition to Active.
func CanActivate(sub *models.Subscription) bool {
	return sub.Status == models.SubscriptionStatusPending
}

// CanTerminate reports whether the subscription may transition to Terminated or Canceled.
// Active subscriptions → terminated; pending subscriptions → canceled.
func CanTerminate(sub *models.Subscription) bool {
	return sub.Status == models.SubscriptionStatusActive ||
		sub.Status == models.SubscriptionStatusPending
}

// ApplyActivate transitions the subscription to Active and stamps StartedAt.
func ApplyActivate(sub *models.Subscription) error {
	switch sub.Status {
	case models.SubscriptionStatusActive:
		return ErrAlreadyActive
	case models.SubscriptionStatusTerminated:
		return ErrAlreadyTerminated
	case models.SubscriptionStatusCanceled:
		return ErrAlreadyCanceled
	}

	if !CanActivate(sub) {
		return ErrInvalidTransition
	}

	now := time.Now()
	sub.Status = models.SubscriptionStatusActive
	sub.StartedAt = &now
	return nil
}

// ApplyTerminate transitions the subscription to Terminated (active) or Canceled (pending).
// Stamps TerminatedAt or CanceledAt accordingly.
func ApplyTerminate(sub *models.Subscription) error {
	switch sub.Status {
	case models.SubscriptionStatusTerminated:
		return ErrAlreadyTerminated
	case models.SubscriptionStatusCanceled:
		return ErrAlreadyCanceled
	}

	if !CanTerminate(sub) {
		return ErrInvalidTransition
	}

	now := time.Now()
	if sub.Status == models.SubscriptionStatusPending {
		sub.Status = models.SubscriptionStatusCanceled
		sub.CanceledAt = &now
	} else {
		sub.Status = models.SubscriptionStatusTerminated
		sub.TerminatedAt = &now
	}
	return nil
}
