package models

import "time"

// PlayCreditsPackage adalah master paket play credits yang dijual admin.
// apply_to_all_stores=true → berlaku di semua cabang.
// apply_to_all_stores=false → cek PlayCreditsPackageStores.
type PlayCreditsPackage struct {
	ID               string     `gorm:"type:char(36);primaryKey" json:"id"`
	Name             string     `gorm:"size:100;not null" json:"name"`
	IconURL          *string    `gorm:"size:255" json:"icon_url"`
	TotalHours       float64    `gorm:"type:decimal(8,1);not null" json:"total_hours"`
	Price            float64    `gorm:"type:decimal(12,2);not null" json:"price"`
	ValidityDays     uint       `gorm:"not null" json:"validity_days"`
	Description      *string    `gorm:"type:text" json:"description"`
	IsActive         bool       `gorm:"default:true" json:"is_active"`
	ApplyToAllStores bool       `gorm:"default:false" json:"apply_to_all_stores"`
	CreatedBy        *string    `gorm:"size:255" json:"created_by"`
	UpdatedBy        *string    `gorm:"size:255" json:"updated_by"`
	DeletedBy        *string    `gorm:"size:255" json:"deleted_by"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	DeletedAt        *time.Time `gorm:"index" json:"deleted_at"`

	PackageStores []PlayCreditsPackageStore `gorm:"foreignKey:PackageID" json:"package_stores,omitempty"`
}
