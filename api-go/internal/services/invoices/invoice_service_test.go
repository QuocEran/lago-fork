package invoices_test

import (
	"context"
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	domain "github.com/getlago/lago/api-go/internal/domain/invoices"
	"github.com/getlago/lago/api-go/internal/models"
	invoiceservices "github.com/getlago/lago/api-go/internal/services/invoices"
)

func newMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	dialector := postgres.New(postgres.Config{
		Conn:                 sqlDB,
		PreferSimpleProtocol: true,
	})
	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)
	return db, mock
}

// ──────────────────────────────────────────
// Validation tests (no DB needed)
// ──────────────────────────────────────────

func TestCreateInvoice_MissingOrganizationID(t *testing.T) {
	db, _ := newMockDB(t)
	svc := invoiceservices.NewService(db)

	_, actualErr := svc.Create(context.Background(), "", invoiceservices.CreateInvoiceInput{
		CustomerID: "cust-1", BillingEntityID: "b-1", Currency: "USD",
	})

	assert.True(t, invoiceservices.IsValidationError(actualErr))
	assert.Contains(t, actualErr.Error(), "organization_id")
}

func TestCreateInvoice_MissingCustomerID(t *testing.T) {
	db, _ := newMockDB(t)
	svc := invoiceservices.NewService(db)

	_, actualErr := svc.Create(context.Background(), "org-1", invoiceservices.CreateInvoiceInput{
		BillingEntityID: "b-1", Currency: "USD",
	})

	assert.True(t, invoiceservices.IsValidationError(actualErr))
	assert.Contains(t, actualErr.Error(), "customer_id")
}

func TestCreateInvoice_MissingBillingEntityID(t *testing.T) {
	db, _ := newMockDB(t)
	svc := invoiceservices.NewService(db)

	_, actualErr := svc.Create(context.Background(), "org-1", invoiceservices.CreateInvoiceInput{
		CustomerID: "cust-1", Currency: "USD",
	})

	assert.True(t, invoiceservices.IsValidationError(actualErr))
	assert.Contains(t, actualErr.Error(), "billing_entity_id")
}

func TestCreateInvoice_MissingCurrency(t *testing.T) {
	db, _ := newMockDB(t)
	svc := invoiceservices.NewService(db)

	_, actualErr := svc.Create(context.Background(), "org-1", invoiceservices.CreateInvoiceInput{
		CustomerID: "cust-1", BillingEntityID: "b-1",
	})

	assert.True(t, invoiceservices.IsValidationError(actualErr))
	assert.Contains(t, actualErr.Error(), "currency")
}

// ──────────────────────────────────────────
// CreateInvoice — happy path
// ──────────────────────────────────────────

func TestCreateInvoice_Success(t *testing.T) {
	db, mock := newMockDB(t)
	svc := invoiceservices.NewService(db)

	mockRows := sqlmock.NewRows([]string{
		"id", "organization_id", "billing_entity_id", "customer_id",
		"status", "payment_status", "invoice_type", "currency",
		"number", "sequential_id", "fees_amount_cents", "taxes_amount_cents",
		"total_amount_cents", "version_number", "net_payment_term",
		"issuing_date", "payment_due_date", "finalized_at", "voided_at",
		"created_at", "updated_at",
	}).AddRow(
		"invoice-uuid-1", "org-1", "billing-1", "cust-1",
		int(models.InvoiceStatusDraft), int(models.InvoicePaymentStatusPending), int(models.InvoiceTypeOneOff), "EUR",
		"", nil, 0, 0,
		0, 4, 0,
		nil, nil, nil, nil,
		time.Now(), time.Now(),
	)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "invoices"`)).
		WillReturnRows(mockRows)
	mock.ExpectCommit()

	inputInput := invoiceservices.CreateInvoiceInput{
		CustomerID:      "cust-1",
		BillingEntityID: "billing-1",
		InvoiceType:     models.InvoiceTypeOneOff,
		Currency:        "EUR",
	}

	actualInvoice, err := svc.Create(context.Background(), "org-1", inputInput)

	require.NoError(t, err)
	assert.Equal(t, "invoice-uuid-1", actualInvoice.ID)
	assert.Equal(t, models.InvoiceStatusDraft, actualInvoice.Status)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ──────────────────────────────────────────
// FinalizeInvoice
// ──────────────────────────────────────────

func TestFinalizeInvoice_Success(t *testing.T) {
	db, mock := newMockDB(t)
	svc := invoiceservices.NewService(db)

	now := time.Now()
	fetchRows := sqlmock.NewRows([]string{
		"id", "organization_id", "billing_entity_id", "customer_id",
		"status", "payment_status", "invoice_type", "currency",
		"number", "sequential_id", "fees_amount_cents", "taxes_amount_cents",
		"total_amount_cents", "version_number", "net_payment_term",
		"issuing_date", "payment_due_date", "finalized_at", "voided_at",
		"created_at", "updated_at",
	}).AddRow(
		"inv-1", "org-1", "billing-1", nil,
		int(models.InvoiceStatusDraft), int(models.InvoicePaymentStatusPending), int(models.InvoiceTypeSubscription), "USD",
		"", nil, 0, 0,
		0, 4, 0,
		nil, nil, nil, nil,
		now, now,
	)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "invoices"`)).
		WillReturnRows(fetchRows)
	mock.ExpectBegin()
	// computeAndApplyTotals: load fees
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "fees"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "amount_cents", "taxes_rate", "organization_id", "billing_entity_id"}).
			AddRow("fee-1", int64(5000), 0.0, "org-1", "billing-1"))
	// assignSequentialNumber: MAX(org_sequential_id)
	mock.ExpectQuery(`SELECT COALESCE`).
		WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(0))
	mock.ExpectExec(`UPDATE "invoices"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	actualInvoice, err := svc.Finalize(context.Background(), "org-1", "inv-1")

	require.NoError(t, err)
	assert.Equal(t, models.InvoiceStatusFinalized, actualInvoice.Status)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFinalizeInvoice_AlreadyVoided_ReturnsTransitionError(t *testing.T) {
	db, mock := newMockDB(t)
	svc := invoiceservices.NewService(db)

	now := time.Now()
	fetchRows := sqlmock.NewRows([]string{
		"id", "organization_id", "billing_entity_id", "customer_id",
		"status", "payment_status", "invoice_type", "currency",
		"number", "sequential_id", "fees_amount_cents", "taxes_amount_cents",
		"total_amount_cents", "version_number", "net_payment_term",
		"issuing_date", "payment_due_date", "finalized_at", "voided_at",
		"created_at", "updated_at",
	}).AddRow(
		"inv-2", "org-1", "billing-1", nil,
		int(models.InvoiceStatusVoided), int(models.InvoicePaymentStatusPending), int(models.InvoiceTypeSubscription), "USD",
		"", nil, 0, 0,
		0, 4, 0,
		nil, nil, nil, &now,
		now, now,
	)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "invoices"`)).
		WillReturnRows(fetchRows)

	_, actualErr := svc.Finalize(context.Background(), "org-1", "inv-2")

	require.Error(t, actualErr)
	assert.True(t, invoiceservices.IsTransitionError(actualErr))
	assert.ErrorIs(t, actualErr, domain.ErrAlreadyVoided)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ──────────────────────────────────────────
// VoidInvoice
// ──────────────────────────────────────────

func TestVoidInvoice_Success(t *testing.T) {
	db, mock := newMockDB(t)
	svc := invoiceservices.NewService(db)

	now := time.Now()
	fetchRows := sqlmock.NewRows([]string{
		"id", "organization_id", "billing_entity_id", "customer_id",
		"status", "payment_status", "invoice_type", "currency",
		"number", "sequential_id", "fees_amount_cents", "taxes_amount_cents",
		"total_amount_cents", "version_number", "net_payment_term",
		"issuing_date", "payment_due_date", "finalized_at", "voided_at",
		"created_at", "updated_at",
	}).AddRow(
		"inv-3", "org-1", "billing-1", nil,
		int(models.InvoiceStatusFinalized), int(models.InvoicePaymentStatusPending), int(models.InvoiceTypeSubscription), "USD",
		"", nil, 0, 0,
		0, 4, 0,
		nil, nil, &now, nil,
		now, now,
	)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "invoices"`)).
		WillReturnRows(fetchRows)
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "invoices"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	actualInvoice, err := svc.Void(context.Background(), "org-1", "inv-3")

	require.NoError(t, err)
	assert.Equal(t, models.InvoiceStatusVoided, actualInvoice.Status)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestVoidInvoice_DraftInvoice_ReturnsCannotVoidDraft(t *testing.T) {
	db, mock := newMockDB(t)
	svc := invoiceservices.NewService(db)

	now := time.Now()
	fetchRows := sqlmock.NewRows([]string{
		"id", "organization_id", "billing_entity_id", "customer_id",
		"status", "payment_status", "invoice_type", "currency",
		"number", "sequential_id", "fees_amount_cents", "taxes_amount_cents",
		"total_amount_cents", "version_number", "net_payment_term",
		"issuing_date", "payment_due_date", "finalized_at", "voided_at",
		"created_at", "updated_at",
	}).AddRow(
		"inv-4", "org-1", "billing-1", nil,
		int(models.InvoiceStatusDraft), int(models.InvoicePaymentStatusPending), int(models.InvoiceTypeSubscription), "USD",
		"", nil, 0, 0,
		0, 4, 0,
		nil, nil, nil, nil,
		now, now,
	)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "invoices"`)).
		WillReturnRows(fetchRows)

	_, actualErr := svc.Void(context.Background(), "org-1", "inv-4")

	require.Error(t, actualErr)
	assert.True(t, invoiceservices.IsTransitionError(actualErr))
	assert.ErrorIs(t, actualErr, domain.ErrCannotVoidDraft)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ──────────────────────────────────────────
// InvoiceStatusFromString helper
// ──────────────────────────────────────────

func TestInvoiceStatusFromString(t *testing.T) {
	cases := []struct {
		inputStatus    string
		expectedStatus int
	}{
		{"draft", 0},
		{"finalized", 1},
		{"voided", 2},
		{"generating", 3},
		{"failed", 4},
		{"unknown", -1},
		{"", -1},
	}

	for _, tc := range cases {
		actualStatus := invoiceservices.InvoiceStatusFromString(tc.inputStatus)
		assert.Equal(t, tc.expectedStatus, actualStatus, "status string: %q", tc.inputStatus)
	}
}
