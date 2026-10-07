package repository

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"game_lounge_be/config"
	"game_lounge_be/models"
	"game_lounge_be/utils"

	"gorm.io/gorm"
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

// CreateToken menyimpan HASH token ke DB (token mentah hanya ada di email).
func CreateToken(staffID, token, ip string) error {
	t := &models.PasswordResetToken{
		StaffID:   staffID,
		Token:     utils.HashToken(token),
		IPAddress: ip,
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}
	return config.DB.Create(t).Error
}

// FindValidToken mencari token yang valid: ada, belum expired, belum dipakai.
func FindValidToken(token string) (*models.PasswordResetToken, error) {
	var t models.PasswordResetToken
	err := config.DB.Preload("Staff.Role").
		Where("token = ? AND expires_at > ? AND used_at IS NULL", utils.HashToken(token), time.Now()).
		First(&t).Error
	return &t, err
}

// ErrTokenUsed: token sudah dipakai/expired di antara validasi dan konsumsi.
var ErrTokenUsed = errors.New("token sudah dipakai atau kadaluwarsa")

// ConsumeTokenAndSetPassword menandai token terpakai DAN mengganti password dalam
// 1 transaction. Token diklaim dengan UPDATE bersyarat: 2 request bersamaan
// dengan token yang sama → hanya 1 yang mendapat RowsAffected == 1.
func ConsumeTokenAndSetPassword(token, staffID, hashedPassword string) error {
	return config.DB.Transaction(func(tx *gorm.DB) error {
		claim := tx.Model(&models.PasswordResetToken{}).
			Where("token = ? AND used_at IS NULL AND expires_at > ?", utils.HashToken(token), time.Now()).
			Update("used_at", time.Now())
		if claim.Error != nil {
			return claim.Error
		}
		if claim.RowsAffected != 1 {
			return ErrTokenUsed
		}
		return tx.Model(&models.Staff{}).
			Where("id = ?", staffID).
			Updates(map[string]interface{}{
				"password_hash": hashedPassword,
				"token_version": gorm.Expr("token_version + 1"), // akhiri sesi lama
			}).Error
	})
}
