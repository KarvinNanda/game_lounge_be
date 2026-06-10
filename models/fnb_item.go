package models

import "time"

// FnbItem menyimpan item menu FnB beserta harga dan ketersediaan.
type FnbItem struct {
	ID          uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	CategoryID  uint       `gorm:"not null;index" json:"category_id"`
	Name        string     `gorm:"size:150;not null" json:"name"`
	Description *string    `gorm:"type:text" json:"description"`
	ImageURL    *string    `gorm:"size:255" json:"image_url"`
	Price       float64    `gorm:"type:decimal(12,2);not null;default:0" json:"price"`
	MokaItemID  *string    `gorm:"size:100;index" json:"moka_item_id"`
	IsAvailable bool       `gorm:"default:true" json:"is_available"`
	IsActive    bool       `gorm:"default:true" json:"is_active"`
	SortOrder   int        `gorm:"default:0" json:"sort_order"`
	CreatedBy   *string    `gorm:"size:255" json:"created_by"`
	UpdatedBy   *string    `gorm:"size:255" json:"updated_by"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `gorm:"index" json:"deleted_at"`

	Category FnbCategory `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
}
