package models

import "time"

// NotificationTemplate menyimpan template pesan Email dan WhatsApp
// yang bisa dikustomisasi admin dari halaman Settings.
// Variabel dinamis menggunakan format: {{nama_variabel}}
type NotificationTemplate struct {
	ID                 uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	NotificationKey    string    `gorm:"size:50;uniqueIndex;not null" json:"notification_key"`
	Name               string    `gorm:"size:150;not null" json:"name"`
	Description        *string   `gorm:"type:text" json:"description"`
	EmailSubject       *string   `gorm:"size:255" json:"email_subject"`
	EmailBody          *string   `gorm:"type:text" json:"email_body"`
	WhatsappBody       *string   `gorm:"type:text" json:"whatsapp_body"`
	AvailableVariables string    `gorm:"type:json;not null" json:"available_variables"`
	IsEmailActive      bool      `gorm:"default:true" json:"is_email_active"`
	IsWhatsappActive   bool      `gorm:"default:true" json:"is_whatsapp_active"`
	CreatedBy          *string   `gorm:"size:255" json:"created_by"`
	UpdatedBy          *string   `gorm:"size:255" json:"updated_by"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}
