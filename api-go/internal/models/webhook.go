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

// InboundWebhookStatus represents the processing state of an inbound webhook.
type InboundWebhookStatus string

const (
	InboundWebhookStatusPending    InboundWebhookStatus = "pending"
	InboundWebhookStatusProcessing InboundWebhookStatus = "processing"
	InboundWebhookStatusSucceeded  InboundWebhookStatus = "succeeded"
	InboundWebhookStatusFailed     InboundWebhookStatus = "failed"
)

// InboundWebhook records an inbound callback from an external payment provider.
type InboundWebhook struct {
	BaseModel
	OrganizationID string               `gorm:"column:organization_id;not null;index"`
	Source         string               `gorm:"column:source;not null;index"`
	EventType      string               `gorm:"column:event_type;not null;default:''"`
	Payload        JSONBMap             `gorm:"column:payload;type:jsonb;not null"`
	Status         InboundWebhookStatus `gorm:"column:status;not null;default:'pending';index"`
	Code           *string              `gorm:"column:code"`
	Signature      *string              `gorm:"column:signature"`
	ProcessingAt   *time.Time           `gorm:"column:processing_at"`
}

func (InboundWebhook) TableName() string { return "inbound_webhooks" }

