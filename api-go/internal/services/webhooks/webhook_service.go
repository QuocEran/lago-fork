package webhooks

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/getlago/lago/api-go/internal/models"
)

// Service provides read access to outbound webhooks for an organization.
type Service interface {
	List(ctx context.Context, orgID string, page, perPage int) ([]models.Webhook, int64, error)
	GetByID(ctx context.Context, orgID, id string) (*models.Webhook, error)
}

var ErrWebhookNotFound = errors.New("webhook not found")

type service struct {
	db *gorm.DB
}

// NewService creates a webhook read service backed by db.
func NewService(db *gorm.DB) Service {
	return &service{db: db}
}

func (s *service) List(ctx context.Context, orgID string, page, perPage int) ([]models.Webhook, int64, error) {
	if perPage <= 0 {
		perPage = 20
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * perPage

	var total int64
	if err := s.db.WithContext(ctx).Model(&models.Webhook{}).Where("organization_id = ?", orgID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var list []models.Webhook
	if err := s.db.WithContext(ctx).Where("organization_id = ?", orgID).
		Order("created_at DESC").
		Limit(perPage).Offset(offset).
		Find(&list).Error; err != nil {
		return nil, 0, err
	}

	return list, total, nil
}

func (s *service) GetByID(ctx context.Context, orgID, id string) (*models.Webhook, error) {
	var wh models.Webhook
	err := s.db.WithContext(ctx).Where("id = ? AND organization_id = ?", id, orgID).First(&wh).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrWebhookNotFound
		}
		return nil, err
	}
	return &wh, nil
}
