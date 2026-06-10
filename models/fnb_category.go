package models

import "time"

// FnbCategory menyimpan kategori menu FnB (Minuman, Makanan Ringan, dst).
type FnbCategory struct {
	ID             uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	Name           string     `gorm:"size:100;not null" json:"name"`
	Description    *string    `gorm:"type:text" json:"description"`
	ImageURL       *string    `gorm:"size:255" json:"image_url"`
	MokaCategoryID *string    `gorm:"size:100;index" json:"moka_category_id"`
	SortOrder      int        `gorm:"default:0" json:"sort_order"`
	IsActive       bool       `gorm:"default:true" json:"is_active"`
	CreatedBy      *string    `gorm:"size:255" json:"created_by"`
	UpdatedBy      *string    `gorm:"size:255" json:"updated_by"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	DeletedAt      *time.Time `gorm:"index" json:"deleted_at"`

	Items []FnbItem `gorm:"foreignKey:CategoryID" json:"items,omitempty"`
}
