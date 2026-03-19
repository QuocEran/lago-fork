package subscriptions_test

import (
	"testing"
	"time"

	domain "github.com/getlago/lago/api-go/internal/domain/subscriptions"
	"github.com/getlago/lago/api-go/internal/models"
)

func makeSub(status models.SubscriptionStatus) *models.Subscription {
	return &models.Subscription{
		BaseModel:      models.BaseModel{ID: "sub-1", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		OrganizationID: "org-1",
		CustomerID:     "cust-1",
		PlanID:         "plan-1",
		ExternalID:     "ext-1",
		Status:         status,
	}
}

func TestApplyActivate_FromPending(t *testing.T) {
	sub := makeSub(models.SubscriptionStatusPending)
	if err := domain.ApplyActivate(sub); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if sub.Status != models.SubscriptionStatusActive {
		t.Errorf("expected Active, got %v", sub.Status)
	}
	if sub.StartedAt == nil {
		t.Error("expected StartedAt to be set")
	}
}

func TestApplyActivate_AlreadyActive(t *testing.T) {
	sub := makeSub(models.SubscriptionStatusActive)
	if err := domain.ApplyActivate(sub); err != domain.ErrAlreadyActive {
		t.Errorf("expected ErrAlreadyActive, got %v", err)
	}
}

func TestApplyActivate_Terminated(t *testing.T) {
	sub := makeSub(models.SubscriptionStatusTerminated)
	if err := domain.ApplyActivate(sub); err != domain.ErrAlreadyTerminated {
		t.Errorf("expected ErrAlreadyTerminated, got %v", err)
	}
}

func TestApplyTerminate_ActiveToTerminated(t *testing.T) {
	sub := makeSub(models.SubscriptionStatusActive)
	if err := domain.ApplyTerminate(sub); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if sub.Status != models.SubscriptionStatusTerminated {
		t.Errorf("expected Terminated, got %v", sub.Status)
	}
	if sub.TerminatedAt == nil {
		t.Error("expected TerminatedAt to be set")
	}
}

func TestApplyTerminate_PendingToCanceled(t *testing.T) {
	sub := makeSub(models.SubscriptionStatusPending)
	if err := domain.ApplyTerminate(sub); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if sub.Status != models.SubscriptionStatusCanceled {
		t.Errorf("expected Canceled, got %v", sub.Status)
	}
	if sub.CanceledAt == nil {
		t.Error("expected CanceledAt to be set")
	}
}

func TestApplyTerminate_AlreadyTerminated(t *testing.T) {
	sub := makeSub(models.SubscriptionStatusTerminated)
	if err := domain.ApplyTerminate(sub); err != domain.ErrAlreadyTerminated {
		t.Errorf("expected ErrAlreadyTerminated, got %v", err)
	}
}

func TestApplyTerminate_AlreadyCanceled(t *testing.T) {
	sub := makeSub(models.SubscriptionStatusCanceled)
	if err := domain.ApplyTerminate(sub); err != domain.ErrAlreadyCanceled {
		t.Errorf("expected ErrAlreadyCanceled, got %v", err)
	}
}
