package testutil

import (
	"time"

	"github.com/google/uuid"

	"github.com/getlago/lago/api-go/internal/models"
)

// OrganizationFixture returns a minimal Organization for tests.
func OrganizationFixture(overrides ...func(*models.Organization)) *models.Organization {
	org := &models.Organization{
		BaseModel: models.BaseModel{
			ID:        uuid.New().String(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Name:            "Test Org",
		DefaultCurrency: "USD",
		Timezone:        "UTC",
		HmacKey:         "test-hmac-key",
	}
	for _, fn := range overrides {
		fn(org)
	}
	return org
}

// CustomerFixture returns a minimal Customer for tests.
func CustomerFixture(orgID string, overrides ...func(*models.Customer)) *models.Customer {
	c := &models.Customer{
		OrganizationID: orgID,
		ExternalID:     "ext-cust-1",
		BillingEntityID: "be-1",
	}
	for _, fn := range overrides {
		fn(c)
	}
	return c
}
