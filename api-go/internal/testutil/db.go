package testutil

import (
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// NewDBMock returns a GORM DB and sqlmock for tests.
// The caller must call dbMock.ExpectationsWereMet() in the test.
func NewDBMock(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	dialector := postgres.New(postgres.Config{
		Conn: sqlDB,
	})
	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	return db, mock
}

// SQLDBFromGORM returns the underlying *sql.DB from a GORM DB for use with sqlmock.
func SQLDBFromGORM(db *gorm.DB) (*sql.DB, error) {
	return db.DB()
}
