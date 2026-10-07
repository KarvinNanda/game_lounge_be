package controller

import (
	"net/http"

	"game_lounge_be/middleware"
	"game_lounge_be/models"
	"game_lounge_be/modules/auth/dto"
	"game_lounge_be/modules/auth/repository"
	"game_lounge_be/modules/auth/service"
	"game_lounge_be/utils"

	"github.com/gin-gonic/gin"
)

// Login memvalidasi kredensial staff lalu memasang httpOnly cookie.
// Token TIDAK dikirim di response body — JavaScript tidak boleh menyentuh token
// (mitigasi pencurian token via XSS).
func Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseValidationError(c, http.StatusBadRequest, "Validasi gagal", err.Error())
		return
	}

	resp, err := service.Login(req)
	if err != nil {
		utils.ResponseError(c, http.StatusUnauthorized, err.Error())
		return
	}

	utils.SetAuthCookie(c, utils.StaffCookieName, resp.Token, utils.StaffTokenMaxAge(), utils.StaffCookiePath)
	utils.ResponseSuccess(c, http.StatusOK, "Login berhasil", gin.H{
		"staff": resp.Staff,
	})
}

func Me(c *gin.Context) {
	staffID := c.GetString("staff_id")
	staff, err := repository.FindStaffByID(staffID)
	if err != nil {
		utils.ResponseError(c, http.StatusNotFound, "Staff tidak ditemukan")
		return
	}
	all, ids := middleware.StoreScope(c)
	utils.ResponseSuccess(c, http.StatusOK, "OK", meResponse{
		Staff:       staff,
		StoreAccess: storeAccess{AllStores: all, StoreIDs: ids},
	})
}

// meResponse = field staff yang sudah ada (di-embed, tetap flat di JSON) +
// store_access, yaitu aturan cabang yang benar-benar dipakai backend.
type meResponse struct {
	*models.Staff
	StoreAccess storeAccess `json:"store_access"`
}

type storeAccess struct {
	AllStores bool     `json:"all_stores"` // Super Admin atau is_all_stores
	StoreIDs  []string `json:"store_ids"`  // dipakai jika all_stores = false
}

// Logout menghapus httpOnly cookie di browser.
func Logout(c *gin.Context) {
	// Naikkan token_version: JWT ini (dan salinannya, mis. cookie yang dicuri)
	// langsung tidak berlaku, bukan hanya cookie di browser yang dihapus.
	if err := repository.BumpTokenVersion(c.GetString("staff_id")); err != nil {
		utils.ResponseError(c, http.StatusInternalServerError, "Gagal logout")
		return
	}
	utils.ClearAuthCookie(c, utils.StaffCookieName, utils.StaffCookiePath)
	utils.ResponseSuccess(c, http.StatusOK, "Logout berhasil", nil)
}
