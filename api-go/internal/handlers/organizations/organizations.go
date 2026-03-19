package organizations

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/getlago/lago/api-go/internal/handlers/shared"
	"github.com/getlago/lago/api-go/internal/models"
	organizationservices "github.com/getlago/lago/api-go/internal/services/organizations"
)

var organizationErrorClassifier = shared.ServiceErrorClassifier{
	NotFoundErrors:  []error{organizationservices.ErrOrganizationNotFound},
	IsValidationErr: organizationservices.IsValidationError,
	NotFoundCode:    "organization_not_found",
}

type updateOrganizationRequest struct {
	Organization organizationservices.UpdateOrganizationInput `json:"organization" binding:"required"`
}

type billingConfigurationResponse struct {
	InvoiceFooter      *string `json:"invoice_footer"`
	InvoiceGracePeriod int     `json:"invoice_grace_period"`
	DocumentLocale     string  `json:"document_locale"`
}

type organizationResponse struct {
	LagoID                    string                       `json:"lago_id"`
	Name                      string                       `json:"name"`
	DefaultCurrency           string                       `json:"default_currency"`
	WebhookURL                string                       `json:"webhook_url"`
	WebhookURLs               []string                     `json:"webhook_urls"`
	Country                   *string                      `json:"country"`
	AddressLine1              *string                      `json:"address_line1"`
	AddressLine2              *string                      `json:"address_line2"`
	State                     *string                      `json:"state"`
	Zipcode                   *string                      `json:"zipcode"`
	Email                     *string                      `json:"email"`
	City                      *string                      `json:"city"`
	LegalName                 *string                      `json:"legal_name"`
	LegalNumber               *string                      `json:"legal_number"`
	Timezone                  string                       `json:"timezone"`
	NetPaymentTerm            int                          `json:"net_payment_term"`
	EmailSettings             []string                     `json:"email_settings"`
	DocumentNumbering         int                          `json:"document_numbering"`
	DocumentNumberPrefix      *string                      `json:"document_number_prefix"`
	TaxIdentificationNumber   *string                      `json:"tax_identification_number"`
	FinalizeZeroAmountInvoice bool                         `json:"finalize_zero_amount_invoice"`
	BillingConfiguration      billingConfigurationResponse `json:"billing_configuration"`
}

type showOrganizationEnvelope struct {
	Organization organizationResponse `json:"organization"`
}

func Show(svc organizationservices.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		organizationIDValue, ok := shared.OrganizationIDFromContext(c)
		if !ok {
			return
		}

		organization, err := svc.Get(c.Request.Context(), organizationIDValue)
		if err != nil {
			shared.HandleServiceError(c, err, organizationErrorClassifier)
			return
		}

		c.JSON(http.StatusOK, showOrganizationEnvelope{Organization: toOrganizationResponse(organization)})
	}
}

func Update(svc organizationservices.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		organizationIDValue, ok := shared.OrganizationIDFromContext(c)
		if !ok {
			return
		}

		var req updateOrganizationRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			shared.RespondError(c, http.StatusBadRequest, "validation_error", gin.H{"message": err.Error()})
			return
		}

		organization, err := svc.Update(c.Request.Context(), organizationIDValue, req.Organization)
		if err != nil {
			shared.HandleServiceError(c, err, organizationErrorClassifier)
			return
		}

		c.JSON(http.StatusOK, showOrganizationEnvelope{Organization: toOrganizationResponse(organization)})
	}
}

func toOrganizationResponse(organization *models.Organization) organizationResponse {
	webhookURLs := []string{}
	if organization.WebhookURL != nil && *organization.WebhookURL != "" {
		webhookURLs = append(webhookURLs, *organization.WebhookURL)
	}

	webhookURL := ""
	if organization.WebhookURL != nil {
		webhookURL = *organization.WebhookURL
	}

	return organizationResponse{
		LagoID:                    organization.ID,
		Name:                      organization.Name,
		DefaultCurrency:           organization.DefaultCurrency,
		WebhookURL:                webhookURL,
		WebhookURLs:               webhookURLs,
		Country:                   organization.Country,
		AddressLine1:              organization.AddressLine1,
		AddressLine2:              organization.AddressLine2,
		State:                     organization.State,
		Zipcode:                   organization.Zipcode,
		Email:                     organization.Email,
		City:                      organization.City,
		LegalName:                 organization.LegalName,
		LegalNumber:               organization.LegalNumber,
		Timezone:                  organization.Timezone,
		NetPaymentTerm:            organization.NetPaymentTerm,
		EmailSettings:             []string(organization.EmailSettings),
		DocumentNumbering:         organization.DocumentNumbering,
		DocumentNumberPrefix:      organization.DocumentNumberPrefix,
		TaxIdentificationNumber:   organization.TaxIdentificationNumber,
		FinalizeZeroAmountInvoice: organization.FinalizeZeroAmountInvoice,
		BillingConfiguration: billingConfigurationResponse{
			InvoiceFooter:      organization.InvoiceFooter,
			InvoiceGracePeriod: organization.InvoiceGracePeriod,
			DocumentLocale:     organization.DocumentLocale,
		},
	}
}
