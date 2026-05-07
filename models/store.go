package models

import "time"

type Store struct {
	ID          string     `gorm:"type:char(36);primaryKey" json:"id"`
	Name        string     `gorm:"size:100;not null" json:"name"`
	Address     string     `gorm:"type:text;not null" json:"address"`
	Whatsapp    *string    `gorm:"size:20" json:"whatsapp"`
	PostalCode  *string    `gorm:"size:10" json:"postal_code"`
	Description *string    `gorm:"type:text" json:"description"`
	PhotoURL    *string    `gorm:"size:255" json:"photo_url"`
	LinkGmaps   *string    `gorm:"size:500" json:"link_gmaps"`
	Status      string     `gorm:"type:enum('active','inactive','draft');default:'draft'" json:"status"`
	CreatedBy   *string    `gorm:"size:255" json:"created_by"`
	UpdatedBy   *string    `gorm:"size:255" json:"updated_by"`
	DeletedBy   *string    `gorm:"size:255" json:"deleted_by"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `gorm:"index" json:"deleted_at"`

	// Relations (eager-loaded via Preload)
	OperatingHours   []StoreOperatingHour   `gorm:"foreignKey:StoreID" json:"operating_hours,omitempty"`
	HolidaySchedules []StoreHolidaySchedule `gorm:"foreignKey:StoreID" json:"holiday_schedules,omitempty"`
	Rooms            []StoreRoom            `gorm:"foreignKey:StoreID" json:"rooms,omitempty"`
}
