package models

import "time"

// APIKey maps to the api_keys table.
// Permissions is a JSONB column holding resource→modes mapping, e.g. {"invoice":["read","write"]}.
type APIKey struct {
	BaseModel
	OrganizationID string     `gorm:"column:organization_id;not null;index"`
	Value          string     `gorm:"column:value;not null;uniqueIndex"`
	Name           *string    `gorm:"column:name"`
	Permissions    JSONBMap   `gorm:"column:permissions;type:jsonb"`
	ExpiresAt      *time.Time `gorm:"column:expires_at"`
	LastUsedAt     *time.Time `gorm:"column:last_used_at"`

	Organization Organization `gorm:"foreignKey:OrganizationID"`
}

func (APIKey) TableName() string { return "api_keys" }
