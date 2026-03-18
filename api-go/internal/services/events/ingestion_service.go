package events

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"

	"github.com/getlago/lago/api-go/internal/kafka"
	"github.com/getlago/lago/api-go/internal/models"
)

type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

func IsValidationError(err error) bool {
	var validationErr *ValidationError
	return errors.As(err, &validationErr)
}

type IngestEventInput struct {
	TransactionID          string
	Code                   string
	Timestamp              time.Time
	Properties             models.JSONBMap
	ExternalCustomerID     *string
	ExternalSubscriptionID *string
}

type IngestedEvent struct {
	Event   *models.Event
	Created bool
}

type Service interface {
	Ingest(ctx context.Context, organizationID string, input IngestEventInput) (*IngestedEvent, error)
	IngestBatch(ctx context.Context, organizationID string, inputs []IngestEventInput) ([]IngestedEvent, error)
	List(ctx context.Context, organizationID string, filter ListEventsFilter) ([]models.Event, *Pagination, error)
}

type service struct {
	db        *gorm.DB
	publisher kafka.EventPublisher
}

func NewService(db *gorm.DB, publisher kafka.EventPublisher) Service {
	return &service{db: db, publisher: publisher}
}

func (s *service) Ingest(ctx context.Context, organizationID string, input IngestEventInput) (*IngestedEvent, error) {
	if err := validateIngestInput(organizationID, input); err != nil {
		return nil, err
	}

	normalizedInput := normalizeIngestInput(input)

	var existing models.Event
	err := s.db.WithContext(ctx).
		Where("organization_id = ? AND transaction_id = ?", organizationID, normalizedInput.TransactionID).
		First(&existing).Error
	if err == nil {
		return &IngestedEvent{Event: &existing, Created: false}, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	event := models.Event{
		OrganizationID:         organizationID,
		TransactionID:          normalizedInput.TransactionID,
		Code:                   normalizedInput.Code,
		Timestamp:              &normalizedInput.Timestamp,
		Properties:             normalizedInput.Properties,
		Metadata:               models.JSONBMap{},
		ExternalCustomerID:     normalizedInput.ExternalCustomerID,
		ExternalSubscriptionID: normalizedInput.ExternalSubscriptionID,
	}

	if err := s.db.WithContext(ctx).Create(&event).Error; err != nil {
		if isUniqueViolation(err) {
			duplicate, fetchErr := s.fetchByOrganizationAndTransactionID(ctx, organizationID, normalizedInput.TransactionID)
			if fetchErr != nil {
				return nil, fetchErr
			}
			return &IngestedEvent{Event: duplicate, Created: false}, nil
		}
		return nil, err
	}

	s.publishRawEvent(ctx, &event)

	return &IngestedEvent{Event: &event, Created: true}, nil
}

func (s *service) IngestBatch(ctx context.Context, organizationID string, inputs []IngestEventInput) ([]IngestedEvent, error) {
	if len(inputs) == 0 {
		return nil, &ValidationError{Message: "events must contain at least one entry"}
	}

	results := make([]IngestedEvent, 0, len(inputs))
	for _, input := range inputs {
		ingested, err := s.Ingest(ctx, organizationID, input)
		if err != nil {
			return nil, err
		}
		results = append(results, *ingested)
	}

	return results, nil
}

func validateIngestInput(organizationID string, input IngestEventInput) error {
	if strings.TrimSpace(organizationID) == "" {
		return &ValidationError{Message: "organization_id is required"}
	}
	if strings.TrimSpace(input.TransactionID) == "" {
		return &ValidationError{Message: "transaction_id is required"}
	}
	if strings.TrimSpace(input.Code) == "" {
		return &ValidationError{Message: "code is required"}
	}

	return nil
}

func normalizeIngestInput(input IngestEventInput) IngestEventInput {
	if input.Timestamp.IsZero() {
		input.Timestamp = time.Now().UTC()
	}
	if input.Properties == nil {
		input.Properties = models.JSONBMap{}
	}
	input.TransactionID = strings.TrimSpace(input.TransactionID)
	input.Code = strings.TrimSpace(input.Code)
	return input
}

func (s *service) fetchByOrganizationAndTransactionID(ctx context.Context, organizationID string, transactionID string) (*models.Event, error) {
	var event models.Event
	err := s.db.WithContext(ctx).
		Where("organization_id = ? AND transaction_id = ?", organizationID, transactionID).
		First(&event).Error
	if err != nil {
		return nil, err
	}

	return &event, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}

	// Fallback for wrapped or translated driver errors.
	errText := strings.ToLower(err.Error())
	return strings.Contains(errText, "duplicate key") || strings.Contains(errText, "unique constraint")
}

// publishRawEvent sends the event to the Kafka events-raw topic. Failures are
// logged as warnings but never propagate to the caller — ingestion must succeed
// even when the broker is unreachable.
func (s *service) publishRawEvent(ctx context.Context, event *models.Event) {
	if err := s.publisher.PublishRawEvent(ctx, event); err != nil {
		slog.Warn("failed to publish raw event to kafka",
			slog.String("transaction_id", event.TransactionID),
			slog.String("organization_id", event.OrganizationID),
			slog.String("error", err.Error()),
		)
	}
}

// ListEventsFilter holds optional query parameters for the events listing.
type ListEventsFilter struct {
	Code                   string
	ExternalSubscriptionID string
	TimestampFrom          *time.Time
	TimestampTo            *time.Time
	Page                   int
	PerPage                int
}

// Pagination carries the metadata returned alongside a paginated result set.
type Pagination struct {
	CurrentPage int
	NextPage    *int
	PrevPage    *int
	TotalPages  int
	TotalCount  int64
}

const defaultPerPage = 20
const maxPerPage = 100

// List returns a page of events for the given organisation, applying the
// supplied filter criteria.
func (s *service) List(ctx context.Context, organizationID string, filter ListEventsFilter) ([]models.Event, *Pagination, error) {
	perPage := filter.PerPage
	if perPage <= 0 {
		perPage = defaultPerPage
	}
	if perPage > maxPerPage {
		perPage = maxPerPage
	}
	page := filter.Page
	if page <= 0 {
		page = 1
	}

	query := s.db.WithContext(ctx).
		Model(&models.Event{}).
		Where("organization_id = ? AND deleted_at IS NULL", organizationID)

	if filter.Code != "" {
		query = query.Where("code = ?", filter.Code)
	}
	if filter.ExternalSubscriptionID != "" {
		query = query.Where("external_subscription_id = ?", filter.ExternalSubscriptionID)
	}
	if filter.TimestampFrom != nil {
		query = query.Where("timestamp >= ?", *filter.TimestampFrom)
	}
	if filter.TimestampTo != nil {
		query = query.Where("timestamp <= ?", *filter.TimestampTo)
	}

	var totalCount int64
	if err := query.Count(&totalCount).Error; err != nil {
		return nil, nil, err
	}

	totalPages := int(totalCount) / perPage
	if int(totalCount)%perPage != 0 {
		totalPages++
	}
	if totalPages == 0 {
		totalPages = 1
	}

	var events []models.Event
	offset := (page - 1) * perPage
	if err := query.Order("timestamp DESC, created_at DESC").Offset(offset).Limit(perPage).Find(&events).Error; err != nil {
		return nil, nil, err
	}

	pagination := &Pagination{
		CurrentPage: page,
		TotalPages:  totalPages,
		TotalCount:  totalCount,
	}
	if page > 1 {
		prev := page - 1
		pagination.PrevPage = &prev
	}
	if page < totalPages {
		next := page + 1
		pagination.NextPage = &next
	}

	return events, pagination, nil
}
