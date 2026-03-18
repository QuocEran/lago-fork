package models_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/getlago/lago/api-go/internal/models"
)

func TestStringArray_Value_Nil(t *testing.T) {
	var s models.StringArray
	v, err := s.Value()
	require.NoError(t, err)
	assert.Equal(t, "{}", v)
}

func TestStringArray_Value_Elements(t *testing.T) {
	s := models.StringArray{"invoice.finalized", "credit_note.created"}
	v, err := s.Value()
	require.NoError(t, err)
	assert.Equal(t, `{"invoice.finalized","credit_note.created"}`, v)
}

func TestStringArray_Scan_Nil(t *testing.T) {
	var s models.StringArray
	require.NoError(t, s.Scan(nil))
	assert.Equal(t, models.StringArray{}, s)
}

func TestStringArray_Scan_Empty(t *testing.T) {
	var s models.StringArray
	require.NoError(t, s.Scan("{}"))
	assert.Equal(t, models.StringArray{}, s)
}

func TestStringArray_Scan_Elements(t *testing.T) {
	var s models.StringArray
	require.NoError(t, s.Scan(`{"a","b","c"}`))
	assert.Equal(t, models.StringArray{"a", "b", "c"}, s)
}

func TestStringArray_RoundTrip(t *testing.T) {
	input := models.StringArray{"email_password", "google_oauth"}
	val, err := input.Value()
	require.NoError(t, err)
	var output models.StringArray
	require.NoError(t, output.Scan(val))
	assert.Equal(t, input, output)
}

func TestJSONBMap_Value_Nil(t *testing.T) {
	var j models.JSONBMap
	v, err := j.Value()
	require.NoError(t, err)
	assert.Nil(t, v)
}

func TestJSONBMap_Value_Data(t *testing.T) {
	j := models.JSONBMap{"invoice": []any{"read", "write"}}
	v, err := j.Value()
	require.NoError(t, err)
	assert.NotNil(t, v)
	assert.Contains(t, v.(string), `"invoice"`)
}

func TestJSONBMap_Scan_Nil(t *testing.T) {
	var j models.JSONBMap
	require.NoError(t, j.Scan(nil))
	assert.Equal(t, models.JSONBMap{}, j)
}

func TestJSONBMap_Scan_Bytes(t *testing.T) {
	var j models.JSONBMap
	require.NoError(t, j.Scan([]byte(`{"invoice":["read","write"]}`)))
	perms, ok := j["invoice"].([]any)
	require.True(t, ok)
	assert.Contains(t, perms, "read")
}

func TestJSONBMap_RoundTrip(t *testing.T) {
	input := models.JSONBMap{"plan": []any{"read"}, "customer": []any{"read", "write"}}
	val, err := input.Value()
	require.NoError(t, err)
	var output models.JSONBMap
	require.NoError(t, output.Scan(val))
	assert.Equal(t, len(input), len(output))
}

func TestTableNames(t *testing.T) {
	cases := []struct {
		model     interface{ TableName() string }
		wantTable string
	}{
		{models.Organization{}, "organizations"},
		{models.BillingEntity{}, "billing_entities"},
		{models.User{}, "users"},
		{models.Membership{}, "memberships"},
		{models.Role{}, "roles"},
		{models.MembershipRole{}, "membership_roles"},
		{models.Invite{}, "invites"},
		{models.PasswordReset{}, "password_resets"},
		{models.APIKey{}, "api_keys"},
		{models.Event{}, "events"},
		{models.Customer{}, "customers"},
		{models.CustomerMetadata{}, "customer_metadata"},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.wantTable, tc.model.TableName(), "wrong table name for %T", tc.model)
	}
}

func TestMembershipStatus_Values(t *testing.T) {
	assert.Equal(t, models.MembershipStatus(0), models.MembershipStatusActive)
	assert.Equal(t, models.MembershipStatus(1), models.MembershipStatusRevoked)
}

func TestInviteStatus_Values(t *testing.T) {
	assert.Equal(t, models.InviteStatus(0), models.InviteStatusPending)
	assert.Equal(t, models.InviteStatus(1), models.InviteStatusAccepted)
	assert.Equal(t, models.InviteStatus(2), models.InviteStatusRevoked)
}
