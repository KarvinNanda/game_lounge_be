package controller

import (
	"net/http"
	"strconv"

	"game_lounge_be/modules/banner/dto"
	"game_lounge_be/modules/banner/service"
	"game_lounge_be/utils"

	"github.com/gin-gonic/gin"
)

// GetAllAdmin mengambil semua banner (aktif & nonaktif) — untuk halaman admin.
func GetAllAdmin(c *gin.Context) {
	banners, err := service.GetAllAdmin()
	if err != nil {
		utils.ResponseError(c, http.StatusInternalServerError, "Gagal mengambil data banner")
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "OK", banners)
}

// GetAllPublic mengambil banner aktif saja — public endpoint untuk customer.
func GetAllPublic(c *gin.Context) {
	banners, err := service.GetAllActive()
	if err != nil {
		utils.ResponseError(c, http.StatusInternalServerError, "Gagal mengambil data banner")
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "OK", banners)
}

// GetByID mengambil detail satu banner.
func GetByID(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	banner, err := service.GetByID(uint(id))
	if err != nil {
		utils.ResponseError(c, http.StatusNotFound, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "OK", banner)
}

// Create membuat banner baru.
func Create(c *gin.Context) {
	var req dto.CreateBannerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseValidationError(c, http.StatusBadRequest, "Validasi gagal", err.Error())
		return
	}
	actorName := c.GetString("staff_username")
	banner, err := service.Create(req, actorName)
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusCreated, "Banner berhasil dibuat", banner)
}

// Update memperbarui banner yang sudah ada.
func Update(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req dto.UpdateBannerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseValidationError(c, http.StatusBadRequest, "Validasi gagal", err.Error())
		return
	}
	actorName := c.GetString("staff_username")
	banner, err := service.Update(uint(id), req, actorName)
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "Banner berhasil diperbarui", banner)
}

// Delete soft-delete sebuah banner.
func Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	actorName := c.GetString("staff_username")
	if err := service.Delete(uint(id), actorName); err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "Banner berhasil dihapus", nil)
}

// ToggleActive membalik status aktif/nonaktif banner.
func ToggleActive(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	actorName := c.GetString("staff_username")
	banner, err := service.ToggleActive(uint(id), actorName)
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	status := "dinonaktifkan"
	if banner.IsActive {
		status = "diaktifkan"
	}
	utils.ResponseSuccess(c, http.StatusOK, "Banner berhasil "+status, banner)
}

// Reorder mengupdate urutan tampil banyak banner sekaligus.
func Reorder(c *gin.Context) {
	var req dto.ReorderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseValidationError(c, http.StatusBadRequest, "Validasi gagal", err.Error())
		return
	}
	if err := service.Reorder(req); err != nil {
		utils.ResponseError(c, http.StatusInternalServerError, "Gagal mengubah urutan banner")
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "Urutan banner berhasil diperbarui", nil)
}
