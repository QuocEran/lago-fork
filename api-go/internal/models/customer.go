package models

type Customer struct {
	SoftDeleteModel
	ExternalID                string             `gorm:"column:external_id;not null;index"`
	Name                      *string            `gorm:"column:name"`
	OrganizationID            string             `gorm:"column:organization_id;not null;index"`
	Country                   *string            `gorm:"column:country"`
	AddressLine1              *string            `gorm:"column:address_line1"`
	AddressLine2              *string            `gorm:"column:address_line2"`
	State                     *string            `gorm:"column:state"`
	Zipcode                   *string            `gorm:"column:zipcode"`
	Email                     *string            `gorm:"column:email"`
	City                      *string            `gorm:"column:city"`
	LegalName                 *string            `gorm:"column:legal_name"`
	LegalNumber               *string            `gorm:"column:legal_number"`
	Currency                  *string            `gorm:"column:currency"`
	Timezone                  *string            `gorm:"column:timezone"`
	NetPaymentTerm            *int               `gorm:"column:net_payment_term"`
	ExternalSalesforceID      *string            `gorm:"column:external_salesforce_id"`
	FinalizeZeroAmountInvoice int                `gorm:"column:finalize_zero_amount_invoice;not null;default:0"`
	Firstname                 *string            `gorm:"column:firstname"`
	Lastname                  *string            `gorm:"column:lastname"`
	CustomerType              *string            `gorm:"column:customer_type"`
	AccountType               string             `gorm:"column:account_type;not null;default:customer"`
	BillingEntityID           string             `gorm:"column:billing_entity_id;not null"`
	Metadata                  []CustomerMetadata `gorm:"foreignKey:CustomerID"`
}

func (Customer) TableName() string { return "customers" }

type CustomerMetadata struct {
	BaseModel
	CustomerID       string `gorm:"column:customer_id;not null;index"`
	OrganizationID   string `gorm:"column:organization_id;not null;index"`
	Key              string `gorm:"column:key;not null"`
	Value            string `gorm:"column:value;not null"`
	DisplayInInvoice bool   `gorm:"column:display_in_invoice;not null;default:false"`
}

func (CustomerMetadata) TableName() string { return "customer_metadata" }
