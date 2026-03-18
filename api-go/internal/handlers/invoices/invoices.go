package invoices

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/getlago/lago/api-go/internal/middleware"
	"github.com/getlago/lago/api-go/internal/models"
	invoiceservices "github.com/getlago/lago/api-go/internal/services/invoices"
)

type createInvoiceRequestEnvelope struct {
	Invoice invoiceservices.CreateInvoiceInput `json:"invoice" binding:"required"`
}

type invoiceResponse struct {
	LagoID                      string  `json:"lago_id"`
	SequentialID                *int    `json:"sequential_id"`
	Number                      string  `json:"number"`
	Status                      string  `json:"status"`
	PaymentStatus               string  `json:"payment_status"`
	InvoiceType                 string  `json:"invoice_type"`
	Currency                    string  `json:"currency"`
	TotalAmountCents            int64   `json:"total_amount_cents"`
	FeesAmountCents             int64   `json:"fees_amount_cents"`
	TaxesAmountCents            int64   `json:"taxes_amount_cents"`
	CouponsAmountCents          int64   `json:"coupons_amount_cents"`
	SubTotalExcludingTaxesCents int64   `json:"sub_total_excluding_taxes_amount_cents"`
	SubTotalIncludingTaxesCents int64   `json:"sub_total_including_taxes_amount_cents"`
	IssuingDate                 *string `json:"issuing_date"`
	PaymentDueDate              *string `json:"payment_due_date"`
	FinalizedAt                 *string `json:"finalized_at"`
	VoidedAt                    *string `json:"voided_at"`
	CreatedAt                   string  `json:"created_at"`
	UpdatedAt                   string  `json:"updated_at"`
}

func Create(svc invoiceservices.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		organizationID, ok := organizationIDFromContext(c)
		if !ok {
			return
		}

		var req createInvoiceRequestEnvelope
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": "error", "error_code": "validation_error", "error_details": gin.H{"message": err.Error()}})
			return
		}

		invoice, err := svc.Create(c.Request.Context(), organizationID, req.Invoice)
		if err != nil {
			handleServiceError(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"invoice": toInvoiceResponse(invoice)})
	}
}

func Index(svc invoiceservices.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		organizationID, ok := organizationIDFromContext(c)
		if !ok {
			return
		}

		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))

		filter := invoiceservices.ListInvoicesFilter{
			CustomerExternalID: c.Query("external_customer_id"),
			Status:             c.Query("status"),
			Page:               page,
			PerPage:            perPage,
		}

		invoices, pagination, err := svc.List(c.Request.Context(), organizationID, filter)
		if err != nil {
			handleServiceError(c, err)
			return
		}

		response := make([]invoiceResponse, 0, len(invoices))
		for i := range invoices {
			response = append(response, toInvoiceResponse(&invoices[i]))
		}

		c.JSON(http.StatusOK, gin.H{
			"invoices": response,
			"meta": gin.H{
				"current_page": pagination.CurrentPage,
				"next_page":    pagination.NextPage,
				"prev_page":    pagination.PrevPage,
				"total_pages":  pagination.TotalPages,
				"total_count":  pagination.TotalCount,
			},
		})
	}
}

func Show(svc invoiceservices.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		organizationID, ok := organizationIDFromContext(c)
		if !ok {
			return
		}

		id := c.Param("id")
		invoice, err := svc.GetByID(c.Request.Context(), organizationID, id)
		if err != nil {
			handleServiceError(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"invoice": toInvoiceResponse(invoice)})
	}
}

func Finalize(svc invoiceservices.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		organizationID, ok := organizationIDFromContext(c)
		if !ok {
			return
		}

		id := c.Param("id")
		invoice, err := svc.Finalize(c.Request.Context(), organizationID, id)
		if err != nil {
			handleServiceError(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"invoice": toInvoiceResponse(invoice)})
	}
}

func Void(svc invoiceservices.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		organizationID, ok := organizationIDFromContext(c)
		if !ok {
			return
		}

		id := c.Param("id")
		invoice, err := svc.Void(c.Request.Context(), organizationID, id)
		if err != nil {
			handleServiceError(c, err)
			return
		}

		c.JSON(http.StatusOK, gin.H{"invoice": toInvoiceResponse(invoice)})
	}
}

func organizationIDFromContext(c *gin.Context) (string, bool) {
	value, exists := c.Get(middleware.GinKeyOrganizationID)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "unauthorized", "error": "missing_organization_context"})
		return "", false
	}

	organizationID, ok := value.(string)
	if !ok || organizationID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "unauthorized", "error": "invalid_organization_context"})
		return "", false
	}

	return organizationID, true
}

func handleServiceError(c *gin.Context, err error) {
	switch {
	case invoiceservices.IsNotFoundError(err):
		c.JSON(http.StatusNotFound, gin.H{"status": "error", "error_code": "invoice_not_found", "error_details": gin.H{}})
	case invoiceservices.IsTransitionError(err):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"status": "error", "error_code": "transition_error", "error_details": gin.H{"message": err.Error()}})
	case invoiceservices.IsValidationError(err):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"status": "error", "error_code": "validation_error", "error_details": gin.H{"message": err.Error()}})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "error_code": "internal_error", "error_details": gin.H{}})
	}
}

func toInvoiceResponse(invoice *models.Invoice) invoiceResponse {
	resp := invoiceResponse{
		LagoID:                      invoice.ID,
		SequentialID:                invoice.SequentialID,
		Number:                      invoice.Number,
		Status:                      invoiceStatusToString(invoice.Status),
		PaymentStatus:               invoicePaymentStatusToString(invoice.PaymentStatus),
		InvoiceType:                 invoiceTypeToString(invoice.InvoiceType),
		Currency:                    invoice.Currency,
		TotalAmountCents:            invoice.TotalAmountCents,
		FeesAmountCents:             invoice.FeesAmountCents,
		TaxesAmountCents:            invoice.TaxesAmountCents,
		CouponsAmountCents:          invoice.CouponsAmountCents,
		SubTotalExcludingTaxesCents: invoice.SubTotalExcludingTaxesCents,
		SubTotalIncludingTaxesCents: invoice.SubTotalIncludingTaxesCents,
		CreatedAt:                   invoice.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:                   invoice.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}

	if invoice.IssuingDate != nil {
		s := invoice.IssuingDate.Format("2006-01-02")
		resp.IssuingDate = &s
	}
	if invoice.PaymentDueDate != nil {
		s := invoice.PaymentDueDate.Format("2006-01-02")
		resp.PaymentDueDate = &s
	}
	if invoice.FinalizedAt != nil {
		s := invoice.FinalizedAt.Format("2006-01-02T15:04:05Z")
		resp.FinalizedAt = &s
	}
	if invoice.VoidedAt != nil {
		s := invoice.VoidedAt.Format("2006-01-02T15:04:05Z")
		resp.VoidedAt = &s
	}

	return resp
}

func invoiceStatusToString(status models.InvoiceStatus) string {
	switch status {
	case models.InvoiceStatusDraft:
		return "draft"
	case models.InvoiceStatusFinalized:
		return "finalized"
	case models.InvoiceStatusVoided:
		return "voided"
	case models.InvoiceStatusGenerating:
		return "generating"
	case models.InvoiceStatusFailed:
		return "failed"
	default:
		return "unknown"
	}
}

func invoicePaymentStatusToString(status models.InvoicePaymentStatus) string {
	switch status {
	case models.InvoicePaymentStatusPending:
		return "pending"
	case models.InvoicePaymentStatusSucceeded:
		return "succeeded"
	case models.InvoicePaymentStatusFailed:
		return "failed"
	default:
		return "unknown"
	}
}

func invoiceTypeToString(t models.InvoiceType) string {
	switch t {
	case models.InvoiceTypeSubscription:
		return "subscription"
	case models.InvoiceTypeAddOn:
		return "add_on"
	case models.InvoiceTypeCredit:
		return "credit"
	case models.InvoiceTypeOneOff:
		return "one_off"
	case models.InvoiceTypeAdvanceCharges:
		return "advance_charges"
	case models.InvoiceTypeProgressiveBilling:
		return "progressive_billing"
	default:
		return "unknown"
	}
}
