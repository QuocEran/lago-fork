package models

// Organization maps to the organizations table.
type Organization struct {
	BaseModel
	Name                           string      `gorm:"column:name;not null"`
	APIKey                         *string     `gorm:"column:api_key"`
	WebhookURL                     *string     `gorm:"column:webhook_url"`
	VatRate                        float64     `gorm:"column:vat_rate;default:0"`
	Country                        *string     `gorm:"column:country"`
	AddressLine1                   *string     `gorm:"column:address_line1"`
	AddressLine2                   *string     `gorm:"column:address_line2"`
	State                          *string     `gorm:"column:state"`
	Zipcode                        *string     `gorm:"column:zipcode"`
	Email                          *string     `gorm:"column:email"`
	City                           *string     `gorm:"column:city"`
	Logo                           *string     `gorm:"column:logo"`
	LegalName                      *string     `gorm:"column:legal_name"`
	LegalNumber                    *string     `gorm:"column:legal_number"`
	InvoiceFooter                  *string     `gorm:"column:invoice_footer;type:text"`
	InvoiceGracePeriod             int         `gorm:"column:invoice_grace_period;default:0"`
	Timezone                       string      `gorm:"column:timezone;default:UTC"`
	DocumentLocale                 string      `gorm:"column:document_locale;default:en"`
	EmailSettings                  StringArray `gorm:"column:email_settings;type:varchar[]"`
	TaxIdentificationNumber        *string     `gorm:"column:tax_identification_number"`
	NetPaymentTerm                 int         `gorm:"column:net_payment_term;default:0"`
	DefaultCurrency                string      `gorm:"column:default_currency;default:USD"`
	DocumentNumbering              int         `gorm:"column:document_numbering;default:0"`
	DocumentNumberPrefix           *string     `gorm:"column:document_number_prefix"`
	EuTaxManagement                bool        `gorm:"column:eu_tax_management;default:false"`
	PremiumIntegrations            StringArray `gorm:"column:premium_integrations;type:varchar[]"`
	CustomAggregation              bool        `gorm:"column:custom_aggregation;default:false"`
	FinalizeZeroAmountInvoice      bool        `gorm:"column:finalize_zero_amount_invoice;default:true"`
	ClickhouseEventsStore          bool        `gorm:"column:clickhouse_events_store;default:false"`
	ClickhouseDeduplicationEnabled bool        `gorm:"column:clickhouse_deduplication_enabled;default:false"`
	HmacKey                        string      `gorm:"column:hmac_key;not null"`
	AuthenticationMethods          StringArray `gorm:"column:authentication_methods;type:varchar[]"`
	AuditLogsPeriod                int         `gorm:"column:audit_logs_period;default:30"`
	PreFilterEvents                bool        `gorm:"column:pre_filter_events;default:false"`
	FeatureFlags                   StringArray `gorm:"column:feature_flags;type:varchar[]"`
	MaxWallets                     *int        `gorm:"column:max_wallets"`

	APIKeys     []APIKey     `gorm:"foreignKey:OrganizationID"`
	Memberships []Membership `gorm:"foreignKey:OrganizationID"`
	Invites     []Invite     `gorm:"foreignKey:OrganizationID"`
	Roles       []Role       `gorm:"foreignKey:OrganizationID"`
}

func (Organization) TableName() string { return "organizations" }
