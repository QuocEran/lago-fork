package subscriptions_test

import (
	"testing"

	domain "github.com/getlago/lago/api-go/internal/domain/subscriptions"
)

func makeState(status domain.SubscriptionStatus) *domain.SubscriptionState {
	return &domain.SubscriptionState{Status: status}
}

func TestApplyActivate_FromPending(t *testing.T) {
	state := makeState(domain.SubscriptionStatusPending)
	if err := domain.ApplyActivate(state); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if state.Status != domain.SubscriptionStatusActive {
		t.Errorf("expected Active, got %v", state.Status)
	}
	if state.StartedAt == nil {
		t.Error("expected StartedAt to be set")
	}
}

func TestApplyActivate_AlreadyActive(t *testing.T) {
	state := makeState(domain.SubscriptionStatusActive)
	if err := domain.ApplyActivate(state); err != domain.ErrAlreadyActive {
		t.Errorf("expected ErrAlreadyActive, got %v", err)
	}
}

func TestApplyActivate_Terminated(t *testing.T) {
	state := makeState(domain.SubscriptionStatusTerminated)
	if err := domain.ApplyActivate(state); err != domain.ErrAlreadyTerminated {
		t.Errorf("expected ErrAlreadyTerminated, got %v", err)
	}
}

func TestApplyTerminate_ActiveToTerminated(t *testing.T) {
	state := makeState(domain.SubscriptionStatusActive)
	if err := domain.ApplyTerminate(state); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if state.Status != domain.SubscriptionStatusTerminated {
		t.Errorf("expected Terminated, got %v", state.Status)
	}
	if state.TerminatedAt == nil {
		t.Error("expected TerminatedAt to be set")
	}
}

func TestApplyTerminate_PendingToCanceled(t *testing.T) {
	state := makeState(domain.SubscriptionStatusPending)
	if err := domain.ApplyTerminate(state); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if state.Status != domain.SubscriptionStatusCanceled {
		t.Errorf("expected Canceled, got %v", state.Status)
	}
	if state.CanceledAt == nil {
		t.Error("expected CanceledAt to be set")
	}
}

func TestApplyTerminate_AlreadyTerminated(t *testing.T) {
	state := makeState(domain.SubscriptionStatusTerminated)
	if err := domain.ApplyTerminate(state); err != domain.ErrAlreadyTerminated {
		t.Errorf("expected ErrAlreadyTerminated, got %v", err)
	}
}

func TestApplyTerminate_AlreadyCanceled(t *testing.T) {
	state := makeState(domain.SubscriptionStatusCanceled)
	if err := domain.ApplyTerminate(state); err != domain.ErrAlreadyCanceled {
		t.Errorf("expected ErrAlreadyCanceled, got %v", err)
	}
}
