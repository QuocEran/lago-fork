package events_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/getlago/lago/api-go/internal/services/events"
)

// Black-box tests for the events ingestion service use the public API only.

func TestListEventsFilter_DefaultPagination(t *testing.T) {
	filter := events.ListEventsFilter{}
	assert.Equal(t, 0, filter.Page)
	assert.Equal(t, 0, filter.PerPage)
}
