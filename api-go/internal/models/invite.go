package models

import "time"

// InviteStatus mirrors the Rails enum (0=pending, 1=accepted, 2=revoked).
type InviteStatus int

const (
	InviteStatusPending  InviteStatus = 0
	InviteStatusAccepted InviteStatus = 1
	InviteStatusRevoked  InviteStatus = 2
)

// Invite maps to the invites table.
// Roles is a text[] column capturing the intended role codes for the invitee.
type Invite struct {
	BaseModel
	OrganizationID string       `gorm:"column:organization_id;not null;index"`
	MembershipID   *string      `gorm:"column:membership_id;index"`
	Email          string       `gorm:"column:email;not null"`
	Token          string       `gorm:"column:token;not null;uniqueIndex"`
	Status         InviteStatus `gorm:"column:status;not null;default:0"`
	Roles          StringArray  `gorm:"column:roles;type:varchar[]"`
	AcceptedAt     *time.Time   `gorm:"column:accepted_at"`
	RevokedAt      *time.Time   `gorm:"column:revoked_at"`

	Organization Organization `gorm:"foreignKey:OrganizationID"`
	Membership   *Membership  `gorm:"foreignKey:MembershipID"`
}

func (Invite) TableName() string { return "invites" }
