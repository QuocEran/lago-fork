package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/getlago/lago/api-go/internal/kafka"
	"github.com/getlago/lago/api-go/internal/server"
)

func TestGraphQLRoute_SchemaIntrospection(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := server.New(nil, nil, "test-version", "test-jwt-secret", &kafka.NoopPublisher{})

	body := `{"query":"query IntrospectionQuery { __schema { queryType { name } } }"}`
	request, err := http.NewRequest(http.MethodPost, "/graphql", strings.NewReader(body))
	require.NoError(t, err)
	request.Header.Set("Content-Type", "application/json")

	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)

	assert.Equal(t, http.StatusOK, response.Code)
	assert.Contains(t, response.Body.String(), "__schema")
	assert.Contains(t, response.Body.String(), "Query")
}
