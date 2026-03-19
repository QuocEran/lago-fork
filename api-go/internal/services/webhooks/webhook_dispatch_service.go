package webhooks

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	"gorm.io/gorm"

	"github.com/getlago/lago/api-go/internal/jobs"
	"github.com/getlago/lago/api-go/internal/models"
)

// DispatchParams holds the data needed to fan out a webhook event.
type DispatchParams struct {
	OrgID      string
	EventType  string
	ObjectID   string
	ObjectType string
	Payload    map[string]any
}

// DispatchService creates Webhook records per endpoint and enqueues delivery jobs.
type DispatchService interface {
	Dispatch(ctx context.Context, client *asynq.Client, params DispatchParams) error
}

type dispatchService struct {
	db *gorm.DB
}

// NewDispatchService creates a DispatchService backed by db.
func NewDispatchService(db *gorm.DB) DispatchService {
	return &dispatchService{db: db}
}

func (s *dispatchService) Dispatch(ctx context.Context, client *asynq.Client, params DispatchParams) error {
	var endpoints []models.WebhookEndpoint
	if err := s.db.Where("organization_id = ?", params.OrgID).Find(&endpoints).Error; err != nil {
		return fmt.Errorf("fetch webhook endpoints: %w", err)
	}
	if len(endpoints) == 0 {
		return nil
	}

	payloadBytes, err := json.Marshal(params.Payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}
	payloadMap := models.JSONBMap{}
	if err := json.Unmarshal(payloadBytes, &payloadMap); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	for i := range endpoints {
		ep := &endpoints[i]
		endpointURL := ep.WebhookURL
		objID := &params.ObjectID
		objType := &params.ObjectType
		wType := &params.EventType

		wh := &models.Webhook{
			OrganizationID:    params.OrgID,
			WebhookEndpointID: &ep.ID,
			ObjectID:          objID,
			ObjectType:        objType,
			WebhookType:       wType,
			Status:            models.WebhookStatusPending,
			Endpoint:          &endpointURL,
			Payload:           payloadMap,
		}
		if err := s.db.Create(wh).Error; err != nil {
			return fmt.Errorf("create webhook record for endpoint %s: %w", ep.ID, err)
		}

		if _, err := jobs.EnqueueSendHTTPWebhook(ctx, client, wh.ID); err != nil {
			return fmt.Errorf("enqueue send_http_webhook for %s: %w", wh.ID, err)
		}
	}
	return nil
}
