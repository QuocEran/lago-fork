package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/getlago/lago/api-go/internal/middleware"
)

// sentinel error for DB failure scenarios
var errFakeDB = &fakeDBError{}

type fakeDBError struct{}

func (e *fakeDBError) Error() string { return "connection refused" }

func newMockGORM(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { sqlDB.Close() })

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

func newAuthRouter(db *gorm.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/protected", middleware.APIKeyAuth(db), func(c *gin.Context) {
		orgID := c.GetString(middleware.GinKeyOrganizationID)
		c.JSON(http.StatusOK, gin.H{"org_id": orgID})
	})
	return r
}

func TestAPIKeyAuth_MissingHeader(t *testing.T) {
	db, _ := newMockGORM(t)
	r := newAuthRouter(db)

	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "missing_api_key")
}

func TestAPIKeyAuth_MalformedHeader_NoBearer(t *testing.T) {
	db, _ := newMockGORM(t)
	r := newAuthRouter(db)

	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "legit-token-without-bearer")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "missing_api_key")
}

func TestAPIKeyAuth_MalformedHeader_WrongScheme(t *testing.T) {
	db, _ := newMockGORM(t)
	r := newAuthRouter(db)

	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "missing_api_key")
}

func TestAPIKeyAuth_ValidToken(t *testing.T) {
	db, mock := newMockGORM(t)
	r := newAuthRouter(db)

	inputOrgID := "org-uuid-1234"
	inputKeyID := "key-uuid-5678"

	rows := sqlmock.NewRows([]string{"id", "organization_id", "expires_at"}).
		AddRow(inputKeyID, inputOrgID, nil)
	orgRows := sqlmock.NewRows([]string{"id", "premium_integrations"}).
		AddRow(inputOrgID, "{}")

	mock.ExpectQuery(`SELECT .* FROM "api_keys"`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(rows)
	mock.ExpectQuery(`SELECT .* FROM "organizations"`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(orgRows)

	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer valid-token-abc")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), inputOrgID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAPIKeyAuth_ExpiredToken(t *testing.T) {
	db, mock := newMockGORM(t)
	r := newAuthRouter(db)

	_ = time.Now()
	rows := sqlmock.NewRows([]string{"id", "organization_id", "expires_at"})
	mock.ExpectQuery(`SELECT .* FROM "api_keys"`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(rows)

	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer expired-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "invalid_api_key")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAPIKeyAuth_DBError(t *testing.T) {
	db, mock := newMockGORM(t)
	r := newAuthRouter(db)

	mock.ExpectQuery(`SELECT .* FROM "api_keys"`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnError(errFakeDB)

	req, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer some-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "invalid_api_key")
	assert.NoError(t, mock.ExpectationsWereMet())
}
