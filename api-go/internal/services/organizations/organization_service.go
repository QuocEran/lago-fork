package organizations

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"

	"github.com/getlago/lago/api-go/internal/models"
)

var ErrOrganizationNotFound = errors.New("organization_not_found")

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

type BillingConfigurationInput struct {
	InvoiceFooter      *string `json:"invoice_footer"`
	InvoiceGracePeriod *int    `json:"invoice_grace_period"`
	DocumentLocale     *string `json:"document_locale"`
}

type UpdateOrganizationInput struct {
	Country                   *string                    `json:"country"`
	DefaultCurrency           *string                    `json:"default_currency"`
	AddressLine1              *string                    `json:"address_line1"`
	AddressLine2              *string                    `json:"address_line2"`
	State                     *string                    `json:"state"`
	Zipcode                   *string                    `json:"zipcode"`
	Email                     *string                    `json:"email"`
	City                      *string                    `json:"city"`
	LegalName                 *string                    `json:"legal_name"`
	LegalNumber               *string                    `json:"legal_number"`
	NetPaymentTerm            *int                       `json:"net_payment_term"`
	TaxIdentificationNumber   *string                    `json:"tax_identification_number"`
	Timezone                  *string                    `json:"timezone"`
	WebhookURL                *string                    `json:"webhook_url"`
	DocumentNumbering         *int                       `json:"document_numbering"`
	DocumentNumberPrefix      *string                    `json:"document_number_prefix"`
	FinalizeZeroAmountInvoice *bool                      `json:"finalize_zero_amount_invoice"`
	EmailSettings             *[]string                  `json:"email_settings"`
	BillingConfiguration      *BillingConfigurationInput `json:"billing_configuration"`
}

type Service interface {
	Get(ctx context.Context, organizationID string) (*models.Organization, error)
	Update(ctx context.Context, organizationID string, input UpdateOrganizationInput) (*models.Organization, error)
}

type service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) Service {
	return &service{db: db}
}

func (s *service) Get(ctx context.Context, organizationID string) (*models.Organization, error) {
	var organization models.Organization
	if err := s.db.WithContext(ctx).First(&organization, "id = ?", organizationID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrganizationNotFound
		}
		return nil, err
	}
	return &organization, nil
}

func (s *service) Update(ctx context.Context, organizationID string, input UpdateOrganizationInput) (*models.Organization, error) {
	if err := validateUpdateInput(input); err != nil {
		return nil, err
	}

	organization, err := s.Get(ctx, organizationID)
	if err != nil {
		return nil, err
	}

	applyUpdateInput(organization, input)
	if err := s.db.WithContext(ctx).Save(organization).Error; err != nil {
		return nil, err
	}

	return organization, nil
}

func validateUpdateInput(input UpdateOrganizationInput) error {
	if input.DefaultCurrency != nil {
		currency := strings.TrimSpace(*input.DefaultCurrency)
		if len(currency) != 3 {
			return &ValidationError{Message: "default_currency must be a 3-letter ISO code"}
		}
	}

	if input.Country != nil {
		country := strings.TrimSpace(*input.Country)
		if len(country) != 2 {
			return &ValidationError{Message: "country must be a 2-letter ISO code"}
		}
	}

	if input.Email != nil {
		email := strings.TrimSpace(*input.Email)
		if email != "" && !strings.Contains(email, "@") {
			return &ValidationError{Message: "email is invalid"}
		}
	}

	if input.NetPaymentTerm != nil && *input.NetPaymentTerm < 0 {
		return &ValidationError{Message: "net_payment_term cannot be negative"}
	}

	if input.BillingConfiguration != nil && input.BillingConfiguration.InvoiceGracePeriod != nil {
		if *input.BillingConfiguration.InvoiceGracePeriod < 0 {
			return &ValidationError{Message: "invoice_grace_period cannot be negative"}
		}
	}

	return nil
}

func applyUpdateInput(organization *models.Organization, input UpdateOrganizationInput) {
	if input.Country != nil {
		country := strings.ToLower(strings.TrimSpace(*input.Country))
		organization.Country = stringPtrOrNil(country)
	}
	if input.DefaultCurrency != nil {
		organization.DefaultCurrency = strings.ToUpper(strings.TrimSpace(*input.DefaultCurrency))
	}
	if input.AddressLine1 != nil {
		organization.AddressLine1 = input.AddressLine1
	}
	if input.AddressLine2 != nil {
		organization.AddressLine2 = input.AddressLine2
	}
	if input.State != nil {
		organization.State = input.State
	}
	if input.Zipcode != nil {
		organization.Zipcode = input.Zipcode
	}
	if input.Email != nil {
		email := strings.TrimSpace(*input.Email)
		organization.Email = stringPtrOrNil(email)
	}
	if input.City != nil {
		organization.City = input.City
	}
	if input.LegalName != nil {
		organization.LegalName = input.LegalName
	}
	if input.LegalNumber != nil {
		organization.LegalNumber = input.LegalNumber
	}
	if input.NetPaymentTerm != nil {
		organization.NetPaymentTerm = *input.NetPaymentTerm
	}
	if input.TaxIdentificationNumber != nil {
		organization.TaxIdentificationNumber = input.TaxIdentificationNumber
	}
	if input.Timezone != nil {
		timezone := strings.TrimSpace(*input.Timezone)
		if timezone != "" {
			organization.Timezone = timezone
		}
	}
	if input.WebhookURL != nil {
		organization.WebhookURL = input.WebhookURL
	}
	if input.DocumentNumbering != nil {
		organization.DocumentNumbering = *input.DocumentNumbering
	}
	if input.DocumentNumberPrefix != nil {
		organization.DocumentNumberPrefix = input.DocumentNumberPrefix
	}
	if input.FinalizeZeroAmountInvoice != nil {
		organization.FinalizeZeroAmountInvoice = *input.FinalizeZeroAmountInvoice
	}
	if input.EmailSettings != nil {
		organization.EmailSettings = models.StringArray(*input.EmailSettings)
	}
	if input.BillingConfiguration != nil {
		if input.BillingConfiguration.InvoiceFooter != nil {
			organization.InvoiceFooter = input.BillingConfiguration.InvoiceFooter
		}
		if input.BillingConfiguration.InvoiceGracePeriod != nil {
			organization.InvoiceGracePeriod = *input.BillingConfiguration.InvoiceGracePeriod
		}
		if input.BillingConfiguration.DocumentLocale != nil {
			documentLocale := strings.TrimSpace(*input.BillingConfiguration.DocumentLocale)
			if documentLocale == "" {
				organization.DocumentLocale = "en"
			} else {
				organization.DocumentLocale = documentLocale
			}
		}
	}
}

func stringPtrOrNil(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}
