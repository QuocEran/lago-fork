package webhookendpoints

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"gorm.io/gorm"

	"github.com/getlago/lago/api-go/internal/models"
)

const maxWebhookEndpointsPerOrg = 10

var (
	ErrWebhookEndpointNotFound = errors.New("webhook endpoint not found")
	ErrWebhookURLConflict      = errors.New("webhook URL already exists for this organization")
	ErrMaxEndpointsReached     = errors.New("maximum webhook endpoints per organization reached")
	ErrInvalidWebhookURL       = errors.New("webhook URL is invalid")
)

// Service manages webhook endpoints for an organization.
type Service interface {
	Create(ctx context.Context, orgID string, params CreateParams) (*models.WebhookEndpoint, error)
	List(ctx context.Context, orgID string, page, limit int) ([]models.WebhookEndpoint, int64, error)
	GetByID(ctx context.Context, orgID, id string) (*models.WebhookEndpoint, error)
	Update(ctx context.Context, orgID, id string, params UpdateParams) (*models.WebhookEndpoint, error)
	Delete(ctx context.Context, orgID, id string) (*models.WebhookEndpoint, error)
}

// CreateParams holds fields for creating a webhook endpoint.
type CreateParams struct {
	ID            *string
	WebhookURL    string
	SignatureAlgo models.WebhookSignatureAlgo
}

// UpdateParams holds fields for updating a webhook endpoint.
type UpdateParams struct {
	WebhookURL    *string
	SignatureAlgo *models.WebhookSignatureAlgo
}

// ValidationError is returned for business-rule violations.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error on %s: %s", e.Field, e.Message)
}

// IsValidationError reports whether err is a ValidationError.
func IsValidationError(err error) bool {
	var ve *ValidationError
	return errors.As(err, &ve)
}

type service struct {
	db *gorm.DB
}

// NewService creates a webhook endpoint Service backed by db.
func NewService(db *gorm.DB) Service {
	return &service{db: db}
}

func (s *service) Create(ctx context.Context, orgID string, params CreateParams) (*models.WebhookEndpoint, error) {
	if err := validateURL(params.WebhookURL); err != nil {
		return nil, err
	}

	db := s.db.WithContext(ctx)
	var count int64
	if err := db.Model(&models.WebhookEndpoint{}).Where("organization_id = ?", orgID).Count(&count).Error; err != nil {
		return nil, err
	}
	if count >= maxWebhookEndpointsPerOrg {
		return nil, ErrMaxEndpointsReached
	}

	var dup models.WebhookEndpoint
	err := db.Where("organization_id = ? AND webhook_url = ?", orgID, params.WebhookURL).First(&dup).Error
	if err == nil {
		return nil, ErrWebhookURLConflict
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	ep := &models.WebhookEndpoint{
		OrganizationID: orgID,
		WebhookURL:     params.WebhookURL,
		SignatureAlgo:  params.SignatureAlgo,
	}
	if params.ID != nil && *params.ID != "" {
		ep.ID = *params.ID
	}

	if err := db.Create(ep).Error; err != nil {
		return nil, err
	}
	return ep, nil
}

func (s *service) List(ctx context.Context, orgID string, page, limit int) ([]models.WebhookEndpoint, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	db := s.db.WithContext(ctx)
	var total int64
	if err := db.Model(&models.WebhookEndpoint{}).Where("organization_id = ?", orgID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var eps []models.WebhookEndpoint
	if err := db.Where("organization_id = ?", orgID).
		Order("created_at ASC").
		Limit(limit).Offset(offset).
		Find(&eps).Error; err != nil {
		return nil, 0, err
	}
	return eps, total, nil
}

func (s *service) GetByID(ctx context.Context, orgID, id string) (*models.WebhookEndpoint, error) {
	var ep models.WebhookEndpoint
	err := s.db.WithContext(ctx).Where("id = ? AND organization_id = ?", id, orgID).First(&ep).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrWebhookEndpointNotFound
	}
	return &ep, err
}

func (s *service) Update(ctx context.Context, orgID, id string, params UpdateParams) (*models.WebhookEndpoint, error) {
	ep, err := s.GetByID(ctx, orgID, id)
	if err != nil {
		return nil, err
	}

	db := s.db.WithContext(ctx)
	updates := map[string]any{}
	if params.WebhookURL != nil {
		if err := validateURL(*params.WebhookURL); err != nil {
			return nil, err
		}
		// Check uniqueness, excluding self.
		var dup models.WebhookEndpoint
		dupErr := db.Where("organization_id = ? AND webhook_url = ? AND id != ?", orgID, *params.WebhookURL, id).First(&dup).Error
		if dupErr == nil {
			return nil, ErrWebhookURLConflict
		} else if !errors.Is(dupErr, gorm.ErrRecordNotFound) {
			return nil, dupErr
		}
		updates["webhook_url"] = *params.WebhookURL
	}
	if params.SignatureAlgo != nil {
		updates["signature_algo"] = *params.SignatureAlgo
	}
	if len(updates) == 0 {
		return ep, nil
	}

	if err := db.Model(ep).Updates(updates).Error; err != nil {
		return nil, err
	}
	return ep, nil
}

func (s *service) Delete(ctx context.Context, orgID, id string) (*models.WebhookEndpoint, error) {
	ep, err := s.GetByID(ctx, orgID, id)
	if err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Delete(ep).Error; err != nil {
		return nil, err
	}
	return ep, nil
}

func validateURL(raw string) error {
	if raw == "" {
		return &ValidationError{Field: "webhook_url", Message: "cannot be blank"}
	}
	u, err := url.ParseRequestURI(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return &ValidationError{Field: "webhook_url", Message: "must be a valid http/https URL"}
	}
	return nil
}
