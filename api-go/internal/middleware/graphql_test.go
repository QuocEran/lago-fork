package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/getlago/lago/api-go/internal/graphql/graphcontext"
	"github.com/getlago/lago/api-go/internal/middleware"
)

func TestGraphQLAPIKeyContext_InjectsRequestContext(t *testing.T) {
	db, mock := newMockGORM(t)

	organizationID := "org-graphql-123"
	apiKeyID := "key-graphql-456"

	rows := sqlmock.NewRows([]string{"id", "organization_id", "expires_at"}).
		AddRow(apiKeyID, organizationID, nil)
	orgRows := sqlmock.NewRows([]string{"id", "premium_integrations"}).
		AddRow(organizationID, "{}")

	mock.ExpectQuery(`SELECT .* FROM "api_keys"`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(rows)
	mock.ExpectQuery(`SELECT .* FROM "organizations"`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(orgRows)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/graphql", middleware.GraphQLAPIKeyContext(db), func(c *gin.Context) {
		actualOrganizationID, hasOrganizationID := graphcontext.OrganizationIDFromContext(c.Request.Context())
		actualAPIKeyID, hasAPIKeyID := graphcontext.APIKeyIDFromContext(c.Request.Context())
		require.True(t, hasOrganizationID)
		require.True(t, hasAPIKeyID)
		assert.Equal(t, organizationID, actualOrganizationID)
		assert.Equal(t, apiKeyID, actualAPIKeyID)
		c.Status(http.StatusOK)
	})

	request, err := http.NewRequest(http.MethodPost, "/graphql", nil)
	require.NoError(t, err)
	request.Header.Set("Authorization", "Bearer valid-token")

	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	assert.Equal(t, http.StatusOK, response.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}
