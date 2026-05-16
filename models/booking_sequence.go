package models

import "time"

// BookingSequence menyimpan global counter untuk Booking ID.
// Single-row table, id selalu = 1.
// Harus di-seed satu kali: INSERT INTO booking_sequences (id, last_sequence) VALUES (1, 0)
// atau biarkan repository.EnsureSequenceExists() yang menginisialisasi otomatis.
type BookingSequence struct {
	ID           int       `gorm:"primaryKey" json:"id"`
	LastSequence uint      `gorm:"default:0" json:"last_sequence"`
	UpdatedAt    time.Time `json:"updated_at"`
}
