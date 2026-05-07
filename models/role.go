package models

import "time"

type Role struct {
	ID          uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string     `gorm:"size:100;not null" json:"name"`
	IsSystem    bool       `gorm:"default:false" json:"is_system"`
	CreatedBy   *string    `gorm:"size:255" json:"created_by"`
	UpdatedBy   *string    `gorm:"size:255" json:"updated_by"`
	DeletedBy   *string    `gorm:"size:255" json:"deleted_by"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `gorm:"index" json:"deleted_at"`

	// Relations (eager-loaded via Preload)
	Permissions []RolePermission `gorm:"foreignKey:RoleID" json:"permissions,omitempty"`
}
