package plans

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"

	"github.com/getlago/lago/api-go/internal/models"
)

// ErrPlanNotFound is returned when no plan matches the lookup.
var ErrPlanNotFound = errors.New("plan_not_found")

// ErrPlanCodeConflict is returned when a code already exists in the org.
var ErrPlanCodeConflict = errors.New("plan_code_taken")

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

// ChargeFilterInput represents a filter on a charge.
type ChargeFilterInput struct {
	InvoiceDisplayName *string        `json:"invoice_display_name"`
	Properties         map[string]any `json:"properties"`
}

// ChargeInput holds the fields for a charge within a plan create/update.
type ChargeInput struct {
	ID                 *string             `json:"id"`
	BillableMetricID   string              `json:"billable_metric_id"`
	ChargeModel        string              `json:"charge_model"`
	Code               string              `json:"code"`
	Properties         map[string]any      `json:"properties"`
	PayInAdvance       *bool               `json:"pay_in_advance"`
	Invoiceable        *bool               `json:"invoiceable"`
	Prorated           *bool               `json:"prorated"`
	MinAmountCents     *int64              `json:"min_amount_cents"`
	InvoiceDisplayName *string             `json:"invoice_display_name"`
	Filters            []ChargeFilterInput `json:"filters"`
}

// CreateInput holds the fields needed to create a plan.
type CreateInput struct {
	Name                    string        `json:"name"`
	Code                    string        `json:"code"`
	Description             *string       `json:"description"`
	Interval                string        `json:"interval"`
	AmountCents             int64         `json:"amount_cents"`
	AmountCurrency          string        `json:"amount_currency"`
	PayInAdvance            bool          `json:"pay_in_advance"`
	BillChargesMonthly      *bool         `json:"bill_charges_monthly"`
	BillFixedChargesMonthly *bool         `json:"bill_fixed_charges_monthly"`
	TrialPeriod             *float64      `json:"trial_period"`
	InvoiceDisplayName      *string       `json:"invoice_display_name"`
	Charges                 []ChargeInput `json:"charges"`
}

// UpdateInput holds the mutable fields for updating a plan.
type UpdateInput struct {
	Name                    *string       `json:"name"`
	Description             *string       `json:"description"`
	Interval                *string       `json:"interval"`
	AmountCents             *int64        `json:"amount_cents"`
	AmountCurrency          *string       `json:"amount_currency"`
	PayInAdvance            *bool         `json:"pay_in_advance"`
	BillChargesMonthly      *bool         `json:"bill_charges_monthly"`
	BillFixedChargesMonthly *bool         `json:"bill_fixed_charges_monthly"`
	TrialPeriod             *float64      `json:"trial_period"`
	InvoiceDisplayName      *string       `json:"invoice_display_name"`
	Charges                 *[]ChargeInput `json:"charges"`
}

// ListFilter defines pagination + search parameters for listing.
type ListFilter struct {
	Page        int
	PerPage     int
	SearchTerm  string
	WithDeleted bool
}

// ListResult holds the result of a List operation.
type ListResult struct {
	Plans       []models.Plan
	TotalCount  int64
	TotalPages  int
	CurrentPage int
	NextPage    *int
	PrevPage    *int
}

// Service defines the plans business-logic contract.
type Service interface {
	Create(ctx context.Context, organizationID string, input CreateInput) (*models.Plan, error)
	List(ctx context.Context, organizationID string, filter ListFilter) (*ListResult, error)
	GetByCode(ctx context.Context, organizationID string, code string) (*models.Plan, error)
	GetByID(ctx context.Context, organizationID string, id string) (*models.Plan, error)
	Update(ctx context.Context, organizationID string, code string, input UpdateInput) (*models.Plan, error)
	Delete(ctx context.Context, organizationID string, code string) (*models.Plan, error)
}

// NewService constructs the plan service backed by the given DB.
func NewService(db *gorm.DB) Service {
	return &service{db: db}
}

type service struct {
	db *gorm.DB
}

func (s *service) Create(ctx context.Context, organizationID string, input CreateInput) (*models.Plan, error) {
	if err := validateCreate(organizationID, input); err != nil {
		return nil, err
	}

	interval, _ := models.PlanIntervalFromString(input.Interval)

	plan := &models.Plan{
		OrganizationID:          organizationID,
		Name:                    strings.TrimSpace(input.Name),
		Code:                    strings.TrimSpace(input.Code),
		Description:             input.Description,
		Interval:                interval,
		AmountCents:             input.AmountCents,
		AmountCurrency:          strings.ToUpper(input.AmountCurrency),
		PayInAdvance:            input.PayInAdvance,
		BillChargesMonthly:      input.BillChargesMonthly,
		BillFixedChargesMonthly: boolVal(input.BillFixedChargesMonthly),
		TrialPeriod:             input.TrialPeriod,
		InvoiceDisplayName:      input.InvoiceDisplayName,
	}

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := checkCodeUnique(tx, organizationID, "", plan.Code); err != nil {
			return err
		}
		if err := tx.Create(plan).Error; err != nil {
			return err
		}
		return syncCharges(tx, organizationID, plan.ID, input.Charges)
	})
	if err != nil {
		return nil, err
	}

	return s.loadPlan(ctx, plan.ID)
}

func (s *service) List(ctx context.Context, organizationID string, filter ListFilter) (*ListResult, error) {
	page, perPage := normalizePagination(filter.Page, filter.PerPage)

	q := s.db.WithContext(ctx).Model(&models.Plan{}).
		Where("organization_id = ? AND parent_id IS NULL", organizationID)

	if !filter.WithDeleted {
		q = q.Where("deleted_at IS NULL")
	}

	if filter.SearchTerm != "" {
		like := "%" + filter.SearchTerm + "%"
		q = q.Where("name ILIKE ? OR code ILIKE ?", like, like)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, err
	}

	var plans []models.Plan
	offset := (page - 1) * perPage
	if err := q.Preload("Charges.Filters").Order("created_at DESC").
		Limit(perPage).Offset(offset).Find(&plans).Error; err != nil {
		return nil, err
	}

	totalPages := int((total + int64(perPage) - 1) / int64(perPage))
	result := &ListResult{
		Plans:       plans,
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

func (s *service) GetByCode(ctx context.Context, organizationID, code string) (*models.Plan, error) {
	var plan models.Plan
	err := s.db.WithContext(ctx).
		Where("organization_id = ? AND code = ? AND deleted_at IS NULL AND parent_id IS NULL", organizationID, code).
		Preload("Charges.Filters").
		First(&plan).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrPlanNotFound
	}
	return &plan, err
}

func (s *service) GetByID(ctx context.Context, organizationID, id string) (*models.Plan, error) {
	var plan models.Plan
	err := s.db.WithContext(ctx).
		Where("organization_id = ? AND id = ? AND deleted_at IS NULL", organizationID, id).
		Preload("Charges.Filters").
		First(&plan).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrPlanNotFound
	}
	return &plan, err
}

func (s *service) Update(ctx context.Context, organizationID, code string, input UpdateInput) (*models.Plan, error) {
	plan, err := s.GetByCode(ctx, organizationID, code)
	if err != nil {
		return nil, err
	}

	updates := map[string]any{}
	if input.Name != nil {
		updates["name"] = strings.TrimSpace(*input.Name)
	}
	if input.Description != nil {
		updates["description"] = *input.Description
	}
	if input.Interval != nil {
		iv, ok := models.PlanIntervalFromString(*input.Interval)
		if !ok {
			return nil, &ValidationError{Message: "invalid interval"}
		}
		updates["interval"] = iv
	}
	if input.AmountCents != nil {
		updates["amount_cents"] = *input.AmountCents
	}
	if input.AmountCurrency != nil {
		updates["amount_currency"] = strings.ToUpper(*input.AmountCurrency)
	}
	if input.PayInAdvance != nil {
		updates["pay_in_advance"] = *input.PayInAdvance
	}
	if input.BillChargesMonthly != nil {
		updates["bill_charges_monthly"] = *input.BillChargesMonthly
	}
	if input.BillFixedChargesMonthly != nil {
		updates["bill_fixed_charges_monthly"] = *input.BillFixedChargesMonthly
	}
	if input.TrialPeriod != nil {
		updates["trial_period"] = *input.TrialPeriod
	}
	if input.InvoiceDisplayName != nil {
		updates["invoice_display_name"] = *input.InvoiceDisplayName
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if len(updates) > 0 {
			if err := tx.Model(plan).Updates(updates).Error; err != nil {
				return err
			}
		}
		if input.Charges != nil {
			return syncCharges(tx, organizationID, plan.ID, *input.Charges)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return s.loadPlan(ctx, plan.ID)
}

func (s *service) Delete(ctx context.Context, organizationID, code string) (*models.Plan, error) {
	plan, err := s.GetByCode(ctx, organizationID, code)
	if err != nil {
		return nil, err
	}

	if err := s.db.WithContext(ctx).Delete(plan).Error; err != nil {
		return nil, err
	}

	return plan, nil
}

// loadPlan reloads a plan with its associations.
func (s *service) loadPlan(ctx context.Context, id string) (*models.Plan, error) {
	var plan models.Plan
	err := s.db.WithContext(ctx).
		Where("id = ?", id).
		Preload("Charges.Filters").
		First(&plan).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrPlanNotFound
	}
	return &plan, err
}

// syncCharges replaces all non-deleted charges on the plan with the given inputs.
// Existing charges with matching IDs are updated; new ones are created; removed ones are soft-deleted.
func syncCharges(tx *gorm.DB, organizationID, planID string, inputs []ChargeInput) error {
	// Soft-delete all existing charges for the plan.
	if err := tx.Where("plan_id = ? AND deleted_at IS NULL", planID).
		Delete(&models.Charge{}).Error; err != nil {
		return err
	}

	for _, ci := range inputs {
		chargeModel, ok := models.ChargeModelFromString(ci.ChargeModel)
		if !ok {
			return &ValidationError{Message: "invalid charge_model: " + ci.ChargeModel}
		}
		props := models.JSONBMap(ci.Properties)
		if props == nil {
			props = models.JSONBMap{}
		}
		charge := models.Charge{
			OrganizationID:     organizationID,
			PlanID:             planID,
			BillableMetricID:   &ci.BillableMetricID,
			ChargeModel:        chargeModel,
			Code:               strings.TrimSpace(ci.Code),
			Properties:         props,
			PayInAdvance:       boolVal(ci.PayInAdvance),
			Invoiceable:        boolValDefault(ci.Invoiceable, true),
			Prorated:           boolVal(ci.Prorated),
			MinAmountCents:     int64Val(ci.MinAmountCents),
			InvoiceDisplayName: ci.InvoiceDisplayName,
		}
		if err := tx.Create(&charge).Error; err != nil {
			return err
		}
		for _, fi := range ci.Filters {
			filterProps := models.JSONBMap(fi.Properties)
			if filterProps == nil {
				filterProps = models.JSONBMap{}
			}
			cf := models.ChargeFilter{
				ChargeID:           charge.ID,
				OrganizationID:     organizationID,
				InvoiceDisplayName: fi.InvoiceDisplayName,
				Properties:         filterProps,
			}
			if err := tx.Create(&cf).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

// checkCodeUnique returns ErrPlanCodeConflict if another plan with the same code exists.
func checkCodeUnique(tx *gorm.DB, organizationID, excludeID, code string) error {
	q := tx.Model(&models.Plan{}).
		Where("organization_id = ? AND code = ? AND deleted_at IS NULL AND parent_id IS NULL", organizationID, code)
	if excludeID != "" {
		q = q.Where("id != ?", excludeID)
	}
	var count int64
	if err := q.Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return ErrPlanCodeConflict
	}
	return nil
}

func validateCreate(organizationID string, input CreateInput) error {
	if strings.TrimSpace(organizationID) == "" {
		return &ValidationError{Message: "organization_id is required"}
	}
	if strings.TrimSpace(input.Name) == "" {
		return &ValidationError{Message: "name is required"}
	}
	if strings.TrimSpace(input.Code) == "" {
		return &ValidationError{Message: "code is required"}
	}
	if _, ok := models.PlanIntervalFromString(input.Interval); !ok {
		return &ValidationError{Message: "invalid interval"}
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

func boolValDefault(b *bool, def bool) bool {
	if b == nil {
		return def
	}
	return *b
}

func int64Val(v *int64) int64 {
	if v == nil {
		return 0
	}
	return *v
}
