package models

import "time"

// Customer menyimpan data pelanggan (member atau regular).
// password_hash null saat dibuat, terisi setelah generate & kirim via email.
type Customer struct {
	ID           string     `gorm:"type:char(36);primaryKey" json:"id"`
	Name         string     `gorm:"size:150;not null" json:"name"`
	Whatsapp     string     `gorm:"size:20;not null" json:"whatsapp"`
	Email        *string    `gorm:"size:150;uniqueIndex" json:"email"`
	PasswordHash *string    `gorm:"size:255" json:"-"`
	AvatarURL    *string    `gorm:"size:255" json:"avatar_url"`
	DateOfBirth  *time.Time `gorm:"type:date" json:"date_of_birth"`
	Gender       *string    `gorm:"type:enum('male','female')" json:"gender"`
	Occupation   *string    `gorm:"size:100" json:"occupation"`
	Type         string     `gorm:"type:enum('member','regular');default:'regular'" json:"type"`
	Status       string     `gorm:"type:enum('active','inactive');default:'active'" json:"status"`
	Notes        *string    `gorm:"type:text" json:"notes"`
	CreatedBy    *string    `gorm:"size:255" json:"created_by"`
	UpdatedBy    *string    `gorm:"size:255" json:"updated_by"`
	DeletedBy    *string    `gorm:"size:255" json:"deleted_by"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `gorm:"index" json:"deleted_at"`

	// Relasi
	FavoriteRoomTypes []CustomerFavoriteRoomType `gorm:"foreignKey:CustomerID" json:"favorite_room_types,omitempty"`
	BookingHistory []Booking `gorm:"foreignKey:CustomerID" json:"booking_history,omitempty"`
}
