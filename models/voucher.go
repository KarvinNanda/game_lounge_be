package models

import "time"

// Voucher adalah kode diskon yang hanya bisa dipakai oleh customer bertipe 'member'.
// Penggunaan dibatasi 1x per user (enforce via UNIQUE di voucher_usages).
// Pengiriman notifikasi (Email/WhatsApp) dilakukan saat voucher dibuat.
type Voucher struct {
	ID            string     `gorm:"type:char(36);primaryKey" json:"id"`
	Name          string     `gorm:"size:150;not null" json:"name"`
	Code          string     `gorm:"size:50;uniqueIndex;not null" json:"code"`
	Description   *string    `gorm:"type:text" json:"description"`
	Type          string     `gorm:"type:enum('booking','play_credits','both');default:'booking'" json:"type"`
	DiscountType  string     `gorm:"type:enum('percentage','nominal');not null" json:"discount_type"`
	DiscountValue float64    `gorm:"type:decimal(12,2);not null" json:"discount_value"`
	MaxDiscount   *float64   `gorm:"type:decimal(12,2)" json:"max_discount"`
	MinPurchase   *float64   `gorm:"type:decimal(12,2)" json:"min_purchase"`
	StartDate     time.Time  `gorm:"type:date;not null" json:"start_date"`
	EndDate       *time.Time `gorm:"type:date" json:"end_date"`
	SendChannel   *string    `gorm:"type:enum('whatsapp','email','all')" json:"send_channel"`
	TotalSent     uint       `gorm:"default:0" json:"total_sent"`
	UsedCount     uint       `gorm:"default:0" json:"used_count"`
	IsAllStores   bool       `gorm:"default:true" json:"is_all_stores"`
	IsAllRoomTypes bool      `gorm:"default:true" json:"is_all_room_types"`
	IsActive      bool       `gorm:"default:true" json:"is_active"`
	CreatedBy     *string    `gorm:"size:255" json:"created_by"`
	UpdatedBy     *string    `gorm:"size:255" json:"updated_by"`
	DeletedBy     *string    `gorm:"size:255" json:"deleted_by"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	DeletedAt     *time.Time `gorm:"index" json:"deleted_at"`

	Stores        []VoucherStore        `gorm:"foreignKey:VoucherID" json:"stores,omitempty"`
	Usages        []VoucherUsage        `gorm:"foreignKey:VoucherID" json:"usages,omitempty"`
	RoomTemplates []VoucherRoomTemplate `gorm:"foreignKey:VoucherID" json:"room_templates,omitempty"`
}
