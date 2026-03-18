package models

import "time"

// User maps to the users table.
// PasswordDigest stores the bcrypt hash; it is never serialised to JSON responses.
type User struct {
	BaseModel
	Email          string `gorm:"column:email;not null;uniqueIndex"`
	PasswordDigest string `gorm:"column:password_digest;not null"`

	Memberships    []Membership    `gorm:"foreignKey:UserID"`
	PasswordResets []PasswordReset `gorm:"foreignKey:UserID"`
}

func (User) TableName() string { return "users" }

// MembershipStatus mirrors the Rails enum (0=active, 1=revoked).
type MembershipStatus int

const (
	MembershipStatusActive  MembershipStatus = 0
	MembershipStatusRevoked MembershipStatus = 1
)

// Membership maps to the memberships table (user ↔ organization join with status).
type Membership struct {
	BaseModel
	UserID         string           `gorm:"column:user_id;not null;index"`
	OrganizationID string           `gorm:"column:organization_id;not null;index"`
	Status         MembershipStatus `gorm:"column:status;not null;default:0"`
	RevokedAt      *time.Time       `gorm:"column:revoked_at"`

	User            User             `gorm:"foreignKey:UserID"`
	Organization    Organization     `gorm:"foreignKey:OrganizationID"`
	MembershipRoles []MembershipRole `gorm:"foreignKey:MembershipID"`
}

func (Membership) TableName() string { return "memberships" }
