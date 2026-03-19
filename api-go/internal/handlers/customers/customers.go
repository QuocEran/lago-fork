package customers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/getlago/lago/api-go/internal/handlers/shared"
	"github.com/getlago/lago/api-go/internal/models"
	customerservices "github.com/getlago/lago/api-go/internal/services/customers"
)

var customerErrorClassifier = shared.ServiceErrorClassifier{
	NotFoundErrors:  []error{customerservices.ErrCustomerNotFound},
	IsValidationErr: customerservices.IsValidationError,
	NotFoundCode:    "customer_not_found",
}

type createCustomerRequestEnvelope struct {
	Customer customerservices.CreateCustomerInput `json:"customer" binding:"required"`
}

type customerMetadataResponse struct {
	Key              string `json:"key"`
	Value            string `json:"value"`
	DisplayInInvoice bool   `json:"display_in_invoice"`
}

type customerResponse struct {
	LagoID     string                     `json:"lago_id"`
	ExternalID string                     `json:"external_id"`
	Name       *string                    `json:"name"`
	Email      *string                    `json:"email"`
	Currency   *string                    `json:"currency"`
	Timezone   *string                    `json:"timezone"`
	Metadata   []customerMetadataResponse `json:"metadata"`
}

func Create(svc customerservices.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		organizationID, ok := shared.OrganizationIDFromContext(c)
		if !ok {
			return
		}

		var req createCustomerRequestEnvelope
		if err := c.ShouldBindJSON(&req); err != nil {
			shared.RespondError(c, http.StatusBadRequest, "validation_error", gin.H{"message": err.Error()})
			return
		}

		customer, err := svc.Create(c.Request.Context(), organizationID, req.Customer)
		if err != nil {
			shared.HandleServiceError(c, err, customerErrorClassifier)
			return
		}

		shared.RespondJSON(c, http.StatusCreated, "customer", toCustomerResponse(customer))
	}
}

func Index(svc customerservices.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		organizationID, ok := shared.OrganizationIDFromContext(c)
		if !ok {
			return
		}

		customers, err := svc.List(c.Request.Context(), organizationID)
		if err != nil {
			shared.HandleServiceError(c, err, customerErrorClassifier)
			return
		}

		response := make([]customerResponse, 0, len(customers))
		for i := range customers {
			customer := customers[i]
			response = append(response, toCustomerResponse(&customer))
		}

		c.JSON(http.StatusOK, gin.H{
			"customers": response,
			"meta":      gin.H{"total_count": len(response)},
		})
	}
}

func Show(svc customerservices.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		organizationID, ok := shared.OrganizationIDFromContext(c)
		if !ok {
			return
		}

		externalID := c.Param("external_id")
		customer, err := svc.GetByExternalID(c.Request.Context(), organizationID, externalID)
		if err != nil {
			shared.HandleServiceError(c, err, customerErrorClassifier)
			return
		}

		c.JSON(http.StatusOK, gin.H{"customer": toCustomerResponse(customer)})
	}
}

func Delete(svc customerservices.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		organizationID, ok := shared.OrganizationIDFromContext(c)
		if !ok {
			return
		}

		externalID := c.Param("external_id")
		customer, err := svc.DeleteByExternalID(c.Request.Context(), organizationID, externalID)
		if err != nil {
			shared.HandleServiceError(c, err, customerErrorClassifier)
			return
		}

		c.JSON(http.StatusOK, gin.H{"customer": toCustomerResponse(customer)})
	}
}

func PortalURL(svc customerservices.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		organizationID, ok := shared.OrganizationIDFromContext(c)
		if !ok {
			return
		}

		externalID := c.Param("external_id")
		portalURL, err := svc.GeneratePortalURL(c.Request.Context(), organizationID, externalID)
		if err != nil {
			shared.HandleServiceError(c, err, customerErrorClassifier)
			return
		}

		c.JSON(http.StatusOK, gin.H{"customer": gin.H{"portal_url": portalURL}})
	}
}

func toCustomerResponse(customer *models.Customer) customerResponse {
	metadata := make([]customerMetadataResponse, 0, len(customer.Metadata))
	for _, metadataItem := range customer.Metadata {
		metadata = append(metadata, customerMetadataResponse{
			Key:              metadataItem.Key,
			Value:            metadataItem.Value,
			DisplayInInvoice: metadataItem.DisplayInInvoice,
		})
	}

	return customerResponse{
		LagoID:     customer.ID,
		ExternalID: customer.ExternalID,
		Name:       customer.Name,
		Email:      customer.Email,
		Currency:   customer.Currency,
		Timezone:   customer.Timezone,
		Metadata:   metadata,
	}
}
