package webhook_endpoints

import (
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
	Create(orgID string, params CreateParams) (*models.WebhookEndpoint, error)
	List(orgID string, page, limit int) ([]models.WebhookEndpoint, int64, error)
	GetByID(orgID, id string) (*models.WebhookEndpoint, error)
	Update(orgID, id string, params UpdateParams) (*models.WebhookEndpoint, error)
	Delete(orgID, id string) (*models.WebhookEndpoint, error)
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

func (s *service) Create(orgID string, params CreateParams) (*models.WebhookEndpoint, error) {
	if err := validateURL(params.WebhookURL); err != nil {
		return nil, err
	}

	var count int64
	if err := s.db.Model(&models.WebhookEndpoint{}).Where("organization_id = ?", orgID).Count(&count).Error; err != nil {
		return nil, err
	}
	if count >= maxWebhookEndpointsPerOrg {
		return nil, ErrMaxEndpointsReached
	}

	var dup models.WebhookEndpoint
	err := s.db.Where("organization_id = ? AND webhook_url = ?", orgID, params.WebhookURL).First(&dup).Error
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

	if err := s.db.Create(ep).Error; err != nil {
		return nil, err
	}
	return ep, nil
}

func (s *service) List(orgID string, page, limit int) ([]models.WebhookEndpoint, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	var total int64
	if err := s.db.Model(&models.WebhookEndpoint{}).Where("organization_id = ?", orgID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var eps []models.WebhookEndpoint
	if err := s.db.Where("organization_id = ?", orgID).
		Order("created_at ASC").
		Limit(limit).Offset(offset).
		Find(&eps).Error; err != nil {
		return nil, 0, err
	}
	return eps, total, nil
}

func (s *service) GetByID(orgID, id string) (*models.WebhookEndpoint, error) {
	var ep models.WebhookEndpoint
	err := s.db.Where("id = ? AND organization_id = ?", id, orgID).First(&ep).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrWebhookEndpointNotFound
	}
	return &ep, err
}

func (s *service) Update(orgID, id string, params UpdateParams) (*models.WebhookEndpoint, error) {
	ep, err := s.GetByID(orgID, id)
	if err != nil {
		return nil, err
	}

	updates := map[string]any{}
	if params.WebhookURL != nil {
		if err := validateURL(*params.WebhookURL); err != nil {
			return nil, err
		}
		// Check uniqueness, excluding self.
		var dup models.WebhookEndpoint
		dupErr := s.db.Where("organization_id = ? AND webhook_url = ? AND id != ?", orgID, *params.WebhookURL, id).First(&dup).Error
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

	if err := s.db.Model(ep).Updates(updates).Error; err != nil {
		return nil, err
	}
	return ep, nil
}

func (s *service) Delete(orgID, id string) (*models.WebhookEndpoint, error) {
	ep, err := s.GetByID(orgID, id)
	if err != nil {
		return nil, err
	}
	if err := s.db.Delete(ep).Error; err != nil {
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
