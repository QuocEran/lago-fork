package models

// Role maps to the roles table.
// Roles can be system-wide (organization_id IS NULL) or org-scoped.
// permissions is a text[] column listing allowed permission keys.
type Role struct {
	SoftDeleteModel
	OrganizationID *string     `gorm:"column:organization_id;index"`
	Code           string      `gorm:"column:code;not null"`
	Name           string      `gorm:"column:name;not null"`
	Description    *string     `gorm:"column:description"`
	Admin          bool        `gorm:"column:admin;not null;default:false"`
	Permissions    StringArray `gorm:"column:permissions;type:varchar[]"`

	Organization    *Organization    `gorm:"foreignKey:OrganizationID"`
	MembershipRoles []MembershipRole `gorm:"foreignKey:RoleID"`
}

func (Role) TableName() string { return "roles" }

// MembershipRole maps to the membership_roles join table.
// It uses soft-delete so revoked assignments remain auditable.
type MembershipRole struct {
	SoftDeleteModel
	MembershipID   string `gorm:"column:membership_id;not null;index"`
	RoleID         string `gorm:"column:role_id;not null;index"`
	OrganizationID string `gorm:"column:organization_id;not null"`

	Membership Membership `gorm:"foreignKey:MembershipID"`
	Role       Role       `gorm:"foreignKey:RoleID"`
}

func (MembershipRole) TableName() string { return "membership_roles" }
