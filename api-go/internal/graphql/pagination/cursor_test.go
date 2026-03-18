package pagination_test

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/getlago/lago/api-go/internal/graphql/pagination"
)

func TestEncodeDecodeOffsetCursor(t *testing.T) {
	inputOffset := 42
	cursor := pagination.EncodeOffsetCursor(inputOffset)

	actualOffset, err := pagination.DecodeOffsetCursor(&cursor)
	require.NoError(t, err)
	assert.Equal(t, inputOffset, actualOffset)
}

func TestDecodeOffsetCursor_EmptyCursor(t *testing.T) {
	actualOffset, err := pagination.DecodeOffsetCursor(nil)
	require.NoError(t, err)
	assert.Equal(t, 0, actualOffset)
}

func TestDecodeOffsetCursor_InvalidPrefix(t *testing.T) {
	invalidCursor := base64.StdEncoding.EncodeToString([]byte("page:1"))

	_, err := pagination.DecodeOffsetCursor(&invalidCursor)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid cursor prefix")
}
