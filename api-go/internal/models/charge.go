package models

// ChargeModel mirrors the Rails integer enum for charges.charge_model.
type ChargeModel int

const (
	ChargeModelStandard           ChargeModel = 0
	ChargeModelGraduated          ChargeModel = 1
	ChargeModelPackage            ChargeModel = 2
	ChargeModelPercentage         ChargeModel = 3
	ChargeModelVolume             ChargeModel = 4
	ChargeModelGraduatedPercentage ChargeModel = 5
	ChargeModelCustom             ChargeModel = 6
	ChargeModelDynamic            ChargeModel = 7
)

// ChargeModelFromString converts a string representation to ChargeModel.
func ChargeModelFromString(s string) (ChargeModel, bool) {
	m := map[string]ChargeModel{
		"standard":            ChargeModelStandard,
		"graduated":           ChargeModelGraduated,
		"package":             ChargeModelPackage,
		"percentage":          ChargeModelPercentage,
		"volume":              ChargeModelVolume,
		"graduated_percentage": ChargeModelGraduatedPercentage,
		"custom":              ChargeModelCustom,
		"dynamic":             ChargeModelDynamic,
	}
	v, ok := m[s]
	return v, ok
}

// ChargeModelToString returns the string representation of a ChargeModel.
func ChargeModelToString(c ChargeModel) string {
	m := map[ChargeModel]string{
		ChargeModelStandard:            "standard",
		ChargeModelGraduated:           "graduated",
		ChargeModelPackage:             "package",
		ChargeModelPercentage:          "percentage",
		ChargeModelVolume:              "volume",
		ChargeModelGraduatedPercentage: "graduated_percentage",
		ChargeModelCustom:              "custom",
		ChargeModelDynamic:             "dynamic",
	}
	if s, ok := m[c]; ok {
		return s
	}
	return "standard"
}

// Charge maps to the charges table.
type Charge struct {
	SoftDeleteModel
	OrganizationID     string        `gorm:"column:organization_id;not null;index"`
	PlanID             string        `gorm:"column:plan_id;not null;index"`
	BillableMetricID   *string       `gorm:"column:billable_metric_id;index"`
	ParentID           *string       `gorm:"column:parent_id;index"`
	ChargeModel        ChargeModel   `gorm:"column:charge_model;not null;default:0"`
	Code               string        `gorm:"column:code;not null"`
	Properties         JSONBMap      `gorm:"column:properties;type:jsonb;not null;default:'{}'"`
	PayInAdvance       bool          `gorm:"column:pay_in_advance;not null;default:false"`
	Invoiceable        bool          `gorm:"column:invoiceable;not null;default:true"`
	Prorated           bool          `gorm:"column:prorated;not null;default:false"`
	MinAmountCents     int64         `gorm:"column:min_amount_cents;not null;default:0"`
	InvoiceDisplayName *string       `gorm:"column:invoice_display_name"`
	RegroupPaidFees    *int          `gorm:"column:regroup_paid_fees"`
	Filters            []ChargeFilter `gorm:"foreignKey:ChargeID"`
}

func (Charge) TableName() string { return "charges" }
