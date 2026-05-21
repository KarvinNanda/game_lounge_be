package models

import "time"

type StoreHolidaySchedule struct {
	ID              uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	StoreID         string     `gorm:"type:char(36);not null;index" json:"store_id"`
	Date            time.Time  `gorm:"type:date;not null" json:"date"`
	OpenTime        string     `gorm:"type:time;not null" json:"open_time"`
	CloseTime       string     `gorm:"type:time;not null" json:"close_time"`
	GlobalHolidayID *uint      `gorm:"index" json:"global_holiday_id"` // NULL = manual (tidak dari global holiday)
	CreatedBy       *string    `gorm:"size:255" json:"created_by"`
	UpdatedBy       *string    `gorm:"size:255" json:"updated_by"`
	DeletedBy       *string    `gorm:"size:255" json:"deleted_by"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	DeletedAt       *time.Time `gorm:"index" json:"deleted_at"`

	Store Store `gorm:"foreignKey:StoreID" json:"store,omitempty"`
}
