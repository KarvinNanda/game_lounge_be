package controller

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"game_lounge_be/config"
	"game_lounge_be/models"
	"game_lounge_be/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ForgotPassword menerima email customer dan memproses reset password secara async.
// SELALU return response yang sama — penyerang tidak tahu apakah email valid.
func ForgotPassword(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Jika email terdaftar, link reset password akan dikirim.",
		})
		return
	}

	ip := c.ClientIP()

	utils.SafeGo(func() { processCustomerPasswordReset(req.Email, ip) })

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Jika email terdaftar, link reset password akan dikirim.",
	})
}

// processCustomerPasswordReset adalah logic internal — dijalankan secara async.
func processCustomerPasswordReset(email, ip string) {
	// 1. Rate limiting: max 3 request per jam per IP
	var requestCount int64
	config.DB.Model(&models.CustomerPasswordReset{}).
		Where("ip_address = ? AND created_at > ?", ip, time.Now().Add(-1*time.Hour)).
		Count(&requestCount)
	if requestCount >= 3 {
		return
	}

	// 2. Cari customer dengan email ini (aktif)
	var customer models.Customer
	if err := config.DB.Where(
		"email = ? AND status = 'active' AND deleted_at IS NULL", email,
	).First(&customer).Error; err != nil {
		return
	}

	// 3. Generate token 32-byte random hex
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return
	}
	token := hex.EncodeToString(tokenBytes)

	// 4. Simpan HASH token ke DB (token mentah hanya ada di email)
	reset := &models.CustomerPasswordReset{
		CustomerID: customer.ID,
		Token:      utils.HashToken(token),
		IPAddress:  ip,
		ExpiresAt:  time.Now().Add(15 * time.Minute),
	}
	if err := config.DB.Create(reset).Error; err != nil {
		return
	}

	// 5. Kirim email reset password
	if customer.Email == nil {
		return
	}
	appURL := os.Getenv("APP_URL")
	resetLink := fmt.Sprintf("%s/reset-password/%s", appURL, token)
	subject := "Reset Password — Quantum Game Station"
	body := fmt.Sprintf(`Halo %s,

Kami menerima permintaan reset password untuk akun kamu di Quantum Game Station.

Klik link berikut untuk membuat password baru:
%s

Link ini hanya berlaku selama 15 menit dan hanya bisa digunakan 1 kali.

Jika kamu tidak meminta reset password, abaikan email ini.
Password kamu tidak akan berubah.

Salam,
Tim Quantum Game Station`, customer.Name, resetLink)

	utils.SendEmail(*customer.Email, customer.Name, subject, body)
}

// ValidateResetToken memvalidasi token sebelum FE menampilkan form reset password.
func ValidateResetToken(c *gin.Context) {
	token := c.Param("token")
	var reset models.CustomerPasswordReset
	if err := config.DB.
		Where("token = ? AND expires_at > ? AND used_at IS NULL", utils.HashToken(token), time.Now()).
		First(&reset).Error; err != nil {
		utils.ResponseError(c, http.StatusBadRequest, "Link reset password tidak valid atau sudah kadaluwarsa")
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "Token valid", nil)
}

// ResetPassword memperbarui password customer menggunakan token yang valid.
func ResetPassword(c *gin.Context) {
	token := c.Param("token")
	var req struct {
		NewPassword     string `json:"new_password" binding:"required,min=8"`
		ConfirmPassword string `json:"confirm_password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseValidationError(c, http.StatusBadRequest, "Validasi gagal", err.Error())
		return
	}
	if req.NewPassword != req.ConfirmPassword {
		utils.ResponseError(c, http.StatusBadRequest, "Konfirmasi password tidak cocok")
		return
	}

	// Cari token valid
	var reset models.CustomerPasswordReset
	if err := config.DB.Preload("Customer").
		Where("token = ? AND expires_at > ? AND used_at IS NULL", utils.HashToken(token), time.Now()).
		First(&reset).Error; err != nil {
		utils.ResponseError(c, http.StatusBadRequest, "Link reset password tidak valid atau sudah kadaluwarsa")
		return
	}

	// Hash password baru
	hashedPassword, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		utils.ResponseError(c, http.StatusInternalServerError, "Gagal memproses password")
		return
	}

	// Klaim token + update password dalam 1 transaction (one-time use yang atomik):
	// 2 request bersamaan dengan token yang sama → hanya 1 yang berhasil.
	errTokenUsed := errors.New("token sudah dipakai")
	err = config.DB.Transaction(func(tx *gorm.DB) error {
		claim := tx.Model(&models.CustomerPasswordReset{}).
			Where("id = ? AND used_at IS NULL AND expires_at > ?", reset.ID, time.Now()).
			Update("used_at", time.Now())
		if claim.Error != nil {
			return claim.Error
		}
		if claim.RowsAffected != 1 {
			return errTokenUsed
		}
		return tx.Model(&models.Customer{}).
			Where("id = ?", reset.CustomerID).
			Updates(map[string]interface{}{
				"password_hash": hashedPassword,
				"token_version": gorm.Expr("token_version + 1"), // akhiri sesi lama
			}).Error
	})
	if errors.Is(err, errTokenUsed) {
		utils.ResponseError(c, http.StatusBadRequest, "Link reset password tidak valid atau sudah kadaluwarsa")
		return
	}
	if err != nil {
		utils.ResponseError(c, http.StatusInternalServerError, "Gagal menyimpan password baru")
		return
	}

	utils.ResponseSuccess(c, http.StatusOK, "Password berhasil diperbarui. Silakan login.", nil)
}
