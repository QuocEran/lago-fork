package subscriptions

import (
	"errors"
	"time"
)

var (
	ErrInvalidTransition = errors.New("invalid_subscription_status_transition")
	ErrAlreadyTerminated  = errors.New("subscription_already_terminated")
	ErrAlreadyCanceled   = errors.New("subscription_already_canceled")
	ErrAlreadyActive      = errors.New("subscription_already_active")
)

// CanActivate reports whether the subscription may transition to Active.
func CanActivate(state *SubscriptionState) bool {
	return state.Status == SubscriptionStatusPending
}

// CanTerminate reports whether the subscription may transition to Terminated or Canceled.
// Active subscriptions → terminated; pending subscriptions → canceled.
func CanTerminate(state *SubscriptionState) bool {
	return state.Status == SubscriptionStatusActive ||
		state.Status == SubscriptionStatusPending
}

// ApplyActivate transitions the state to Active and stamps StartedAt.
func ApplyActivate(state *SubscriptionState) error {
	switch state.Status {
	case SubscriptionStatusActive:
		return ErrAlreadyActive
	case SubscriptionStatusTerminated:
		return ErrAlreadyTerminated
	case SubscriptionStatusCanceled:
		return ErrAlreadyCanceled
	}

	if !CanActivate(state) {
		return ErrInvalidTransition
	}

	now := time.Now()
	state.Status = SubscriptionStatusActive
	state.StartedAt = &now
	return nil
}

// ApplyTerminate transitions the state to Terminated (active) or Canceled (pending).
// Stamps TerminatedAt or CanceledAt accordingly.
func ApplyTerminate(state *SubscriptionState) error {
	switch state.Status {
	case SubscriptionStatusTerminated:
		return ErrAlreadyTerminated
	case SubscriptionStatusCanceled:
		return ErrAlreadyCanceled
	}

	if !CanTerminate(state) {
		return ErrInvalidTransition
	}

	now := time.Now()
	if state.Status == SubscriptionStatusPending {
		state.Status = SubscriptionStatusCanceled
		state.CanceledAt = &now
	} else {
		state.Status = SubscriptionStatusTerminated
		state.TerminatedAt = &now
	}
	return nil
}
