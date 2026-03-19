package models

// ChargeFilter maps to the charge_filters table.
type ChargeFilter struct {
	SoftDeleteModel
	ChargeID           string   `gorm:"column:charge_id;not null;index"`
	OrganizationID     string   `gorm:"column:organization_id;not null;index"`
	InvoiceDisplayName *string  `gorm:"column:invoice_display_name"`
	Properties         JSONBMap `gorm:"column:properties;type:jsonb;not null;default:'{}'"`
}

func (ChargeFilter) TableName() string { return "charge_filters" }
