package controller

import (
	"net/http"
	"strconv"

	"game_lounge_be/modules/pricing/dto"
	"game_lounge_be/modules/pricing/repository"
	"game_lounge_be/modules/pricing/service"
	"game_lounge_be/utils"

	"github.com/gin-gonic/gin"
)

// ── Pricing Config ────────────────────────────────────────────

// GetAll mengambil semua store dengan status pricing masing-masing.
func GetAll(c *gin.Context) {
	var filter dto.PricingListFilter
	c.ShouldBindQuery(&filter)
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PerPage < 1 {
		filter.PerPage = 20
	}

	result, total, err := service.GetAllPricingsWithStores(filter)
	if err != nil {
		utils.ResponseError(c, http.StatusInternalServerError, "Gagal mengambil data pricing")
		return
	}

	totalPage := int(total) / filter.PerPage
	if int(total)%filter.PerPage != 0 {
		totalPage++
	}

	utils.ResponseSuccessPaginate(c, http.StatusOK, "OK", result, utils.Meta{
		Page: filter.Page, PerPage: filter.PerPage, Total: total, TotalPage: totalPage,
	})
}

// GetByStore mengambil config pricing lengkap untuk 1 store.
func GetByStore(c *gin.Context) {
	storeID := c.Param("store_id")
	pricing, err := service.GetPricingByStore(storeID)
	if err != nil {
		utils.ResponseError(c, http.StatusNotFound, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "OK", pricing)
}

// Create membuat pricing config baru untuk store.
func Create(c *gin.Context) {
	var req dto.CreatePricingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseValidationError(c, http.StatusBadRequest, "Validasi gagal", err.Error())
		return
	}
	actorName := c.GetString("staff_username")
	pricing, err := service.CreatePricing(req, actorName)
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusCreated, "Pricing berhasil dibuat", pricing)
}

// UpdateConfig update config utama pricing (edge case, toggle HH/mixed).
func UpdateConfig(c *gin.Context) {
	storeID := c.Param("store_id")
	var req dto.UpdatePricingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseValidationError(c, http.StatusBadRequest, "Validasi gagal", err.Error())
		return
	}
	actorName := c.GetString("staff_username")
	pricing, err := service.UpdatePricingConfig(storeID, req, actorName)
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "Config pricing berhasil diupdate", pricing)
}

// Delete soft-delete pricing config.
func Delete(c *gin.Context) {
	storeID := c.Param("store_id")
	actorName := c.GetString("staff_username")
	if err := service.DeletePricing(storeID, actorName); err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "Pricing berhasil dihapus", nil)
}

// ── Happy Hour Schedules ──────────────────────────────────────

// AddSchedule menambahkan rentang waktu happy hour.
func AddSchedule(c *gin.Context) {
	storeID := c.Param("store_id")
	var req dto.AddScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseValidationError(c, http.StatusBadRequest, "Validasi gagal", err.Error())
		return
	}
	actorName := c.GetString("staff_username")
	s, err := service.AddSchedule(storeID, req, actorName)
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusCreated, "Jadwal happy hour berhasil ditambahkan", s)
}

// DeleteSchedule menghapus jadwal happy hour.
func DeleteSchedule(c *gin.Context) {
	storeID := c.Param("store_id")
	id, _ := strconv.Atoi(c.Param("id"))
	actorName := c.GetString("staff_username")
	if err := service.DeleteSchedule(uint(id), storeID, actorName); err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "Jadwal berhasil dihapus", nil)
}

// ── Happy Hour Prices ─────────────────────────────────────────

// GetHappyHourPrices mengambil harga HH untuk 1 store.
func GetHappyHourPrices(c *gin.Context) {
	storeID := c.Param("store_id")
	prices, err := repository.FindHappyHourPricesByStore(storeID)
	if err != nil {
		utils.ResponseSuccess(c, http.StatusOK, "OK", []interface{}{})
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "OK", prices)
}

// UpsertHappyHourPrices bulk insert-or-update harga HH.
func UpsertHappyHourPrices(c *gin.Context) {
	storeID := c.Param("store_id")
	var req dto.BulkUpsertHappyHourPricesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseValidationError(c, http.StatusBadRequest, "Validasi gagal", err.Error())
		return
	}
	actorName := c.GetString("staff_username")
	prices, err := service.BulkUpsertHappyHourPrices(storeID, req, actorName)
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "Harga Happy Hour berhasil diupdate", prices)
}

// ── Package Prices ────────────────────────────────────────────

// GetPackagePrices mengambil harga paket untuk 1 store.
func GetPackagePrices(c *gin.Context) {
	storeID := c.Param("store_id")
	prices, err := repository.FindPackagePricesByStore(storeID)
	if err != nil {
		utils.ResponseSuccess(c, http.StatusOK, "OK", []interface{}{})
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "OK", prices)
}

// UpsertPackagePrices bulk insert-or-update harga paket.
func UpsertPackagePrices(c *gin.Context) {
	storeID := c.Param("store_id")
	var req dto.BulkUpsertPackagePricesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseValidationError(c, http.StatusBadRequest, "Validasi gagal", err.Error())
		return
	}
	actorName := c.GetString("staff_username")
	prices, err := service.BulkUpsertPackagePrices(storeID, req, actorName)
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "Harga paket berhasil diupdate", prices)
}

// DeletePackagePrice menghapus 1 paket harga.
func DeletePackagePrice(c *gin.Context) {
	storeID := c.Param("store_id")
	id, _ := strconv.Atoi(c.Param("id"))
	actorName := c.GetString("staff_username")
	if err := service.DeletePackagePrice(uint(id), storeID, actorName); err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "Paket berhasil dihapus", nil)
}

// ── Flash Sales ───────────────────────────────────────────────

// GetFlashSales mengambil semua flash sale untuk 1 store.
func GetFlashSales(c *gin.Context) {
	storeID := c.Param("store_id")
	sales, err := repository.FindFlashSalesByStore(storeID)
	if err != nil {
		utils.ResponseError(c, http.StatusInternalServerError, "Gagal mengambil flash sale")
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "OK", sales)
}

// CreateFlashSale membuat flash sale baru.
func CreateFlashSale(c *gin.Context) {
	storeID := c.Param("store_id")
	var req dto.CreateFlashSaleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseValidationError(c, http.StatusBadRequest, "Validasi gagal", err.Error())
		return
	}
	actorName := c.GetString("staff_username")
	sale, err := service.CreateFlashSale(storeID, req, actorName)
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusCreated, "Flash sale berhasil dibuat", sale)
}

// UpdateFlashSale mengupdate flash sale.
func UpdateFlashSale(c *gin.Context) {
	id := c.Param("id")
	var req dto.UpdateFlashSaleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseValidationError(c, http.StatusBadRequest, "Validasi gagal", err.Error())
		return
	}
	actorName := c.GetString("staff_username")
	sale, err := service.UpdateFlashSale(id, req, actorName)
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "Flash sale berhasil diupdate", sale)
}

// DeleteFlashSale menghapus flash sale.
func DeleteFlashSale(c *gin.Context) {
	id := c.Param("id")
	actorName := c.GetString("staff_username")
	if err := service.DeleteFlashSale(id, actorName); err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "Flash sale berhasil dihapus", nil)
}

// ── Calculator ────────────────────────────────────────────────

// Calculate menghitung estimasi harga booking.
func Calculate(c *gin.Context) {
	var req dto.CalculatePriceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseValidationError(c, http.StatusBadRequest, "Validasi gagal", err.Error())
		return
	}
	result, err := service.CalculatePrice(req)
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "OK", result)
}
