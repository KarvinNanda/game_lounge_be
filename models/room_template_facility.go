package models

type RoomTemplateFacility struct {
	ID             uint `gorm:"primaryKey;autoIncrement" json:"id"`
	RoomTemplateID uint `gorm:"not null;index" json:"room_template_id"`
	FacilityID     uint `gorm:"not null;index" json:"facility_id"`
}
