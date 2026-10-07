package controller

import (
	"net/http"

	"game_lounge_be/modules/admin_recovery/service"
	"game_lounge_be/utils"

	"github.com/gin-gonic/gin"
)

// Request menerima email dan memproses permintaan reset password.
// SELALU return response yang sama (security: tidak bocorkan info email).
func Request(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		// Tetap response sama meski validasi gagal — tidak bocorkan info apapun
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Jika email terdaftar, link reset password akan dikirim.",
		})
		return
	}

	// ClientIP() hanya memakai X-Forwarded-For dari TRUSTED_PROXIES (router.go).
	// Jangan baca header itu langsung: client bisa memalsukannya untuk bypass limit per IP.
	ip := c.ClientIP()

	// Proses request secara async — response tidak menunggu hasil proses
	utils.SafeGo(func() { service.RequestReset(req.Email, ip) })

	// SELALU return response yang sama — penyerang tidak tahu apakah email valid
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Jika email terdaftar, link reset password akan dikirim.",
	})
}

// Validate mengecek apakah token masih valid sebelum FE tampilkan form reset.
func Validate(c *gin.Context) {
	token := c.Param("token")
	if err := service.ValidateToken(token); err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "Token valid", nil)
}

// Reset memperbarui password menggunakan token yang valid.
func Reset(c *gin.Context) {
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

	if err := service.ResetPassword(token, req.NewPassword); err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "Password berhasil diperbarui. Silakan login.", nil)
}
