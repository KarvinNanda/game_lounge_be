package controller

import (
	"net/http"

	"game_lounge_be/modules/sales/dto"
	"game_lounge_be/modules/sales/service"
	"game_lounge_be/utils"

	"github.com/gin-gonic/gin"
)

// GetSummary mengambil semua data untuk dashboard utama sales.
// Query params: period, store_id, date_from, date_to (jika period=custom)
func GetSummary(c *gin.Context) {
	var filter dto.SalesFilter
	c.ShouldBindQuery(&filter)

	summary, err := service.GetSalesSummary(filter)
	if err != nil {
		utils.ResponseError(c, http.StatusInternalServerError, "Gagal mengambil data sales")
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "OK", summary)
}

// GetTrend mengambil data trend untuk grafik garis.
// Query params: period, store_id, granularity (daily|weekly|monthly), date_from, date_to
func GetTrend(c *gin.Context) {
	var filter dto.SalesFilter
	c.ShouldBindQuery(&filter)

	trend, err := service.GetSalesTrend(filter)
	if err != nil {
		utils.ResponseError(c, http.StatusInternalServerError, "Gagal mengambil data trend")
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "OK", trend)
}

// GetTransactions mengambil daftar transaksi (untuk modal detail & export).
// Query params: period, store_id, type (all|booking|play_credits), date_from, date_to, page, per_page
func GetTransactions(c *gin.Context) {
	var filter dto.SalesFilter
	c.ShouldBindQuery(&filter)
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PerPage < 1 {
		filter.PerPage = 20
	}

	items, total, err := service.GetTransactions(filter)
	if err != nil {
		utils.ResponseError(c, http.StatusInternalServerError, "Gagal mengambil transaksi")
		return
	}

	totalPage := int(total) / filter.PerPage
	if int(total)%filter.PerPage != 0 {
		totalPage++
	}

	utils.ResponseSuccessPaginate(c, http.StatusOK, "OK", items, utils.Meta{
		Page: filter.Page, PerPage: filter.PerPage, Total: total, TotalPage: totalPage,
	})
}
