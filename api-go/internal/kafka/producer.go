package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/sasl/scram"

	"github.com/getlago/lago/api-go/internal/models"
)

const (
	scramSHA256 = "SCRAM-SHA-256"
	scramSHA512 = "SCRAM-SHA-512"
	sourceHTTPGo = "http_go"
)

// rawEventMessage mirrors the events-processor Event JSON shape so that the
// downstream processor can consume messages without modification.
type rawEventMessage struct {
	OrganizationID          string          `json:"organization_id"`
	ExternalSubscriptionID  string          `json:"external_subscription_id"`
	TransactionID           string          `json:"transaction_id"`
	Code                    string          `json:"code"`
	Properties              map[string]any  `json:"properties"`
	PreciseTotalAmountCents string          `json:"precise_total_amount_cents"`
	Source                  string          `json:"source"`
	Timestamp               float64         `json:"timestamp"`
	SourceMetadata          sourceMetadata  `json:"source_metadata"`
	IngestedAt              string          `json:"ingested_at"`
}

type sourceMetadata struct {
	APIPostProcessed bool `json:"api_post_processed"`
	Reprocess        bool `json:"reprocess"`
}

// EventPublisher publishes raw events to the Kafka events-raw topic.
type EventPublisher interface {
	PublishRawEvent(ctx context.Context, event *models.Event) error
}

// Config holds everything needed to create a Kafka producer.
type Config struct {
	BootstrapServers string
	Topic            string
	TLS              bool
	ScramAlgorithm   string
	Username         string
	Password         string
}

// KafkaPublisher sends events to Redpanda/Kafka using franz-go.
type KafkaPublisher struct {
	client *kgo.Client
	topic  string
}

// NewKafkaPublisher creates a KafkaPublisher or returns an error if the client
// cannot connect to the brokers.
func NewKafkaPublisher(cfg Config) (*KafkaPublisher, error) {
	if strings.TrimSpace(cfg.BootstrapServers) == "" {
		return nil, fmt.Errorf("kafka bootstrap servers must not be empty")
	}

	servers := strings.Split(cfg.BootstrapServers, ",")
	opts := []kgo.Opt{
		kgo.SeedBrokers(servers...),
		kgo.DefaultProduceTopic(cfg.Topic),
	}

	if cfg.TLS {
		opts = append(opts, kgo.DialTLS())
	}

	if cfg.ScramAlgorithm != "" {
		auth := scram.Auth{User: cfg.Username, Pass: cfg.Password}
		var saslOpt kgo.Opt
		switch cfg.ScramAlgorithm {
		case scramSHA256:
			saslOpt = kgo.SASL(auth.AsSha256Mechanism())
		case scramSHA512:
			saslOpt = kgo.SASL(auth.AsSha512Mechanism())
		default:
			return nil, fmt.Errorf("unsupported scram algorithm: %s", cfg.ScramAlgorithm)
		}
		opts = append(opts, saslOpt)
	}

	client, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka client: %w", err)
	}

	return &KafkaPublisher{client: client, topic: cfg.Topic}, nil
}

// PublishRawEvent serialises the event to the raw-events Kafka message schema
// and produces it synchronously. Errors are returned to the caller but should
// be treated as non-fatal (log and continue).
func (p *KafkaPublisher) PublishRawEvent(ctx context.Context, event *models.Event) error {
	msg := toRawEventMessage(event)

	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal raw event: %w", err)
	}

	record := &kgo.Record{
		Topic: p.topic,
		Key:   []byte(event.TransactionID),
		Value: payload,
	}

	result := p.client.ProduceSync(ctx, record)
	if err := result.FirstErr(); err != nil {
		return fmt.Errorf("kafka produce failed: %w", err)
	}

	return nil
}

// Close releases the underlying Kafka client resources.
func (p *KafkaPublisher) Close() {
	p.client.Close()
}

func toRawEventMessage(event *models.Event) rawEventMessage {
	var ts float64
	if event.Timestamp != nil {
		ts = float64(event.Timestamp.UnixMilli()) / 1000.0
	}

	extSubID := ""
	if event.ExternalSubscriptionID != nil {
		extSubID = *event.ExternalSubscriptionID
	}

	totalAmountCents := ""
	if event.PreciseTotalAmountCents != nil {
		totalAmountCents = *event.PreciseTotalAmountCents
	}

	properties := map[string]any{}
	if event.Properties != nil {
		properties = map[string]any(event.Properties)
	}

	return rawEventMessage{
		OrganizationID:          event.OrganizationID,
		ExternalSubscriptionID:  extSubID,
		TransactionID:           event.TransactionID,
		Code:                    event.Code,
		Properties:              properties,
		PreciseTotalAmountCents: totalAmountCents,
		Source:                  sourceHTTPGo,
		Timestamp:               ts,
		SourceMetadata:          sourceMetadata{APIPostProcessed: false, Reprocess: false},
		IngestedAt:              time.Now().UTC().Format(time.RFC3339Nano),
	}
}

// NoopPublisher is a no-op implementation used when Kafka is not configured or
// in unit tests.
type NoopPublisher struct{}

func (n *NoopPublisher) PublishRawEvent(_ context.Context, _ *models.Event) error {
	slog.Debug("kafka not configured, skipping raw event publish")
	return nil
}
