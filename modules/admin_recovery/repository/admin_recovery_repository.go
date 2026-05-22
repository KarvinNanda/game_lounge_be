package repository

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"game_lounge_be/config"
	"game_lounge_be/models"
)

// GenerateToken membuat 32-byte random hex token yang unik.
func GenerateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// FindSuperAdminByEmail mencari staff dengan email tersebut yang rolenya super admin (is_system=true).
func FindSuperAdminByEmail(email string) (*models.Staff, error) {
	var staff models.Staff
	err := config.DB.Preload("Role").
		Joins("JOIN roles ON roles.id = staffs.role_id").
		Where("staffs.email = ? AND staffs.deleted_at IS NULL AND roles.is_system = true", email).
		First(&staff).Error
	return &staff, err
}

// CountRequestsFromIP menghitung berapa kali IP ini request dalam 1 jam terakhir.
// Dipakai untuk rate limiting: max 3 request/jam/IP.
func CountRequestsFromIP(ip string) (int64, error) {
	var count int64
	err := config.DB.Model(&models.PasswordResetToken{}).
		Where("ip_address = ? AND created_at > ?", ip, time.Now().Add(-1*time.Hour)).
		Count(&count).Error
	return count, err
}

// CreateToken menyimpan token baru ke DB.
func CreateToken(staffID, token, ip string) error {
	t := &models.PasswordResetToken{
		StaffID:   staffID,
		Token:     token,
		IPAddress: ip,
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}
	return config.DB.Create(t).Error
}

// FindValidToken mencari token yang valid: ada, belum expired, belum dipakai.
func FindValidToken(token string) (*models.PasswordResetToken, error) {
	var t models.PasswordResetToken
	err := config.DB.Preload("Staff.Role").
		Where("token = ? AND expires_at > ? AND used_at IS NULL", token, time.Now()).
		First(&t).Error
	return &t, err
}

// MarkTokenUsed menandai token sudah dipakai (one-time use).
func MarkTokenUsed(token string) error {
	now := time.Now()
	return config.DB.Model(&models.PasswordResetToken{}).
		Where("token = ?", token).
		Update("used_at", now).Error
}

// UpdateStaffPassword memperbarui password hash staff berdasarkan ID.
func UpdateStaffPassword(staffID, hashedPassword string) error {
	return config.DB.Model(&models.Staff{}).
		Where("id = ?", staffID).
		Update("password_hash", hashedPassword).Error
}
