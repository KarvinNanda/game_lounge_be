package models

import "time"

// Banner adalah konten slider yang tampil di halaman home customer.
// sort_order menentukan urutan tampil (ascending = lebih dulu tampil).
// Hanya banner is_active=true yang tampil di customer app.
type Banner struct {
	ID             uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	Title          string     `gorm:"size:150;not null" json:"title"`
	Subtitle       *string    `gorm:"size:255" json:"subtitle"`
	Description    *string    `gorm:"type:text" json:"description"`
	ImageURL       string     `gorm:"size:255;not null" json:"image_url"`
	DetailImageURL *string    `gorm:"size:255" json:"detail_image_url"`
	SortOrder      uint       `gorm:"default:0" json:"sort_order"`
	IsActive       bool       `gorm:"default:true" json:"is_active"`
	CreatedBy      *string    `gorm:"size:255" json:"created_by"`
	UpdatedBy      *string    `gorm:"size:255" json:"updated_by"`
	DeletedBy      *string    `gorm:"size:255" json:"deleted_by"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	DeletedAt      *time.Time `gorm:"index" json:"deleted_at"`
}
