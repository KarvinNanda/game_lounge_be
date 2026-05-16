package models

// VoucherStore adalah junction table: voucher berlaku di cabang mana.
type VoucherStore struct {
	ID        uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	VoucherID string `gorm:"type:char(36);not null;index" json:"voucher_id"`
	StoreID   string `gorm:"type:char(36);not null;index" json:"store_id"`

	Store Store `gorm:"foreignKey:StoreID" json:"store,omitempty"`
}
