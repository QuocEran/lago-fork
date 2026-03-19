package models

// PlanInterval mirrors the Rails integer enum for plans.interval.
type PlanInterval int

const (
	PlanIntervalWeekly    PlanInterval = 0
	PlanIntervalMonthly   PlanInterval = 1
	PlanIntervalYearly    PlanInterval = 2
	PlanIntervalQuarterly PlanInterval = 3
	PlanIntervalSemiAnnual PlanInterval = 4
)

// PlanIntervalFromString converts a string representation to PlanInterval.
func PlanIntervalFromString(s string) (PlanInterval, bool) {
	m := map[string]PlanInterval{
		"weekly":     PlanIntervalWeekly,
		"monthly":    PlanIntervalMonthly,
		"yearly":     PlanIntervalYearly,
		"quarterly":  PlanIntervalQuarterly,
		"semiannual": PlanIntervalSemiAnnual,
	}
	v, ok := m[s]
	return v, ok
}

// PlanIntervalToString returns the string representation of a PlanInterval.
func PlanIntervalToString(i PlanInterval) string {
	m := map[PlanInterval]string{
		PlanIntervalWeekly:     "weekly",
		PlanIntervalMonthly:    "monthly",
		PlanIntervalYearly:     "yearly",
		PlanIntervalQuarterly:  "quarterly",
		PlanIntervalSemiAnnual: "semiannual",
	}
	if s, ok := m[i]; ok {
		return s
	}
	return "monthly"
}

// Plan maps to the plans table.
type Plan struct {
	SoftDeleteModel
	OrganizationID          string       `gorm:"column:organization_id;not null;index"`
	ParentID                *string      `gorm:"column:parent_id;index"`
	Name                    string       `gorm:"column:name;not null"`
	Code                    string       `gorm:"column:code;not null"`
	Description             *string      `gorm:"column:description"`
	Interval                PlanInterval `gorm:"column:interval;not null"`
	AmountCents             int64        `gorm:"column:amount_cents;not null"`
	AmountCurrency          string       `gorm:"column:amount_currency;not null"`
	PayInAdvance            bool         `gorm:"column:pay_in_advance;not null;default:false"`
	BillChargesMonthly      *bool        `gorm:"column:bill_charges_monthly"`
	BillFixedChargesMonthly bool         `gorm:"column:bill_fixed_charges_monthly;not null;default:false"`
	TrialPeriod             *float64     `gorm:"column:trial_period"`
	InvoiceDisplayName      *string      `gorm:"column:invoice_display_name"`
	PendingDeletion         bool         `gorm:"column:pending_deletion;not null;default:false"`
	Charges                 []Charge     `gorm:"foreignKey:PlanID"`
}

func (Plan) TableName() string { return "plans" }
