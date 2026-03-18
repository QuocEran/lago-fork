package events

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
)

func TestIsUniqueViolation_WithPostgresCode(t *testing.T) {
	inputErr := &pgconn.PgError{Code: "23505"}

	actualResult := isUniqueViolation(inputErr)

	assert.True(t, actualResult)
}

func TestIsUniqueViolation_WithWrappedPostgresCode(t *testing.T) {
	inputErr := fmt.Errorf("outer: %w", &pgconn.PgError{Code: "23505"})

	actualResult := isUniqueViolation(inputErr)

	assert.True(t, actualResult)
}

func TestIsUniqueViolation_WithUniqueConstraintMessage(t *testing.T) {
	inputErr := errors.New("duplicate key value violates unique constraint idx_events_org_transaction_unique")

	actualResult := isUniqueViolation(inputErr)

	assert.True(t, actualResult)
}

func TestIsUniqueViolation_WithNonUniqueError(t *testing.T) {
	inputErr := errors.New("connection reset by peer")

	actualResult := isUniqueViolation(inputErr)

	assert.False(t, actualResult)
}
