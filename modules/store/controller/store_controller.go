package controller

import (
	"net/http"
	"time"

	"game_lounge_be/config"
	"game_lounge_be/models"
	"game_lounge_be/modules/store/dto"
	"game_lounge_be/modules/store/service"
	"game_lounge_be/utils"

	"github.com/gin-gonic/gin"
)

func GetAll(c *gin.Context) {
	var filter dto.StoreFilter
	c.ShouldBindQuery(&filter)
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PerPage < 1 {
		filter.PerPage = 10
	}

	stores, total, stats, err := service.GetAllStores(filter)
	if err != nil {
		utils.ResponseError(c, http.StatusInternalServerError, "Gagal mengambil data store")
		return
	}

	totalPage := int(total) / filter.PerPage
	if int(total)%filter.PerPage != 0 {
		totalPage++
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true, "message": "OK",
		"data":  stores,
		"meta":  utils.Meta{Page: filter.Page, PerPage: filter.PerPage, Total: total, TotalPage: totalPage},
		"stats": stats,
	})
}

func GetByID(c *gin.Context) {
	id := c.Param("id")
	store, err := service.GetStoreByID(id)
	if err != nil {
		utils.ResponseError(c, http.StatusNotFound, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "OK", store)
}

func Create(c *gin.Context) {
	var req dto.CreateStoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseValidationError(c, http.StatusBadRequest, "Validasi gagal", err.Error())
		return
	}
	actorName := c.GetString("staff_username")
	store, err := service.CreateStore(req, actorName)
	if err != nil {
		utils.ResponseError(c, http.StatusInternalServerError, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusCreated, "Store berhasil dibuat", store)
}

func Update(c *gin.Context) {
	id := c.Param("id")
	var req dto.UpdateStoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseValidationError(c, http.StatusBadRequest, "Validasi gagal", err.Error())
		return
	}
	actorName := c.GetString("staff_username")
	store, err := service.UpdateStore(id, req, actorName)
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "Store berhasil diupdate", store)
}

func Delete(c *gin.Context) {
	id := c.Param("id")
	actorName := c.GetString("staff_username")
	if err := service.DeleteStore(id, actorName); err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "Store berhasil dihapus", nil)
}

// GetOperatingHours mengembalikan jam operasional efektif store pada tanggal tertentu.
// Query params: store_id (required), date (required, format YYYY-MM-DD)
// Dipakai oleh FE booking untuk set grid jam + tampilkan warning holiday.
func GetOperatingHours(c *gin.Context) {
	storeID := c.Query("store_id")
	dateStr := c.Query("date")

	if storeID == "" || dateStr == "" {
		utils.ResponseError(c, http.StatusBadRequest, "store_id dan date wajib diisi")
		return
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, "format date tidak valid (YYYY-MM-DD)")
		return
	}

	result, err := service.GetEffectiveOperatingHours(storeID, date)
	if err != nil {
		utils.ResponseError(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.ResponseSuccess(c, http.StatusOK, "OK", result)
}

// ToggleRoomActive mengaktifkan atau menonaktifkan satu ruangan dalam sebuah store.
// Hanya mengubah is_active, tidak mempengaruhi data booking yang sudah ada.
// PATCH /store-rooms/:id/toggle
// Body: { "is_active": true/false }
func ToggleRoomActive(c *gin.Context) {
	roomID := c.Param("id")

	var req struct {
		IsActive bool `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseError(c, http.StatusBadRequest, "Payload tidak valid")
		return
	}

	actorName := c.GetString("staff_username")

	var room models.StoreRoom
	if err := config.DB.Where("id = ? AND deleted_at IS NULL", roomID).First(&room).Error; err != nil {
		utils.ResponseError(c, http.StatusNotFound, "Ruangan tidak ditemukan")
		return
	}

	if err := config.DB.Model(&room).Updates(map[string]interface{}{
		"is_active":  req.IsActive,
		"updated_by": actorName,
	}).Error; err != nil {
		utils.ResponseError(c, http.StatusInternalServerError, "Gagal mengubah status ruangan")
		return
	}

	// Reload dengan relasi untuk response lengkap
	config.DB.Preload("RoomTemplate").Where("id = ?", roomID).First(&room)

	status := "dinonaktifkan"
	if req.IsActive {
		status = "diaktifkan"
	}
	utils.ResponseSuccess(c, http.StatusOK, "Ruangan berhasil "+status, room)
}
