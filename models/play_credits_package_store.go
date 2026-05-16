package models

// PlayCreditsPackageStore adalah junction table paket ↔ store.
type PlayCreditsPackageStore struct {
	ID        uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	PackageID string `gorm:"type:char(36);not null;index" json:"package_id"`
	StoreID   string `gorm:"type:char(36);not null;index" json:"store_id"`

	Store Store `gorm:"foreignKey:StoreID" json:"store,omitempty"`
}
