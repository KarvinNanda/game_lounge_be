package controller

import (
	"net/http"
	"time"

	"game_lounge_be/config"
	"game_lounge_be/models"
	"game_lounge_be/utils"

	"github.com/gin-gonic/gin"
)

// GetMyVouchers mengambil voucher yang tersedia untuk customer.
// Logika: ambil semua voucher aktif & valid, lalu exclude yang sudah dipakai customer ini.
// Sama persis dengan query di admin booking page.
func GetMyVouchers(c *gin.Context) {
	customerID := c.GetString("customer_id")
	storeID := c.Query("store_id") // opsional

	// Voucher hanya untuk member (dicek juga di ValidateVoucher saat dipakai).
	var customer models.Customer
	if err := config.DB.Select("type").Where("id = ?", customerID).First(&customer).Error; err != nil || customer.Type != "member" {
		utils.ResponseSuccess(c, http.StatusOK, "OK", []any{})
		return
	}

	type VoucherResult struct {
		ID            string   `json:"voucher_id"`
		Code          string   `json:"code"`
		Name          string   `json:"name"`
		Description   *string  `json:"description"`
		DiscountType  string   `json:"discount_type"`
		DiscountValue float64  `json:"discount_value"`
		MaxDiscount   *float64 `json:"max_discount"`
		MinPurchase   float64  `json:"min_purchase"`
		EndDate       *string  `json:"end_date"`
		IsAllStores   bool     `json:"is_all_stores"`
	}

	// Tanggal hari ini (WIB) dari Go — CURDATE() MySQL memakai zona server DB.
	loc, _ := time.LoadLocation("Asia/Jakarta")
	today := time.Now().In(loc).Format("2006-01-02")

	query := config.DB.Table("vouchers v").
		Select(`v.id, v.code, v.name, v.description,
                v.discount_type, v.discount_value,
                v.max_discount, v.min_purchase,
                v.end_date, v.is_all_stores`).
		Where(`v.deleted_at IS NULL
            AND v.is_active = true
            AND v.start_date <= ?
            AND (v.end_date IS NULL OR v.end_date >= ?)
            AND (v.type = 'booking' OR v.type = 'both')
            AND v.id NOT IN (
                SELECT vu.voucher_id FROM voucher_usages vu
                WHERE vu.customer_id = ?
            )`, today, today, customerID)

	if storeID != "" {
		query = query.Where(`(
            v.is_all_stores = true
            OR v.id IN (
                SELECT vs.voucher_id FROM voucher_stores vs
                WHERE vs.store_id = ?
            )
        )`, storeID)
	}

	query = query.Order("v.created_at DESC")

	var vouchers []VoucherResult
	query.Find(&vouchers)

	utils.ResponseSuccess(c, http.StatusOK, "OK", vouchers)
}
