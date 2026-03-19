package kafka_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/getlago/lago/api-go/internal/kafka"
	"github.com/getlago/lago/api-go/internal/models"
)

func TestNoopPublisher_PublishRawEvent(t *testing.T) {
	publisher := &kafka.NoopPublisher{}
	inputEvent := &models.Event{
		SoftDeleteModel: models.SoftDeleteModel{BaseModel: models.BaseModel{ID: "evt-1"}},
		OrganizationID:  "org-1",
		TransactionID:   "tx-1",
		Code:            "usage.created",
	}

	err := publisher.PublishRawEvent(context.Background(), inputEvent)

	require.NoError(t, err)
}

func TestNewKafkaPublisher_RejectsEmptyBootstrapServers(t *testing.T) {
	_, err := kafka.NewKafkaPublisher(kafka.Config{
		BootstrapServers: "",
		Topic:            "events-raw",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bootstrap servers")
}

func TestNewKafkaPublisher_RejectsUnknownScramAlgorithm(t *testing.T) {
	_, err := kafka.NewKafkaPublisher(kafka.Config{
		BootstrapServers: "localhost:9092",
		Topic:            "events-raw",
		ScramAlgorithm:   "SCRAM-MD5",
		Username:         "user",
		Password:         "pass",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported scram algorithm")
}
