package models

import "time"

// PasswordReset tracks one-time password reset tokens for users.
type PasswordReset struct {
	BaseModel
	UserID   string    `gorm:"column:user_id;not null;index"`
	Token    string    `gorm:"column:token;not null;uniqueIndex"`
	ExpireAt time.Time `gorm:"column:expire_at;not null"`

	User User `gorm:"foreignKey:UserID"`
}

func (PasswordReset) TableName() string { return "password_resets" }
