package invoices

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"

	domain "github.com/getlago/lago/api-go/internal/domain/invoices"
	"github.com/getlago/lago/api-go/internal/models"
)

// ErrInvoiceNotFound is returned when an invoice cannot be located by ID.
var ErrInvoiceNotFound = errors.New("invoice_not_found")

// ValidationError captures a user-input validation failure.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string { return e.Message }

// IsValidationError reports whether err is a ValidationError.
func IsValidationError(err error) bool {
	var v *ValidationError
	return errors.As(err, &v)
}

// IsNotFoundError reports whether err signals that an invoice was not found.
func IsNotFoundError(err error) bool {
	return errors.Is(err, ErrInvoiceNotFound)
}

// IsTransitionError reports whether err is a domain state-machine error.
func IsTransitionError(err error) bool {
	return errors.Is(err, domain.ErrInvalidTransition) ||
		errors.Is(err, domain.ErrAlreadyFinalized) ||
		errors.Is(err, domain.ErrAlreadyVoided) ||
		errors.Is(err, domain.ErrCannotVoidDraft)
}

// CreateInvoiceInput holds the data required to create a new draft invoice.
type CreateInvoiceInput struct {
	CustomerID      string             `json:"customer_id"`
	BillingEntityID string             `json:"billing_entity_id"`
	InvoiceType     models.InvoiceType `json:"invoice_type"`
	Currency        string             `json:"currency"`
}

// ListInvoicesFilter contains optional filters for paginated invoice listing.
type ListInvoicesFilter struct {
	CustomerExternalID string
	Status             string
	Page               int
	PerPage            int
}

// Pagination holds metadata returned alongside a paginated result set.
type Pagination struct {
	CurrentPage int
	TotalPages  int
	NextPage    *int
	PrevPage    *int
	TotalCount  int64
}

// Service is the primary interface for invoice operations.
type Service interface {
	Create(ctx context.Context, organizationID string, input CreateInvoiceInput) (*models.Invoice, error)
	List(ctx context.Context, organizationID string, filter ListInvoicesFilter) ([]models.Invoice, *Pagination, error)
	GetByID(ctx context.Context, organizationID string, id string) (*models.Invoice, error)
	Finalize(ctx context.Context, organizationID string, id string) (*models.Invoice, error)
	Void(ctx context.Context, organizationID string, id string) (*models.Invoice, error)
}

type service struct {
	db *gorm.DB
}

// NewService constructs an invoice Service backed by the provided DB.
func NewService(db *gorm.DB) Service {
	return &service{db: db}
}

func (s *service) Create(ctx context.Context, organizationID string, input CreateInvoiceInput) (*models.Invoice, error) {
	if err := validateCreateInput(organizationID, input); err != nil {
		return nil, err
	}

	currency := normalizeCurrency(input.Currency)

	invoice := models.Invoice{
		OrganizationID:  organizationID,
		BillingEntityID: input.BillingEntityID,
		InvoiceType:     input.InvoiceType,
		Currency:        currency,
		Status:          models.InvoiceStatusDraft,
		PaymentStatus:   models.InvoicePaymentStatusPending,
		VersionNumber:   4,
		Timezone:        "UTC",
		Number:          "",
	}

	if strings.TrimSpace(input.CustomerID) != "" {
		customerID := strings.TrimSpace(input.CustomerID)
		invoice.CustomerID = &customerID
	}

	if err := s.db.WithContext(ctx).Create(&invoice).Error; err != nil {
		return nil, err
	}

	return &invoice, nil
}

func (s *service) List(ctx context.Context, organizationID string, filter ListInvoicesFilter) ([]models.Invoice, *Pagination, error) {
	if strings.TrimSpace(organizationID) == "" {
		return nil, nil, &ValidationError{Message: "organization_id is required"}
	}

	page, perPage := normalizePagination(filter.Page, filter.PerPage)

	query := s.db.WithContext(ctx).Model(&models.Invoice{}).
		Where("invoices.organization_id = ?", organizationID)

	if strings.TrimSpace(filter.CustomerExternalID) != "" {
		query = query.Joins("JOIN customers ON customers.id = invoices.customer_id").
			Where("customers.external_id = ?", strings.TrimSpace(filter.CustomerExternalID))
	}

	statusInt := invoiceStatusFromString(filter.Status)
	if statusInt >= 0 {
		query = query.Where("invoices.status = ?", statusInt)
	}

	var totalCount int64
	if err := query.Count(&totalCount).Error; err != nil {
		return nil, nil, err
	}

	var invoices []models.Invoice
	offset := (page - 1) * perPage
	if err := query.Order("invoices.created_at DESC").
		Limit(perPage).
		Offset(offset).
		Find(&invoices).Error; err != nil {
		return nil, nil, err
	}

	pagination := buildPagination(page, perPage, totalCount)
	return invoices, pagination, nil
}

func (s *service) GetByID(ctx context.Context, organizationID string, id string) (*models.Invoice, error) {
	if strings.TrimSpace(organizationID) == "" {
		return nil, &ValidationError{Message: "organization_id is required"}
	}
	if strings.TrimSpace(id) == "" {
		return nil, &ValidationError{Message: "id is required"}
	}

	var invoice models.Invoice
	err := s.db.WithContext(ctx).
		Where("id = ? AND organization_id = ?", id, organizationID).
		First(&invoice).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvoiceNotFound
		}
		return nil, err
	}

	return &invoice, nil
}

func (s *service) Finalize(ctx context.Context, organizationID string, id string) (*models.Invoice, error) {
	invoice, err := s.GetByID(ctx, organizationID, id)
	if err != nil {
		return nil, err
	}

	if err := domain.ApplyFinalize(invoice); err != nil {
		return nil, err
	}

	if err := s.db.WithContext(ctx).Save(invoice).Error; err != nil {
		return nil, err
	}

	return invoice, nil
}

func (s *service) Void(ctx context.Context, organizationID string, id string) (*models.Invoice, error) {
	invoice, err := s.GetByID(ctx, organizationID, id)
	if err != nil {
		return nil, err
	}

	if err := domain.ApplyVoid(invoice); err != nil {
		return nil, err
	}

	if err := s.db.WithContext(ctx).Save(invoice).Error; err != nil {
		return nil, err
	}

	return invoice, nil
}

// validateCreateInput checks that all required fields are present.
func validateCreateInput(organizationID string, input CreateInvoiceInput) error {
	if strings.TrimSpace(organizationID) == "" {
		return &ValidationError{Message: "organization_id is required"}
	}
	if strings.TrimSpace(input.CustomerID) == "" {
		return &ValidationError{Message: "customer_id is required"}
	}
	if strings.TrimSpace(input.BillingEntityID) == "" {
		return &ValidationError{Message: "billing_entity_id is required"}
	}
	if strings.TrimSpace(input.Currency) == "" {
		return &ValidationError{Message: "currency is required"}
	}
	return nil
}

// normalizeCurrency uppercases and trims whitespace from a currency code.
func normalizeCurrency(currency string) string {
	return strings.ToUpper(strings.TrimSpace(currency))
}

// InvoiceStatusFromString converts a status string to its integer value.
// Returns -1 for unknown strings so callers can skip the filter.
// Exported for use in tests.
func InvoiceStatusFromString(status string) int {
	return invoiceStatusFromString(status)
}

func invoiceStatusFromString(status string) int {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "draft":
		return 0
	case "finalized":
		return 1
	case "voided":
		return 2
	case "generating":
		return 3
	case "failed":
		return 4
	default:
		return -1
	}
}

func normalizePagination(page, perPage int) (int, int) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}
	return page, perPage
}

func buildPagination(page, perPage int, totalCount int64) *Pagination {
	totalPages := int(totalCount) / perPage
	if int(totalCount)%perPage != 0 {
		totalPages++
	}
	if totalPages == 0 {
		totalPages = 1
	}

	p := &Pagination{
		CurrentPage: page,
		TotalPages:  totalPages,
		TotalCount:  totalCount,
	}

	if page < totalPages {
		next := page + 1
		p.NextPage = &next
	}
	if page > 1 {
		prev := page - 1
		p.PrevPage = &prev
	}

	return p
}
