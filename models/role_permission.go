package models

type RolePermission struct {
	ID         uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	RoleID     uint   `gorm:"not null;index" json:"role_id"`
	Permission string `gorm:"size:100;not null" json:"permission"`
}
