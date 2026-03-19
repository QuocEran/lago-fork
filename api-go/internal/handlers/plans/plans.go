package plans

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/getlago/lago/api-go/internal/middleware"
	"github.com/getlago/lago/api-go/internal/models"
	plansvc "github.com/getlago/lago/api-go/internal/services/plans"
)

// ── request types ─────────────────────────────────────────────────────────────

type chargeFilterRequest struct {
	InvoiceDisplayName *string        `json:"invoice_display_name"`
	Properties         map[string]any `json:"properties"`
}

type chargeRequest struct {
	ID                 *string               `json:"id"`
	BillableMetricID   string                `json:"billable_metric_id" binding:"required"`
	ChargeModel        string                `json:"charge_model"        binding:"required"`
	Code               string                `json:"code"               binding:"required"`
	Properties         map[string]any        `json:"properties"`
	PayInAdvance       *bool                 `json:"pay_in_advance"`
	Invoiceable        *bool                 `json:"invoiceable"`
	Prorated           *bool                 `json:"prorated"`
	MinAmountCents     *int64                `json:"min_amount_cents"`
	InvoiceDisplayName *string               `json:"invoice_display_name"`
	Filters            []chargeFilterRequest `json:"filters"`
}

type createPlanRequest struct {
	Name                    string          `json:"name"            binding:"required"`
	Code                    string          `json:"code"            binding:"required"`
	Description             *string         `json:"description"`
	Interval                string          `json:"interval"        binding:"required"`
	AmountCents             int64           `json:"amount_cents"    binding:"required"`
	AmountCurrency          string          `json:"amount_currency" binding:"required"`
	PayInAdvance            bool            `json:"pay_in_advance"`
	BillChargesMonthly      *bool           `json:"bill_charges_monthly"`
	BillFixedChargesMonthly *bool           `json:"bill_fixed_charges_monthly"`
	TrialPeriod             *float64        `json:"trial_period"`
	InvoiceDisplayName      *string         `json:"invoice_display_name"`
	Charges                 []chargeRequest `json:"charges"`
}

type createEnvelope struct {
	Plan createPlanRequest `json:"plan" binding:"required"`
}

type updatePlanRequest struct {
	Name                    *string         `json:"name"`
	Description             *string         `json:"description"`
	Interval                *string         `json:"interval"`
	AmountCents             *int64          `json:"amount_cents"`
	AmountCurrency          *string         `json:"amount_currency"`
	PayInAdvance            *bool           `json:"pay_in_advance"`
	BillChargesMonthly      *bool           `json:"bill_charges_monthly"`
	BillFixedChargesMonthly *bool           `json:"bill_fixed_charges_monthly"`
	TrialPeriod             *float64        `json:"trial_period"`
	InvoiceDisplayName      *string         `json:"invoice_display_name"`
	Charges                 *[]chargeRequest `json:"charges"`
}

type updateEnvelope struct {
	Plan updatePlanRequest `json:"plan" binding:"required"`
}

// ── response types ─────────────────────────────────────────────────────────────

type chargeFilterResponse struct {
	LagoID             string         `json:"lago_id"`
	InvoiceDisplayName *string        `json:"invoice_display_name"`
	Properties         map[string]any `json:"properties"`
}

type chargeResponse struct {
	LagoID             string                 `json:"lago_id"`
	BillableMetricID   *string                `json:"billable_metric_id"`
	ChargeModel        string                 `json:"charge_model"`
	Code               string                 `json:"code"`
	Properties         map[string]any         `json:"properties"`
	PayInAdvance       bool                   `json:"pay_in_advance"`
	Invoiceable        bool                   `json:"invoiceable"`
	Prorated           bool                   `json:"prorated"`
	MinAmountCents     int64                  `json:"min_amount_cents"`
	InvoiceDisplayName *string                `json:"invoice_display_name"`
	Filters            []chargeFilterResponse `json:"filters"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
}

type planResponse struct {
	LagoID                  string           `json:"lago_id"`
	Name                    string           `json:"name"`
	Code                    string           `json:"code"`
	Description             *string          `json:"description"`
	Interval                string           `json:"interval"`
	AmountCents             int64            `json:"amount_cents"`
	AmountCurrency          string           `json:"amount_currency"`
	PayInAdvance            bool             `json:"pay_in_advance"`
	BillChargesMonthly      *bool            `json:"bill_charges_monthly"`
	BillFixedChargesMonthly bool             `json:"bill_fixed_charges_monthly"`
	TrialPeriod             *float64         `json:"trial_period"`
	InvoiceDisplayName      *string          `json:"invoice_display_name"`
	Charges                 []chargeResponse `json:"charges"`
	CreatedAt               time.Time        `json:"created_at"`
	UpdatedAt               time.Time        `json:"updated_at"`
}

type paginationMeta struct {
	CurrentPage int   `json:"current_page"`
	NextPage    *int  `json:"next_page"`
	PrevPage    *int  `json:"prev_page"`
	TotalPages  int   `json:"total_pages"`
	TotalCount  int64 `json:"total_count"`
}

// ── handlers ──────────────────────────────────────────────────────────────────

// Create handles POST /api/v1/plans.
func Create(svc plansvc.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		orgID, ok := organizationIDFromContext(c)
		if !ok {
			return
		}
		var req createEnvelope
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "error_code": "validation_error", "error_details": gin.H{"message": err.Error()}})
			return
		}
		input := toCreateInput(req.Plan)
		plan, err := svc.Create(c.Request.Context(), orgID, input)
		if err != nil {
			handleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"plan": toPlanResponse(plan)})
	}
}

// Index handles GET /api/v1/plans.
func Index(svc plansvc.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		orgID, ok := organizationIDFromContext(c)
		if !ok {
			return
		}
		filter := parseListFilter(c)
		result, err := svc.List(c.Request.Context(), orgID, filter)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "error_code": "internal_error", "error_details": gin.H{}})
			return
		}
		items := make([]planResponse, 0, len(result.Plans))
		for i := range result.Plans {
			items = append(items, toPlanResponse(&result.Plans[i]))
		}
		c.JSON(http.StatusOK, gin.H{
			"plans": items,
			"meta": paginationMeta{
				CurrentPage: result.CurrentPage,
				NextPage:    result.NextPage,
				PrevPage:    result.PrevPage,
				TotalPages:  result.TotalPages,
				TotalCount:  result.TotalCount,
			},
		})
	}
}

// Show handles GET /api/v1/plans/:code.
func Show(svc plansvc.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		orgID, ok := organizationIDFromContext(c)
		if !ok {
			return
		}
		code := c.Param("code")
		plan, err := svc.GetByCode(c.Request.Context(), orgID, code)
		if err != nil {
			handleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"plan": toPlanResponse(plan)})
	}
}

// Update handles PUT /api/v1/plans/:code.
func Update(svc plansvc.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		orgID, ok := organizationIDFromContext(c)
		if !ok {
			return
		}
		var req updateEnvelope
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "error_code": "validation_error", "error_details": gin.H{"message": err.Error()}})
			return
		}
		code := c.Param("code")
		input := toUpdateInput(req.Plan)
		plan, err := svc.Update(c.Request.Context(), orgID, code, input)
		if err != nil {
			handleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"plan": toPlanResponse(plan)})
	}
}

// Destroy handles DELETE /api/v1/plans/:code.
func Destroy(svc plansvc.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		orgID, ok := organizationIDFromContext(c)
		if !ok {
			return
		}
		code := c.Param("code")
		plan, err := svc.Delete(c.Request.Context(), orgID, code)
		if err != nil {
			handleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"plan": toPlanResponse(plan)})
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func organizationIDFromContext(c *gin.Context) (string, bool) {
	value, exists := c.Get(middleware.GinKeyOrganizationID)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "unauthorized", "error": "missing_organization_context"})
		return "", false
	}
	orgID, ok := value.(string)
	if !ok || orgID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "unauthorized", "error": "invalid_organization_context"})
		return "", false
	}
	return orgID, true
}

func handleServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, plansvc.ErrPlanNotFound):
		c.JSON(http.StatusNotFound, gin.H{"status": "error", "error_code": "plan_not_found", "error_details": gin.H{}})
	case errors.Is(err, plansvc.ErrPlanCodeConflict):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"status": "error", "error_code": "value_already_exist", "error_details": gin.H{"code": []string{"value_already_exist"}}})
	case plansvc.IsValidationError(err):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"status": "error", "error_code": "validation_error", "error_details": gin.H{"message": err.Error()}})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "error_code": "internal_error", "error_details": gin.H{}})
	}
}

func toCreateInput(req createPlanRequest) plansvc.CreateInput {
	return plansvc.CreateInput{
		Name:                    req.Name,
		Code:                    req.Code,
		Description:             req.Description,
		Interval:                req.Interval,
		AmountCents:             req.AmountCents,
		AmountCurrency:          req.AmountCurrency,
		PayInAdvance:            req.PayInAdvance,
		BillChargesMonthly:      req.BillChargesMonthly,
		BillFixedChargesMonthly: req.BillFixedChargesMonthly,
		TrialPeriod:             req.TrialPeriod,
		InvoiceDisplayName:      req.InvoiceDisplayName,
		Charges:                 toChargeInputs(req.Charges),
	}
}

func toUpdateInput(req updatePlanRequest) plansvc.UpdateInput {
	var chargeInputs *[]plansvc.ChargeInput
	if req.Charges != nil {
		ci := toChargeInputs(*req.Charges)
		chargeInputs = &ci
	}
	return plansvc.UpdateInput{
		Name:                    req.Name,
		Description:             req.Description,
		Interval:                req.Interval,
		AmountCents:             req.AmountCents,
		AmountCurrency:          req.AmountCurrency,
		PayInAdvance:            req.PayInAdvance,
		BillChargesMonthly:      req.BillChargesMonthly,
		BillFixedChargesMonthly: req.BillFixedChargesMonthly,
		TrialPeriod:             req.TrialPeriod,
		InvoiceDisplayName:      req.InvoiceDisplayName,
		Charges:                 chargeInputs,
	}
}

func toChargeInputs(reqs []chargeRequest) []plansvc.ChargeInput {
	out := make([]plansvc.ChargeInput, 0, len(reqs))
	for _, r := range reqs {
		filters := make([]plansvc.ChargeFilterInput, 0, len(r.Filters))
		for _, f := range r.Filters {
			filters = append(filters, plansvc.ChargeFilterInput{
				InvoiceDisplayName: f.InvoiceDisplayName,
				Properties:         f.Properties,
			})
		}
		out = append(out, plansvc.ChargeInput{
			ID:                 r.ID,
			BillableMetricID:   r.BillableMetricID,
			ChargeModel:        r.ChargeModel,
			Code:               r.Code,
			Properties:         r.Properties,
			PayInAdvance:       r.PayInAdvance,
			Invoiceable:        r.Invoiceable,
			Prorated:           r.Prorated,
			MinAmountCents:     r.MinAmountCents,
			InvoiceDisplayName: r.InvoiceDisplayName,
			Filters:            filters,
		})
	}
	return out
}

func toPlanResponse(p *models.Plan) planResponse {
	charges := make([]chargeResponse, 0, len(p.Charges))
	for i := range p.Charges {
		charges = append(charges, toChargeResponse(&p.Charges[i]))
	}
	return planResponse{
		LagoID:                  p.ID,
		Name:                    p.Name,
		Code:                    p.Code,
		Description:             p.Description,
		Interval:                models.PlanIntervalToString(p.Interval),
		AmountCents:             p.AmountCents,
		AmountCurrency:          p.AmountCurrency,
		PayInAdvance:            p.PayInAdvance,
		BillChargesMonthly:      p.BillChargesMonthly,
		BillFixedChargesMonthly: p.BillFixedChargesMonthly,
		TrialPeriod:             p.TrialPeriod,
		InvoiceDisplayName:      p.InvoiceDisplayName,
		Charges:                 charges,
		CreatedAt:               p.CreatedAt,
		UpdatedAt:               p.UpdatedAt,
	}
}

func toChargeResponse(c *models.Charge) chargeResponse {
	filters := make([]chargeFilterResponse, 0, len(c.Filters))
	for i := range c.Filters {
		f := c.Filters[i]
		filters = append(filters, chargeFilterResponse{
			LagoID:             f.ID,
			InvoiceDisplayName: f.InvoiceDisplayName,
			Properties:         map[string]any(f.Properties),
		})
	}
	props := map[string]any(c.Properties)
	if props == nil {
		props = map[string]any{}
	}
	return chargeResponse{
		LagoID:             c.ID,
		BillableMetricID:   c.BillableMetricID,
		ChargeModel:        models.ChargeModelToString(c.ChargeModel),
		Code:               c.Code,
		Properties:         props,
		PayInAdvance:       c.PayInAdvance,
		Invoiceable:        c.Invoiceable,
		Prorated:           c.Prorated,
		MinAmountCents:     c.MinAmountCents,
		InvoiceDisplayName: c.InvoiceDisplayName,
		Filters:            filters,
		CreatedAt:          c.CreatedAt,
		UpdatedAt:          c.UpdatedAt,
	}
}

func parseListFilter(c *gin.Context) plansvc.ListFilter {
	f := plansvc.ListFilter{
		SearchTerm:  c.Query("search_term"),
		WithDeleted: c.Query("with_deleted") == "true",
	}
	if v := c.Query("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			f.Page = n
		}
	}
	if v := c.Query("per_page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			f.PerPage = n
		}
	}
	return f
}
