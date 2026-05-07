package models

import "time"

type Staff struct {
	ID           string     `gorm:"type:char(36);primaryKey" json:"id"`
	RoleID       uint       `gorm:"not null" json:"role_id"`
	Username     string     `gorm:"size:100;uniqueIndex;not null" json:"username"`
	Email        string     `gorm:"size:150;uniqueIndex;not null" json:"email"`
	Phone        *string    `gorm:"size:20" json:"phone"`
	PasswordHash string     `gorm:"size:255;not null" json:"-"`
	// AvatarURL    *string    `gorm:"size:255" json:"avatar_url"`
	IsAllStores  bool       `gorm:"default:false" json:"is_all_stores"`
	CreatedBy    *string    `gorm:"size:255" json:"created_by"`
	UpdatedBy    *string    `gorm:"size:255" json:"updated_by"`
	DeletedBy    *string    `gorm:"size:255" json:"deleted_by"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `gorm:"index" json:"deleted_at"`

	Role        Role         `gorm:"foreignKey:RoleID" json:"role,omitempty"`
	StaffStores []StaffStore `gorm:"foreignKey:StaffID" json:"staff_stores,omitempty"`
}
