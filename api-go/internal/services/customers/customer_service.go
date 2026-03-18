package customers

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/getlago/lago/api-go/internal/models"
)

const defaultCustomerPortalBaseURL = "http://localhost:3000/customer-portal"

var ErrCustomerNotFound = errors.New("customer_not_found")

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

type MetadataInput struct {
	Key              string `json:"key"`
	Value            string `json:"value"`
	DisplayInInvoice bool   `json:"display_in_invoice"`
}

type CreateCustomerInput struct {
	ExternalID string          `json:"external_id"`
	Name       *string         `json:"name"`
	Email      *string         `json:"email"`
	Currency   *string         `json:"currency"`
	Timezone   *string         `json:"timezone"`
	Metadata   []MetadataInput `json:"metadata"`
}

type Service interface {
	Create(ctx context.Context, organizationID string, input CreateCustomerInput) (*models.Customer, error)
	List(ctx context.Context, organizationID string) ([]models.Customer, error)
	GetByExternalID(ctx context.Context, organizationID string, externalID string) (*models.Customer, error)
	DeleteByExternalID(ctx context.Context, organizationID string, externalID string) (*models.Customer, error)
	GeneratePortalURL(ctx context.Context, organizationID string, externalID string) (string, error)
}

type service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) Service {
	return &service{db: db}
}

func (s *service) Create(ctx context.Context, organizationID string, input CreateCustomerInput) (*models.Customer, error) {
	if err := validateCreateInput(organizationID, input); err != nil {
		return nil, err
	}

	normalizedInput := normalizeCreateInput(input)

	billingEntityID, err := s.resolveDefaultBillingEntityID(ctx, organizationID)
	if err != nil {
		return nil, err
	}

	customer := models.Customer{
		ExternalID:      normalizedInput.ExternalID,
		OrganizationID:  organizationID,
		Name:            normalizedInput.Name,
		Email:           normalizedInput.Email,
		Currency:        normalizedInput.Currency,
		Timezone:        normalizedInput.Timezone,
		AccountType:     "customer",
		BillingEntityID: billingEntityID,
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if createErr := tx.Create(&customer).Error; createErr != nil {
			return createErr
		}

		if len(normalizedInput.Metadata) == 0 {
			return nil
		}

		metadataRows := make([]models.CustomerMetadata, 0, len(normalizedInput.Metadata))
		for _, metadata := range normalizedInput.Metadata {
			metadataRows = append(metadataRows, models.CustomerMetadata{
				CustomerID:       customer.ID,
				OrganizationID:   organizationID,
				Key:              metadata.Key,
				Value:            metadata.Value,
				DisplayInInvoice: metadata.DisplayInInvoice,
			})
		}

		return tx.Create(&metadataRows).Error
	})
	if err != nil {
		return nil, err
	}

	return s.GetByExternalID(ctx, organizationID, normalizedInput.ExternalID)
}

func (s *service) List(ctx context.Context, organizationID string) ([]models.Customer, error) {
	if strings.TrimSpace(organizationID) == "" {
		return nil, &ValidationError{Message: "organization_id is required"}
	}

	customers := make([]models.Customer, 0)
	if err := s.db.WithContext(ctx).
		Where("organization_id = ?", organizationID).
		Preload("Metadata").
		Order("created_at DESC").
		Find(&customers).Error; err != nil {
		return nil, err
	}

	return customers, nil
}

func (s *service) GetByExternalID(ctx context.Context, organizationID string, externalID string) (*models.Customer, error) {
	if strings.TrimSpace(organizationID) == "" {
		return nil, &ValidationError{Message: "organization_id is required"}
	}
	if strings.TrimSpace(externalID) == "" {
		return nil, &ValidationError{Message: "external_id is required"}
	}

	var customer models.Customer
	err := s.db.WithContext(ctx).
		Where("organization_id = ? AND external_id = ?", organizationID, externalID).
		Preload("Metadata").
		First(&customer).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCustomerNotFound
		}
		return nil, err
	}

	return &customer, nil
}

func (s *service) DeleteByExternalID(ctx context.Context, organizationID string, externalID string) (*models.Customer, error) {
	customer, err := s.GetByExternalID(ctx, organizationID, externalID)
	if err != nil {
		return nil, err
	}

	if err := s.db.WithContext(ctx).Delete(customer).Error; err != nil {
		return nil, err
	}

	return customer, nil
}

func (s *service) GeneratePortalURL(ctx context.Context, organizationID string, externalID string) (string, error) {
	customer, err := s.GetByExternalID(ctx, organizationID, externalID)
	if err != nil {
		return "", err
	}

	var organization models.Organization
	if err := s.db.WithContext(ctx).First(&organization, "id = ?", organizationID).Error; err != nil {
		return "", err
	}

	portalToken := buildPortalToken(customer.ID, organization.HmacKey)
	return buildPortalURL(customer.ExternalID, portalToken), nil
}

func validateCreateInput(organizationID string, input CreateCustomerInput) error {
	if strings.TrimSpace(organizationID) == "" {
		return &ValidationError{Message: "organization_id is required"}
	}
	if strings.TrimSpace(input.ExternalID) == "" {
		return &ValidationError{Message: "external_id is required"}
	}
	if input.Currency != nil && strings.TrimSpace(*input.Currency) != "" && len(strings.TrimSpace(*input.Currency)) != 3 {
		return &ValidationError{Message: "currency must be a 3-letter ISO code"}
	}

	for _, metadata := range input.Metadata {
		if strings.TrimSpace(metadata.Key) == "" {
			return &ValidationError{Message: "metadata key is required"}
		}
	}

	return nil
}

func normalizeCreateInput(input CreateCustomerInput) CreateCustomerInput {
	input.ExternalID = strings.TrimSpace(input.ExternalID)

	if input.Name != nil {
		trimmedName := strings.TrimSpace(*input.Name)
		input.Name = stringPtrOrNil(trimmedName)
	}
	if input.Email != nil {
		trimmedEmail := strings.TrimSpace(*input.Email)
		input.Email = stringPtrOrNil(trimmedEmail)
	}
	if input.Currency != nil {
		trimmedCurrency := strings.ToUpper(strings.TrimSpace(*input.Currency))
		input.Currency = stringPtrOrNil(trimmedCurrency)
	}
	if input.Timezone != nil {
		trimmedTimezone := strings.TrimSpace(*input.Timezone)
		input.Timezone = stringPtrOrNil(trimmedTimezone)
	}

	for i := range input.Metadata {
		input.Metadata[i].Key = strings.TrimSpace(input.Metadata[i].Key)
		input.Metadata[i].Value = strings.TrimSpace(input.Metadata[i].Value)
	}

	return input
}

func (s *service) resolveDefaultBillingEntityID(ctx context.Context, organizationID string) (string, error) {
	var defaultBillingEntity models.BillingEntity
	err := s.db.WithContext(ctx).
		Where("organization_id = ? AND is_default = ?", organizationID, true).
		First(&defaultBillingEntity).Error
	if err == nil {
		return defaultBillingEntity.ID, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", err
	}

	var fallbackBillingEntity models.BillingEntity
	err = s.db.WithContext(ctx).
		Where("organization_id = ?", organizationID).
		First(&fallbackBillingEntity).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", &ValidationError{Message: "organization billing entity is required"}
		}
		return "", err
	}

	return fallbackBillingEntity.ID, nil
}

func buildPortalToken(customerID string, hmacKey string) string {
	nowUnix := time.Now().Unix()
	payload := fmt.Sprintf("%s:%d", customerID, nowUnix)

	mac := hmac.New(sha256.New, []byte(hmacKey))
	_, _ = mac.Write([]byte(payload))
	signature := hex.EncodeToString(mac.Sum(nil))

	tokenPayload := payload + "." + signature
	return base64.RawURLEncoding.EncodeToString([]byte(tokenPayload))
}

func buildPortalURL(externalID string, portalToken string) string {
	baseURL := strings.TrimSpace(os.Getenv("CUSTOMER_PORTAL_BASE_URL"))
	if baseURL == "" {
		baseURL = defaultCustomerPortalBaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")

	return fmt.Sprintf("%s/%s?token=%s", baseURL, externalID, portalToken)
}

func stringPtrOrNil(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	return &value
}
