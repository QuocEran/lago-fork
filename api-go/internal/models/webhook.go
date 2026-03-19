package models

import "time"

// WebhookSignatureAlgo mirrors the Rails integer enum for signature_algo.
type WebhookSignatureAlgo int

const (
	WebhookSignatureAlgoHMACSHA256 WebhookSignatureAlgo = 0
	WebhookSignatureAlgoJWT        WebhookSignatureAlgo = 1
)

// WebhookEndpoint maps to the Rails `webhook_endpoints` table.
type WebhookEndpoint struct {
	BaseModel
	OrganizationID string               `gorm:"column:organization_id;not null;index"`
	WebhookURL     string               `gorm:"column:webhook_url;not null"`
	SignatureAlgo  WebhookSignatureAlgo `gorm:"column:signature_algo;not null;default:0"`
}

func (WebhookEndpoint) TableName() string { return "webhook_endpoints" }

// WebhookStatus mirrors the Rails integer enum for webhook status.
type WebhookStatus int

const (
	WebhookStatusPending   WebhookStatus = 0
	WebhookStatusSucceeded WebhookStatus = 1
	WebhookStatusFailed    WebhookStatus = 2
)

// Webhook maps to the Rails `webhooks` table.
// Each row represents one outbound delivery attempt record.
type Webhook struct {
	BaseModel
	OrganizationID    string        `gorm:"column:organization_id;not null;index"`
	WebhookEndpointID *string       `gorm:"column:webhook_endpoint_id;index"`
	ObjectID          *string       `gorm:"column:object_id"`
	ObjectType        *string       `gorm:"column:object_type"`
	WebhookType       *string       `gorm:"column:webhook_type"`
	Status            WebhookStatus `gorm:"column:status;not null;default:0;index"`
	Retries           int           `gorm:"column:retries;not null;default:0"`
	HTTPStatus        *int          `gorm:"column:http_status"`
	Endpoint          *string       `gorm:"column:endpoint"`
	Payload           JSONBMap      `gorm:"column:payload;type:jsonb"`
	Response          JSONBMap      `gorm:"column:response;type:jsonb"`
	LastRetriedAt     *time.Time    `gorm:"column:last_retried_at"`
}

func (Webhook) TableName() string { return "webhooks" }
