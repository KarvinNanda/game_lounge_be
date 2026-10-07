package controller

import (
	"net/http"

	"game_lounge_be/config"
	"game_lounge_be/models"
	"game_lounge_be/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Login memproses login customer (cek tabel customers, bukan staffs).
func Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseValidationError(c, http.StatusBadRequest, "Validasi gagal", err.Error())
		return
	}

	var customer models.Customer
	if err := config.DB.Where(
		"email = ? AND deleted_at IS NULL AND status = 'active'",
		req.Email,
	).First(&customer).Error; err != nil {
		utils.ResponseError(c, http.StatusUnauthorized, "Email atau password salah")
		return
	}

	// PasswordHash adalah *string — cek nil sebelum dereference
	if customer.PasswordHash == nil || !utils.CheckPassword(req.Password, *customer.PasswordHash) {
		utils.ResponseError(c, http.StatusUnauthorized, "Email atau password salah")
		return
	}

	token, err := utils.GenerateCustomerJWT(customer.ID, customer.Type, customer.TokenVersion)
	if err != nil {
		utils.ResponseError(c, http.StatusInternalServerError, "Gagal generate token")
		return
	}

	// Token dikirim via httpOnly cookie, TIDAK di response body —
	// JavaScript tidak boleh menyentuh token (mitigasi XSS).
	utils.SetAuthCookie(c, utils.CustomerCookieName, token, utils.CustomerTokenMaxAge(), utils.CustomerCookiePath)
	utils.ResponseSuccess(c, http.StatusOK, "Login berhasil", gin.H{
		"customer": customer,
	})
}

// Me mengambil profil customer yang sedang login.
func Me(c *gin.Context) {
	customerID := c.GetString("customer_id")
	var customer models.Customer
	if err := config.DB.
		Preload("FavoriteRoomTypes.RoomTemplate").
		Where("id = ? AND deleted_at IS NULL", customerID).
		First(&customer).Error; err != nil {
		utils.ResponseError(c, http.StatusNotFound, "Customer tidak ditemukan")
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "OK", customer)
}

// Logout menghapus httpOnly cookie customer di browser.
func Logout(c *gin.Context) {
	// Naikkan token_version → JWT ini (dan salinannya) langsung tidak berlaku.
	if err := bumpCustomerTokenVersion(config.DB, c.GetString("customer_id")); err != nil {
		utils.ResponseError(c, http.StatusInternalServerError, "Gagal logout")
		return
	}
	utils.ClearAuthCookie(c, utils.CustomerCookieName, utils.CustomerCookiePath)
	utils.ResponseSuccess(c, http.StatusOK, "Logout berhasil", nil)
}

// UpdateProfile memperbarui nama dan nomor WhatsApp customer.
// Email tidak bisa diubah karena dipakai sebagai identifier login.
func UpdateProfile(c *gin.Context) {
	customerID := c.GetString("customer_id")

	var req struct {
		Name     string `json:"name" binding:"required,min=2,max=150"`
		Whatsapp string `json:"whatsapp" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseValidationError(c, http.StatusBadRequest, "Validasi gagal", err.Error())
		return
	}

	if err := config.DB.Model(&models.Customer{}).
		Where("id = ?", customerID).
		Updates(map[string]interface{}{
			"name":     req.Name,
			"whatsapp": req.Whatsapp,
		}).Error; err != nil {
		utils.ResponseError(c, http.StatusInternalServerError, "Gagal menyimpan profil")
		return
	}

	var customer models.Customer
	config.DB.Where("id = ?", customerID).First(&customer)
	utils.ResponseSuccess(c, http.StatusOK, "Profil berhasil diperbarui", customer)
}

// ChangePassword memperbarui password customer dengan validasi password lama.
func ChangePassword(c *gin.Context) {
	customerID := c.GetString("customer_id")

	var req struct {
		OldPassword     string `json:"old_password" binding:"required"`
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

	var customer models.Customer
	if err := config.DB.Where("id = ?", customerID).First(&customer).Error; err != nil {
		utils.ResponseError(c, http.StatusNotFound, "Customer tidak ditemukan")
		return
	}

	if customer.PasswordHash == nil || !utils.CheckPassword(req.OldPassword, *customer.PasswordHash) {
		// 400, bukan 401: 401 berarti sesi invalid dan membuat FE me-logout customer.
		utils.ResponseError(c, http.StatusBadRequest, "Password lama tidak sesuai")
		return
	}

	newHash, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		utils.ResponseError(c, http.StatusInternalServerError, "Gagal memproses password")
		return
	}

	// Ganti password + akhiri semua sesi lain (token_version naik).
	if err := config.DB.Model(&models.Customer{}).
		Where("id = ?", customerID).
		Updates(map[string]interface{}{
			"password_hash": newHash,
			"token_version": gorm.Expr("token_version + 1"),
		}).Error; err != nil {
		utils.ResponseError(c, http.StatusInternalServerError, "Gagal menyimpan password baru")
		return
	}

	// Sesi yang sedang dipakai tetap login: terbitkan token baru dengan versi terbaru.
	token, err := utils.GenerateCustomerJWT(customer.ID, customer.Type, customer.TokenVersion+1)
	if err == nil {
		utils.SetAuthCookie(c, utils.CustomerCookieName, token, utils.CustomerTokenMaxAge(), utils.CustomerCookiePath)
	}

	utils.ResponseSuccess(c, http.StatusOK, "Password berhasil diubah", nil)
}

// bumpCustomerTokenVersion → semua JWT customer yang sudah terbit ditolak CustomerAuth.
func bumpCustomerTokenVersion(db *gorm.DB, customerID string) error {
	return db.Model(&models.Customer{}).Where("id = ?", customerID).
		Update("token_version", gorm.Expr("token_version + 1")).Error
}
