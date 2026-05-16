package models

import "time"

// StorePricing menyimpan konfigurasi pricing utama per store.
// 1 store hanya boleh punya 1 pricing config (UNIQUE store_id).
type StorePricing struct {
	ID                 uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	StoreID            string     `gorm:"type:char(36);not null;uniqueIndex" json:"store_id"`
	IsHappyHourEnabled bool       `gorm:"default:true" json:"is_happy_hour_enabled"`
	IsMixedTimeEnabled bool       `gorm:"default:true" json:"is_mixed_time_enabled"`
	EdgeCase2h         string     `gorm:"column:edge_case_2h;type:enum('two_x_1h','force_3h');default:'two_x_1h'" json:"edge_case_2h"`
	EdgeCase4h         string     `gorm:"column:edge_case_4h;type:enum('3h_plus_1h','force_5h');default:'3h_plus_1h'" json:"edge_case_4h"`
	CreatedBy          *string    `gorm:"size:255" json:"created_by"`
	UpdatedBy          *string    `gorm:"size:255" json:"updated_by"`
	DeletedBy          *string    `gorm:"size:255" json:"deleted_by"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	DeletedAt          *time.Time `gorm:"index" json:"deleted_at"`

	// references:StoreID — pakai StorePricing.StoreID (bukan primary key ID) sebagai join key
	Store              Store                    `gorm:"foreignKey:StoreID;references:ID" json:"store,omitempty"`
	HappyHourSchedules []StoreHappyHourSchedule `gorm:"foreignKey:StoreID;references:StoreID" json:"happy_hour_schedules,omitempty"`
	HappyHourPrices    []StoreHappyHourPrice    `gorm:"foreignKey:StoreID;references:StoreID" json:"happy_hour_prices,omitempty"`
	PackagePrices      []StorePackagePrice      `gorm:"foreignKey:StoreID;references:StoreID" json:"package_prices,omitempty"`
	FlashSales         []StoreFlashSale         `gorm:"foreignKey:StoreID;references:StoreID" json:"flash_sales,omitempty"`
}
