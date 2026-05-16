package controller

import (
	"net/http"

	"game_lounge_be/modules/voucher/dto"
	"game_lounge_be/modules/voucher/service"
	"game_lounge_be/utils"

	"github.com/gin-gonic/gin"
)

// GetAll mengambil list voucher dengan filter, pagination, dan stats.
func GetAll(c *gin.Context) {
	var filter dto.VoucherFilter
	c.ShouldBindQuery(&filter)
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PerPage < 1 {
		filter.PerPage = 10
	}

	vouchers, total, stats, err := service.GetAllVouchers(filter)
	if err != nil {
		utils.ResponseError(c, http.StatusInternalServerError, "Gagal mengambil data voucher")
		return
	}

	totalPage := int(total) / filter.PerPage
	if int(total)%filter.PerPage != 0 {
		totalPage++
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "OK",
		"data":    vouchers,
		"meta": utils.Meta{
			Page: filter.Page, PerPage: filter.PerPage,
			Total: total, TotalPage: totalPage,
		},
		"stats": stats,
	})
}

// GetByID mengambil detail 1 voucher.
func GetByID(c *gin.Context) {
	id := c.Param("id")
	voucher, err := service.GetVoucherByID(id)
	if err != nil {
		utils.ResponseError(c, http.StatusNotFound, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "OK", voucher)
}

// GetCustomerAvailable mengambil voucher yang bisa dipakai oleh customer tertentu
// di store tertentu. Digunakan frontend saat create booking untuk dropdown pilihan voucher.
// Query params: customer_id (required), store_id (required), type (booking|play_credits|both, default: booking)
func GetCustomerAvailable(c *gin.Context) {
	customerID := c.Query("customer_id")
	storeID := c.Query("store_id")
	useType := c.Query("type")

	if customerID == "" || storeID == "" {
		utils.ResponseError(c, http.StatusBadRequest, "customer_id dan store_id wajib diisi")
		return
	}
	if useType == "" {
		useType = "booking"
	}

	vouchers, err := service.GetAvailableVouchersForCustomer(customerID, storeID, useType)
	if err != nil {
		utils.ResponseError(c, http.StatusInternalServerError, "Gagal mengambil voucher")
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "OK", vouchers)
}

// GenerateCode menghasilkan kode voucher dari nama voucher (?name=...).
func GenerateCode(c *gin.Context) {
	name := c.Query("name")
	code := service.GenerateCode(name)
	utils.ResponseSuccess(c, http.StatusOK, "OK", gin.H{"code": code})
}

// Create membuat voucher baru.
func Create(c *gin.Context) {
	var req dto.CreateVoucherRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseValidationError(c, http.StatusBadRequest, "Validasi gagal", err.Error())
		return
	}
	actorName := c.GetString("staff_username")
	voucher, err := service.CreateVoucher(req, actorName)
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	msg := "Voucher berhasil dibuat"
	if req.SendChannel != "" {
		msg += " dan sedang dikirim ke member"
	}
	utils.ResponseSuccess(c, http.StatusCreated, msg, voucher)
}

// Update mengupdate voucher.
func Update(c *gin.Context) {
	id := c.Param("id")
	var req dto.UpdateVoucherRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseValidationError(c, http.StatusBadRequest, "Validasi gagal", err.Error())
		return
	}
	actorName := c.GetString("staff_username")
	voucher, err := service.UpdateVoucher(id, req, actorName)
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "Voucher berhasil diupdate", voucher)
}

// Delete soft-delete voucher.
func Delete(c *gin.Context) {
	id := c.Param("id")
	actorName := c.GetString("staff_username")
	if err := service.DeleteVoucher(id, actorName); err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "Voucher berhasil dihapus", nil)
}

// Validate dipakai modul Booking untuk cek voucher sebelum digunakan.
func Validate(c *gin.Context) {
	var req dto.ValidateVoucherRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseValidationError(c, http.StatusBadRequest, "Validasi gagal", err.Error())
		return
	}
	result, err := service.ValidateVoucher(req)
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, result.Message, result)
}
