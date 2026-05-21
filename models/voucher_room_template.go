package models

// VoucherRoomTemplate adalah junction table voucher ↔ room template.
// Hanya dipakai jika voucher.IsAllRoomTypes = false.
type VoucherRoomTemplate struct {
	ID             uint         `gorm:"primaryKey;autoIncrement" json:"id"`
	VoucherID      string       `gorm:"type:char(36);not null;index" json:"voucher_id"`
	RoomTemplateID uint         `gorm:"not null" json:"room_template_id"`
	RoomTemplate   RoomTemplate `gorm:"foreignKey:RoomTemplateID" json:"room_template,omitempty"`
}
