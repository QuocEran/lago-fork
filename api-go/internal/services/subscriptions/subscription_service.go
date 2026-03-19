package subscriptions

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	domain "github.com/getlago/lago/api-go/internal/domain/subscriptions"
	"github.com/getlago/lago/api-go/internal/models"
)

var (
	// ErrSubscriptionNotFound is returned when no subscription matches the lookup.
	ErrSubscriptionNotFound = errors.New("subscription_not_found")
	// ErrExternalIDConflict is returned when an active/pending subscription already uses the external_id.
	ErrExternalIDConflict = errors.New("subscription_external_id_taken")
)

// ValidationError wraps a user-facing validation message.
type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }

// IsValidationError reports whether err is a ValidationError.
func IsValidationError(err error) bool {
	var v *ValidationError
	return errors.As(err, &v)
}

// CreateInput holds fields needed to create a subscription.
type CreateInput struct {
	CustomerID     string     `json:"customer_id"`
	PlanID         string     `json:"plan_id"`
	ExternalID     string     `json:"external_id"`
	Name           *string    `json:"name"`
	BillingTime    string     `json:"billing_time"`
	SubscriptionAt *time.Time `json:"subscription_at"`
	EndingAt       *time.Time `json:"ending_at"`
}

// UpdateInput holds mutable fields for updating a subscription.
type UpdateInput struct {
	Name           *string    `json:"name"`
	SubscriptionAt *time.Time `json:"subscription_at"`
	EndingAt       *time.Time `json:"ending_at"`
}

// ListFilter defines filter + pagination for listing subscriptions.
type ListFilter struct {
	CustomerExternalID string
	Status             []string
	Page               int
	PerPage            int
}

// ListResult holds the result of a List operation.
type ListResult struct {
	Subscriptions []models.Subscription
	TotalCount    int64
	TotalPages    int
	CurrentPage   int
	NextPage      *int
	PrevPage      *int
}

// Service defines the subscriptions business-logic contract.
type Service interface {
	Create(ctx context.Context, organizationID string, input CreateInput) (*models.Subscription, error)
	GetByID(ctx context.Context, organizationID, id string) (*models.Subscription, error)
	GetByExternalID(ctx context.Context, organizationID, externalID string) (*models.Subscription, error)
	List(ctx context.Context, organizationID string, filter ListFilter) (*ListResult, error)
	Update(ctx context.Context, organizationID, externalID string, input UpdateInput) (*models.Subscription, error)
	Terminate(ctx context.Context, organizationID, id string) (*models.Subscription, error)
}

// NewService constructs the subscription service.
func NewService(db *gorm.DB) Service {
	return &service{db: db}
}

type service struct {
	db *gorm.DB
}

func (s *service) Create(ctx context.Context, organizationID string, input CreateInput) (*models.Subscription, error) {
	if err := validateCreate(organizationID, input); err != nil {
		return nil, err
	}

	// Resolve customer.
	var customer models.Customer
	if err := s.db.WithContext(ctx).
		Where("organization_id = ? AND external_id = ? AND deleted_at IS NULL", organizationID, input.CustomerID).
		First(&customer).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &ValidationError{Message: "customer_not_found"}
		}
		return nil, err
	}

	// Resolve plan.
	var plan models.Plan
	if err := s.db.WithContext(ctx).
		Where("organization_id = ? AND id = ? AND deleted_at IS NULL", organizationID, input.PlanID).
		First(&plan).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &ValidationError{Message: "plan_not_found"}
		}
		return nil, err
	}

	// Check external_id uniqueness within active/pending subscriptions.
	var count int64
	s.db.WithContext(ctx).Model(&models.Subscription{}).
		Where("organization_id = ? AND external_id = ? AND status IN (?,?)",
			organizationID, input.ExternalID,
			models.SubscriptionStatusPending, models.SubscriptionStatusActive).
		Count(&count)
	if count > 0 {
		return nil, ErrExternalIDConflict
	}

	billingTime, _ := models.BillingTimeFromString(input.BillingTime)

	// Determine initial status based on subscription_at.
	now := time.Now()
	status := models.SubscriptionStatusActive
	var startedAt *time.Time
	if input.SubscriptionAt != nil && input.SubscriptionAt.After(now) {
		status = models.SubscriptionStatusPending
	} else {
		status = models.SubscriptionStatusActive
		startedAt = &now
		if input.SubscriptionAt != nil {
			startedAt = input.SubscriptionAt
		}
	}

	sub := &models.Subscription{
		OrganizationID: organizationID,
		CustomerID:     customer.ID,
		PlanID:         plan.ID,
		ExternalID:     strings.TrimSpace(input.ExternalID),
		Name:           input.Name,
		BillingTime:    billingTime,
		Status:         status,
		SubscriptionAt: input.SubscriptionAt,
		StartedAt:      startedAt,
		EndingAt:       input.EndingAt,
	}

	if err := s.db.WithContext(ctx).Create(sub).Error; err != nil {
		return nil, err
	}

	return s.loadSubscription(ctx, sub.ID)
}

func (s *service) GetByID(ctx context.Context, organizationID, id string) (*models.Subscription, error) {
	var sub models.Subscription
	err := s.db.WithContext(ctx).
		Preload("Customer").Preload("Plan").
		Where("organization_id = ? AND id = ?", organizationID, id).
		First(&sub).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrSubscriptionNotFound
	}
	return &sub, err
}

func (s *service) GetByExternalID(ctx context.Context, organizationID, externalID string) (*models.Subscription, error) {
	var sub models.Subscription
	err := s.db.WithContext(ctx).
		Preload("Customer").Preload("Plan").
		Where("organization_id = ? AND external_id = ? AND status IN (?,?)",
			organizationID, externalID,
			models.SubscriptionStatusPending, models.SubscriptionStatusActive).
		Order("created_at DESC").
		First(&sub).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrSubscriptionNotFound
	}
	return &sub, err
}

func (s *service) List(ctx context.Context, organizationID string, filter ListFilter) (*ListResult, error) {
	page, perPage := normalizePagination(filter.Page, filter.PerPage)

	q := s.db.WithContext(ctx).Model(&models.Subscription{}).
		Where("subscriptions.organization_id = ?", organizationID)

	if filter.CustomerExternalID != "" {
		q = q.Joins("JOIN customers ON customers.id = subscriptions.customer_id").
			Where("customers.external_id = ?", filter.CustomerExternalID)
	}

	if len(filter.Status) > 0 {
		statuses := make([]int, 0, len(filter.Status))
		for _, s := range filter.Status {
			if v, ok := models.SubscriptionStatusFromString(s); ok {
				statuses = append(statuses, int(v))
			}
		}
		if len(statuses) > 0 {
			q = q.Where("subscriptions.status IN ?", statuses)
		}
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, err
	}

	var subs []models.Subscription
	offset := (page - 1) * perPage
	if err := q.Preload("Customer").Preload("Plan").
		Order("subscriptions.created_at DESC").
		Limit(perPage).Offset(offset).Find(&subs).Error; err != nil {
		return nil, err
	}

	totalPages := int((total + int64(perPage) - 1) / int64(perPage))
	result := &ListResult{
		Subscriptions: subs,
		TotalCount:    total,
		TotalPages:    totalPages,
		CurrentPage:   page,
	}
	if page < totalPages {
		next := page + 1
		result.NextPage = &next
	}
	if page > 1 {
		prev := page - 1
		result.PrevPage = &prev
	}
	return result, nil
}

func (s *service) Update(ctx context.Context, organizationID, externalID string, input UpdateInput) (*models.Subscription, error) {
	sub, err := s.GetByExternalID(ctx, organizationID, externalID)
	if err != nil {
		return nil, err
	}

	updates := map[string]any{}
	if input.Name != nil {
		updates["name"] = *input.Name
	}
	if input.SubscriptionAt != nil {
		updates["subscription_at"] = *input.SubscriptionAt
	}
	if input.EndingAt != nil {
		updates["ending_at"] = *input.EndingAt
	}

	if len(updates) > 0 {
		if err := s.db.WithContext(ctx).Model(sub).Updates(updates).Error; err != nil {
			return nil, err
		}
	}

	return s.loadSubscription(ctx, sub.ID)
}

func (s *service) Terminate(ctx context.Context, organizationID, id string) (*models.Subscription, error) {
	sub, err := s.GetByID(ctx, organizationID, id)
	if err != nil {
		return nil, err
	}

	if err := domain.ApplyTerminate(sub); err != nil {
		if errors.Is(err, domain.ErrAlreadyTerminated) || errors.Is(err, domain.ErrAlreadyCanceled) {
			return nil, &ValidationError{Message: err.Error()}
		}
		return nil, &ValidationError{Message: err.Error()}
	}

	updates := map[string]any{"status": int(sub.Status)}
	if sub.TerminatedAt != nil {
		updates["terminated_at"] = sub.TerminatedAt
	}
	if sub.CanceledAt != nil {
		updates["canceled_at"] = sub.CanceledAt
	}

	if err := s.db.WithContext(ctx).Model(sub).Updates(updates).Error; err != nil {
		return nil, err
	}

	return s.loadSubscription(ctx, sub.ID)
}

// loadSubscription reloads a subscription with its associations.
func (s *service) loadSubscription(ctx context.Context, id string) (*models.Subscription, error) {
	var sub models.Subscription
	err := s.db.WithContext(ctx).
		Preload("Customer").Preload("Plan").
		Where("id = ?", id).
		First(&sub).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrSubscriptionNotFound
	}
	return &sub, err
}

func validateCreate(organizationID string, input CreateInput) error {
	if strings.TrimSpace(organizationID) == "" {
		return &ValidationError{Message: "organization_id is required"}
	}
	if strings.TrimSpace(input.CustomerID) == "" {
		return &ValidationError{Message: "customer_id is required"}
	}
	if strings.TrimSpace(input.PlanID) == "" {
		return &ValidationError{Message: "plan_id is required"}
	}
	if strings.TrimSpace(input.ExternalID) == "" {
		return &ValidationError{Message: "external_id is required"}
	}
	return nil
}

func normalizePagination(page, perPage int) (int, int) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	return page, perPage
}
