package billablemetrics

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"

	"github.com/getlago/lago/api-go/internal/models"
)

// ErrBillableMetricNotFound is returned when no billable metric matches the lookup.
var ErrBillableMetricNotFound = errors.New("billable_metric_not_found")

// ErrBillableMetricCodeConflict is returned when a code already exists in the org.
var ErrBillableMetricCodeConflict = errors.New("billable_metric_code_taken")

// ValidationError wraps a user-facing validation message.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string { return e.Message }

// IsValidationError reports whether err is a ValidationError.
func IsValidationError(err error) bool {
	var v *ValidationError
	return errors.As(err, &v)
}

// FilterInput represents a single filter key/values pair used in create/update.
type FilterInput struct {
	Key    string   `json:"key"`
	Values []string `json:"values"`
}

// CreateInput holds the fields needed to create a billable metric.
type CreateInput struct {
	Name              string        `json:"name"`
	Code              string        `json:"code"`
	Description       *string       `json:"description"`
	AggregationType   string        `json:"aggregation_type"`
	FieldName         *string       `json:"field_name"`
	Recurring         *bool         `json:"recurring"`
	Expression        *string       `json:"expression"`
	CustomAggregator  *string       `json:"custom_aggregator"`
	WeightedInterval  *string       `json:"weighted_interval"`
	RoundingFunction  *string       `json:"rounding_function"`
	RoundingPrecision *int          `json:"rounding_precision"`
	Filters           []FilterInput `json:"filters"`
}

// UpdateInput holds the mutable fields for updating a billable metric.
type UpdateInput struct {
	Name              *string       `json:"name"`
	Description       *string       `json:"description"`
	AggregationType   *string       `json:"aggregation_type"`
	FieldName         *string       `json:"field_name"`
	Recurring         *bool         `json:"recurring"`
	Expression        *string       `json:"expression"`
	CustomAggregator  *string       `json:"custom_aggregator"`
	WeightedInterval  *string       `json:"weighted_interval"`
	RoundingFunction  *string       `json:"rounding_function"`
	RoundingPrecision *int          `json:"rounding_precision"`
	Filters           *[]FilterInput `json:"filters"`
}

// ListFilter defines pagination + search parameters for listing.
type ListFilter struct {
	Page     int
	PerPage  int
	Search   string
	Recurring *bool
}

// ListResult holds the result of a List operation.
type ListResult struct {
	Metrics     []models.BillableMetric
	TotalCount  int64
	TotalPages  int
	CurrentPage int
	NextPage    *int
	PrevPage    *int
}

// Service defines the billable metrics business-logic contract.
type Service interface {
	Create(ctx context.Context, organizationID string, input CreateInput) (*models.BillableMetric, error)
	List(ctx context.Context, organizationID string, filter ListFilter) (*ListResult, error)
	GetByCode(ctx context.Context, organizationID string, code string) (*models.BillableMetric, error)
	GetByID(ctx context.Context, organizationID string, id string) (*models.BillableMetric, error)
	Update(ctx context.Context, organizationID string, code string, input UpdateInput) (*models.BillableMetric, error)
	Delete(ctx context.Context, organizationID string, code string) (*models.BillableMetric, error)
}

type service struct {
	db *gorm.DB
}

// NewService constructs a billable metrics service.
func NewService(db *gorm.DB) Service {
	return &service{db: db}
}

func (s *service) Create(ctx context.Context, organizationID string, input CreateInput) (*models.BillableMetric, error) {
	if err := validateCreateInput(organizationID, input); err != nil {
		return nil, err
	}

	aggType, _ := models.AggregationTypeFromString(input.AggregationType)

	metric := models.BillableMetric{
		OrganizationID:    organizationID,
		Name:              strings.TrimSpace(input.Name),
		Code:              strings.TrimSpace(input.Code),
		Description:       input.Description,
		AggregationType:   aggType,
		FieldName:         input.FieldName,
		Recurring:         boolVal(input.Recurring),
		Expression:        input.Expression,
		CustomAggregator:  input.CustomAggregator,
		WeightedInterval:  input.WeightedInterval,
		RoundingFunction:  input.RoundingFunction,
		RoundingPrecision: input.RoundingPrecision,
	}

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Check code uniqueness (non-deleted) within org.
		var existing models.BillableMetric
		if err := tx.Where("organization_id = ? AND code = ? AND deleted_at IS NULL", organizationID, metric.Code).
			First(&existing).Error; err == nil {
			return ErrBillableMetricCodeConflict
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		if err := tx.Create(&metric).Error; err != nil {
			return err
		}

		return syncFilters(tx, metric.ID, organizationID, input.Filters)
	})
	if err != nil {
		return nil, err
	}

	return s.GetByCode(ctx, organizationID, metric.Code)
}

func (s *service) List(ctx context.Context, organizationID string, filter ListFilter) (*ListResult, error) {
	if strings.TrimSpace(organizationID) == "" {
		return nil, &ValidationError{Message: "organization_id is required"}
	}

	page, perPage := normalizePagination(filter.Page, filter.PerPage)

	q := s.db.WithContext(ctx).
		Model(&models.BillableMetric{}).
		Where("organization_id = ? AND deleted_at IS NULL", organizationID)

	if filter.Search != "" {
		like := "%" + filter.Search + "%"
		q = q.Where("name ILIKE ? OR code ILIKE ?", like, like)
	}
	if filter.Recurring != nil {
		q = q.Where("recurring = ?", *filter.Recurring)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, err
	}

	var metrics []models.BillableMetric
	offset := (page - 1) * perPage
	if err := q.
		Preload("Filters", "deleted_at IS NULL").
		Order("created_at DESC").
		Limit(perPage).
		Offset(offset).
		Find(&metrics).Error; err != nil {
		return nil, err
	}

	totalPages := int((total + int64(perPage) - 1) / int64(perPage))
	if totalPages == 0 {
		totalPages = 1
	}

	result := &ListResult{
		Metrics:     metrics,
		TotalCount:  total,
		TotalPages:  totalPages,
		CurrentPage: page,
	}
	if page < totalPages {
		next := page + 1
		result.NextPage = &next
	}
	if page > 1 {
		prev := page - 1
		result.PrevPage = &prev
	}

	return result, nil
}

func (s *service) GetByCode(ctx context.Context, organizationID string, code string) (*models.BillableMetric, error) {
	if strings.TrimSpace(organizationID) == "" {
		return nil, &ValidationError{Message: "organization_id is required"}
	}
	if strings.TrimSpace(code) == "" {
		return nil, &ValidationError{Message: "code is required"}
	}

	var metric models.BillableMetric
	err := s.db.WithContext(ctx).
		Where("organization_id = ? AND code = ? AND deleted_at IS NULL", organizationID, code).
		Preload("Filters", "deleted_at IS NULL").
		First(&metric).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrBillableMetricNotFound
		}
		return nil, err
	}

	return &metric, nil
}

func (s *service) GetByID(ctx context.Context, organizationID string, id string) (*models.BillableMetric, error) {
	if strings.TrimSpace(organizationID) == "" {
		return nil, &ValidationError{Message: "organization_id is required"}
	}
	if strings.TrimSpace(id) == "" {
		return nil, &ValidationError{Message: "id is required"}
	}

	var metric models.BillableMetric
	err := s.db.WithContext(ctx).
		Where("organization_id = ? AND id = ? AND deleted_at IS NULL", organizationID, id).
		Preload("Filters", "deleted_at IS NULL").
		First(&metric).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrBillableMetricNotFound
		}
		return nil, err
	}

	return &metric, nil
}

func (s *service) Update(ctx context.Context, organizationID string, code string, input UpdateInput) (*models.BillableMetric, error) {
	metric, err := s.GetByCode(ctx, organizationID, code)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		metric.Name = strings.TrimSpace(*input.Name)
	}
	if input.Description != nil {
		metric.Description = input.Description
	}
	if input.AggregationType != nil {
		aggType, ok := models.AggregationTypeFromString(*input.AggregationType)
		if !ok {
			return nil, &ValidationError{Message: "invalid aggregation_type"}
		}
		metric.AggregationType = aggType
	}
	if input.FieldName != nil {
		metric.FieldName = input.FieldName
	}
	if input.Recurring != nil {
		metric.Recurring = *input.Recurring
	}
	if input.Expression != nil {
		metric.Expression = input.Expression
	}
	if input.CustomAggregator != nil {
		metric.CustomAggregator = input.CustomAggregator
	}
	if input.WeightedInterval != nil {
		metric.WeightedInterval = input.WeightedInterval
	}
	if input.RoundingFunction != nil {
		metric.RoundingFunction = input.RoundingFunction
	}
	if input.RoundingPrecision != nil {
		metric.RoundingPrecision = input.RoundingPrecision
	}

	if err := validateMetricState(metric); err != nil {
		return nil, err
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(metric).Error; err != nil {
			return err
		}
		if input.Filters != nil {
			return syncFilters(tx, metric.ID, organizationID, *input.Filters)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return s.GetByCode(ctx, organizationID, metric.Code)
}

func (s *service) Delete(ctx context.Context, organizationID string, code string) (*models.BillableMetric, error) {
	metric, err := s.GetByCode(ctx, organizationID, code)
	if err != nil {
		return nil, err
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Soft-delete all active filters first.
		if err := tx.Exec(
			"UPDATE billable_metric_filters SET deleted_at = now() WHERE billable_metric_id = ? AND deleted_at IS NULL",
			metric.ID,
		).Error; err != nil {
			return err
		}
		return tx.Delete(metric).Error
	})
	if err != nil {
		return nil, err
	}

	return metric, nil
}

// syncFilters replaces the active filters for a billable metric.
// It soft-deletes removed filters and upserts new ones.
func syncFilters(tx *gorm.DB, metricID string, organizationID string, inputs []FilterInput) error {
	// Soft-delete all current filters.
	if err := tx.Exec(
		"UPDATE billable_metric_filters SET deleted_at = now() WHERE billable_metric_id = ? AND deleted_at IS NULL",
		metricID,
	).Error; err != nil {
		return err
	}

	if len(inputs) == 0 {
		return nil
	}

	newFilters := make([]models.BillableMetricFilter, 0, len(inputs))
	for _, fi := range inputs {
		if strings.TrimSpace(fi.Key) == "" {
			continue
		}
		vals := make(models.StringArray, 0, len(fi.Values))
		for _, v := range fi.Values {
			if strings.TrimSpace(v) != "" {
				vals = append(vals, v)
			}
		}
		newFilters = append(newFilters, models.BillableMetricFilter{
			BillableMetricID: metricID,
			OrganizationID:   organizationID,
			Key:              strings.TrimSpace(fi.Key),
			Values:           vals,
		})
	}

	if len(newFilters) == 0 {
		return nil
	}

	return tx.Create(&newFilters).Error
}

func validateCreateInput(organizationID string, input CreateInput) error {
	if strings.TrimSpace(organizationID) == "" {
		return &ValidationError{Message: "organization_id is required"}
	}
	if strings.TrimSpace(input.Name) == "" {
		return &ValidationError{Message: "name is required"}
	}
	if strings.TrimSpace(input.Code) == "" {
		return &ValidationError{Message: "code is required"}
	}
	if _, ok := models.AggregationTypeFromString(input.AggregationType); !ok {
		return &ValidationError{Message: "invalid aggregation_type"}
	}

	tempMetric := &models.BillableMetric{
		AggregationType: func() models.AggregationType {
			a, _ := models.AggregationTypeFromString(input.AggregationType)
			return a
		}(),
		FieldName:        input.FieldName,
		Recurring:        boolVal(input.Recurring),
		WeightedInterval: input.WeightedInterval,
		CustomAggregator: input.CustomAggregator,
	}
	return validateMetricState(tempMetric)
}

func validateMetricState(m *models.BillableMetric) error {
	aggStr := models.AggregationTypeToString(m.AggregationType)

	// field_name required unless count_agg or custom_agg.
	if aggStr != "count_agg" && aggStr != "custom_agg" {
		if m.FieldName == nil || strings.TrimSpace(*m.FieldName) == "" {
			return &ValidationError{Message: "field_name is required for this aggregation type"}
		}
	}

	// weighted_interval required for weighted_sum_agg.
	if aggStr == "weighted_sum_agg" {
		if m.WeightedInterval == nil || strings.TrimSpace(*m.WeightedInterval) == "" {
			return &ValidationError{Message: "weighted_interval is required for weighted_sum_agg"}
		}
	}

	// recurring incompatible with count_agg, max_agg, latest_agg.
	if m.Recurring {
		if aggStr == "count_agg" || aggStr == "max_agg" || aggStr == "latest_agg" {
			return &ValidationError{Message: "recurring is not compatible with this aggregation type"}
		}
	}

	// custom_aggregator required for custom_agg.
	if aggStr == "custom_agg" {
		if m.CustomAggregator == nil || strings.TrimSpace(*m.CustomAggregator) == "" {
			return &ValidationError{Message: "custom_aggregator is required for custom_agg"}
		}
	}

	return nil
}

func normalizePagination(page, perPage int) (int, int) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	return page, perPage
}

func boolVal(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}
