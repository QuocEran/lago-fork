package webhooks

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"

	"gorm.io/gorm"

	"github.com/getlago/lago/api-go/internal/jobs/handlers"
	"github.com/getlago/lago/api-go/internal/models"
)

// ErrSignatureInvalid is returned when the webhook signature verification fails.
var ErrSignatureInvalid = errors.New("webhook signature invalid")

// InboundService handles incoming webhook payloads from payment providers.
type InboundService interface {
	HandleStripeWebhook(ctx context.Context, orgID string, signature string, rawBody []byte, code string) error
	HandleGocardlessWebhook(ctx context.Context, orgID string, rawBody []byte) error
}

type inboundService struct {
	db            *gorm.DB
	stripeSecret  string
}

// NewInboundService creates an InboundService. stripeSecret is used to verify Stripe webhook signatures.
func NewInboundService(db *gorm.DB, stripeSecret string) InboundService {
	return &inboundService{db: db, stripeSecret: stripeSecret}
}

func (s *inboundService) HandleStripeWebhook(ctx context.Context, orgID string, signature string, rawBody []byte, code string) error {
	if s.stripeSecret != "" && signature != "" {
		if !handlers.VerifySignature(s.stripeSecret, bytes.NewReader(rawBody), signature) {
			return ErrSignatureInvalid
		}
	}

	var payloadMap models.JSONBMap
	if err := json.Unmarshal(rawBody, &payloadMap); err != nil {
		payloadMap = models.JSONBMap{}
	}

	eventType, _ := payloadMap["type"].(string)
	var codePtr, sigPtr *string
	if code != "" {
		codePtr = &code
	}
	if signature != "" {
		sigPtr = &signature
	}

	inbound := &models.InboundWebhook{
		OrganizationID: orgID,
		Source:         "stripe",
		EventType:      eventType,
		Payload:        payloadMap,
		Status:         models.InboundWebhookStatusPending,
		Code:           codePtr,
		Signature:      sigPtr,
	}
	return s.db.WithContext(ctx).Create(inbound).Error
}

func (s *inboundService) HandleGocardlessWebhook(ctx context.Context, orgID string, rawBody []byte) error {
	var payloadMap models.JSONBMap
	if err := json.Unmarshal(rawBody, &payloadMap); err != nil {
		payloadMap = models.JSONBMap{}
	}

	inbound := &models.InboundWebhook{
		OrganizationID: orgID,
		Source:         "gocardless",
		EventType:      "",
		Payload:        payloadMap,
		Status:         models.InboundWebhookStatusPending,
	}
	return s.db.WithContext(ctx).Create(inbound).Error
}
