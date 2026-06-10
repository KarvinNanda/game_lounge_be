package controller

import (
	"net/http"

	"game_lounge_be/modules/event_booking/dto"
	"game_lounge_be/modules/event_booking/service"
	"game_lounge_be/utils"

	"github.com/gin-gonic/gin"
)

// ── Event Booking ─────────────────────────────────────────────────────────────

// GetAll mengambil list event booking dengan filter & pagination.
func GetAll(c *gin.Context) {
	var filter dto.EventBookingFilter
	c.ShouldBindQuery(&filter)
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PerPage < 1 {
		filter.PerPage = 20
	}

	bookings, total, err := service.GetAll(filter)
	if err != nil {
		utils.ResponseError(c, http.StatusInternalServerError, "Gagal mengambil data")
		return
	}

	totalPage := int(total) / filter.PerPage
	if int(total)%filter.PerPage != 0 {
		totalPage++
	}
	utils.ResponseSuccessPaginate(c, http.StatusOK, "OK", bookings, utils.Meta{
		Page: filter.Page, PerPage: filter.PerPage, Total: total, TotalPage: totalPage,
	})
}

// GetByID mengambil detail 1 event booking.
func GetByID(c *gin.Context) {
	b, err := service.GetByID(c.Param("id"))
	if err != nil {
		utils.ResponseError(c, http.StatusNotFound, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "OK", b)
}

// Create membuat event booking baru.
func Create(c *gin.Context) {
	var req dto.CreateEventBookingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseValidationError(c, http.StatusBadRequest, "Validasi gagal", err.Error())
		return
	}
	actorName := c.GetString("staff_username")
	b, err := service.Create(req, actorName)
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusCreated, "Event booking berhasil dibuat", b)
}

// Cancel membatalkan event booking dengan alasan.
func Cancel(c *gin.Context) {
	var req dto.CancelEventBookingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseValidationError(c, http.StatusBadRequest, "Validasi gagal", err.Error())
		return
	}
	actorName := c.GetString("staff_username")
	b, err := service.Cancel(c.Param("id"), req, actorName)
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "Event booking berhasil dibatalkan", b)
}

// GetForDashboard mengambil event booking aktif untuk grid pada store & tanggal tertentu.
func GetForDashboard(c *gin.Context) {
	storeID := c.Query("store_id")
	date := c.Query("date")
	if storeID == "" || date == "" {
		utils.ResponseError(c, http.StatusBadRequest, "store_id dan date wajib diisi")
		return
	}
	bookings, err := service.GetForDashboard(storeID, date)
	if err != nil {
		utils.ResponseError(c, http.StatusInternalServerError, "Gagal mengambil data")
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "OK", bookings)
}

// ── Event Pricing ─────────────────────────────────────────────────────────────

// GetEventPrice mengambil harga event untuk store tertentu.
func GetEventPrice(c *gin.Context) {
	price, err := service.GetEventPrice(c.Param("id"))
	if err != nil {
		utils.ResponseError(c, http.StatusInternalServerError, "Gagal mengambil harga event")
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "OK", price)
}

// UpsertEventPrice menyimpan (create/update) harga event untuk store.
func UpsertEventPrice(c *gin.Context) {
	var req struct {
		PricePerDay float64 `json:"price_per_day" binding:"required,min=0"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseValidationError(c, http.StatusBadRequest, "Validasi gagal", err.Error())
		return
	}
	actorName := c.GetString("staff_username")
	price, err := service.UpsertEventPrice(c.Param("id"), req.PricePerDay, actorName)
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "Harga event berhasil disimpan", price)
}

// PreviewPrice menghitung estimasi harga event sebelum booking dibuat.
func PreviewPrice(c *gin.Context) {
	storeID   := c.Query("store_id")
	startTime := c.Query("start_time")
	endTime   := c.Query("end_time")
	if storeID == "" || startTime == "" || endTime == "" {
		utils.ResponseError(c, http.StatusBadRequest, "store_id, start_time, dan end_time wajib diisi")
		return
	}
	result, err := service.PreviewPrice(storeID, startTime, endTime)
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "OK", result)
}
