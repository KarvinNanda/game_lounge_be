package controller

import (
	"net/http"

	"game_lounge_be/modules/booking/dto"
	"game_lounge_be/modules/booking/service"
	"game_lounge_be/utils"

	"github.com/gin-gonic/gin"
)

// GetAll mengambil list booking dengan filter & pagination.
func GetAll(c *gin.Context) {
	var filter dto.BookingFilter
	c.ShouldBindQuery(&filter)
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PerPage < 1 {
		filter.PerPage = 20
	}

	bookings, total, err := service.GetAllBookings(filter)
	if err != nil {
		utils.ResponseError(c, http.StatusInternalServerError, "Gagal mengambil data booking")
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

// GetByID mengambil detail 1 booking.
func GetByID(c *gin.Context) {
	id := c.Param("id")
	booking, err := service.GetBookingByID(id)
	if err != nil {
		utils.ResponseError(c, http.StatusNotFound, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "OK", booking)
}

// Create membuat booking baru.
func Create(c *gin.Context) {
	var req dto.CreateBookingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseValidationError(c, http.StatusBadRequest, "Validasi gagal", err.Error())
		return
	}
	actorName := c.GetString("staff_username")
	booking, err := service.CreateBooking(req, actorName)
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusCreated, "Booking berhasil dibuat", booking)
}

// Cancel membatalkan booking dengan alasan.
func Cancel(c *gin.Context) {
	id := c.Param("id")
	var req dto.CancelBookingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseValidationError(c, http.StatusBadRequest, "Validasi gagal", err.Error())
		return
	}
	actorName := c.GetString("staff_username")
	booking, err := service.CancelBooking(id, req, actorName)
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "Booking berhasil dibatalkan", booking)
}

// Complete menandai booking sebagai selesai secara manual.
func Complete(c *gin.Context) {
	id := c.Param("id")
	actorName := c.GetString("staff_username")
	booking, err := service.CompleteBooking(id, actorName)
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "Booking berhasil diselesaikan", booking)
}

// GetDashboard mengambil data kalender grid per store per tanggal.
func GetDashboard(c *gin.Context) {
	var filter dto.DashboardFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		utils.ResponseValidationError(c, http.StatusBadRequest, "Parameter tidak lengkap", err.Error())
		return
	}
	data, err := service.GetDashboard(filter)
	if err != nil {
		utils.ResponseError(c, http.StatusInternalServerError, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "OK", data)
}

// GetSessionsEndingSoon mengambil sesi yang akan berakhir dalam 30 menit.
func GetSessionsEndingSoon(c *gin.Context) {
	storeID := c.Query("store_id")
	sessions, err := service.GetSessionsEndingSoon(storeID)
	if err != nil {
		utils.ResponseError(c, http.StatusInternalServerError, "Gagal mengambil data sesi")
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "OK", sessions)
}

// GetAvailableCredits mengambil play credits tersedia untuk customer di store tertentu.
func GetAvailableCredits(c *gin.Context) {
	customerID := c.Query("customer_id")
	storeID := c.Query("store_id")
	if customerID == "" || storeID == "" {
		utils.ResponseError(c, http.StatusBadRequest, "customer_id dan store_id wajib diisi")
		return
	}
	// Default durasi minimal 0.5 jam untuk filter credits yang tersedia
	credits, err := service.GetAvailableCredits(customerID, storeID, 0.5)
	if err != nil {
		utils.ResponseError(c, http.StatusInternalServerError, "Gagal mengambil data credits")
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "OK", credits)
}
