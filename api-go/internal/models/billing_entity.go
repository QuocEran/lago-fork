package models

// BillingEntity represents a sub-org billing workspace owned by an Organization.
type BillingEntity struct {
	SoftDeleteModel
	OrganizationID              string      `gorm:"column:organization_id;not null;index"`
	Code                        *string     `gorm:"column:code"`
	Name                        string      `gorm:"column:name;not null"`
	IsDefault                   bool        `gorm:"column:is_default;default:false"`
	Timezone                    string      `gorm:"column:timezone;default:UTC"`
	DefaultCurrency             string      `gorm:"column:default_currency;default:USD"`
	AddressLine1                *string     `gorm:"column:address_line1"`
	AddressLine2                *string     `gorm:"column:address_line2"`
	City                        *string     `gorm:"column:city"`
	Zipcode                     *string     `gorm:"column:zipcode"`
	State                       *string     `gorm:"column:state"`
	Country                     *string     `gorm:"column:country"`
	Email                       *string     `gorm:"column:email"`
	LegalName                   *string     `gorm:"column:legal_name"`
	LegalNumber                 *string     `gorm:"column:legal_number"`
	TaxIdentificationNumber     *string     `gorm:"column:tax_identification_number"`
	Logo                        *string     `gorm:"column:logo"`
	DocumentLocale              string      `gorm:"column:document_locale;default:en"`
	DocumentNumbering           int         `gorm:"column:document_numbering;default:0"`
	DocumentNumberPrefix        *string     `gorm:"column:document_number_prefix"`
	NetPaymentTerm              int         `gorm:"column:net_payment_term;default:0"`
	InvoiceGracePeriod          int         `gorm:"column:invoice_grace_period;default:0"`
	InvoiceFooter               *string     `gorm:"column:invoice_footer;type:text"`
	VatRate                     float64    `gorm:"column:vat_rate;default:0"`
	FinalizeZeroAmountInvoice   bool        `gorm:"column:finalize_zero_amount_invoice;default:true"`
	EmailSettings               StringArray `gorm:"column:email_settings;type:varchar[]"`
	EInvoicingEnabled           bool        `gorm:"column:einvoicing_enabled;default:false"`
	LastSequentialInvoiceNumber int         `gorm:"column:last_sequential_invoice_number;default:0"`
	OrganizationSequentialID    int         `gorm:"column:organization_sequential_id;default:0"`

	Organization Organization `gorm:"foreignKey:OrganizationID"`
}

func (BillingEntity) TableName() string { return "billing_entities" }
