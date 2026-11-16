package controller

import (
	"net/http"
	"time"

	"game_lounge_be/config"
	"game_lounge_be/models"
	customerBookingRepo "game_lounge_be/modules/customer_booking/repository"
	"game_lounge_be/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

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
	// Set only for a voucher reserved by one of the customer's booking holds
	// (include_reserved=true). The voucher returns after ReservedUntil.
	ReservedUntil *time.Time `json:"reserved_until" gorm:"-"`
	HoldID        *string    `json:"hold_id" gorm:"-"`
}

// validVouchersQuery: active, in-date booking vouchers, optionally limited to a store.
func validVouchersQuery(today, storeID string) *gorm.DB {
	query := config.DB.Table("vouchers v").
		Select(`v.id, v.code, v.name, v.description,
                v.discount_type, v.discount_value,
                v.max_discount, v.min_purchase,
                v.end_date, v.is_all_stores`).
		Where(`v.deleted_at IS NULL
            AND v.is_active = true
            AND v.start_date <= ?
            AND (v.end_date IS NULL OR v.end_date >= ?)
            AND (v.type = 'booking' OR v.type = 'both')`, today, today)

	if storeID != "" {
		query = query.Where(`(
            v.is_all_stores = true
            OR v.id IN (
                SELECT vs.voucher_id FROM voucher_stores vs
                WHERE vs.store_id = ?
            )
        )`, storeID)
	}
	return query.Order("v.created_at DESC")
}

// GetMyVouchers mengambil voucher yang tersedia untuk customer.
// Logika: ambil semua voucher aktif & valid, lalu exclude yang sudah dipakai customer ini.
// Sama persis dengan query di admin booking page.
//
// include_reserved=true also returns vouchers held by the customer's booking hold
// (usage without booking), with reserved_until and hold_id, so the app can show
// when the voucher comes back.
func GetMyVouchers(c *gin.Context) {
	customerID := c.GetString("customer_id")
	storeID := c.Query("store_id") // opsional

	// Voucher hanya untuk member (dicek juga di ValidateVoucher saat dipakai).
	var customer models.Customer
	if err := config.DB.Select("type").Where("id = ?", customerID).First(&customer).Error; err != nil || customer.Type != "member" {
		utils.ResponseSuccess(c, http.StatusOK, "OK", []any{})
		return
	}

	// Tanggal hari ini (WIB) dari Go — CURDATE() MySQL memakai zona server DB.
	loc, _ := time.LoadLocation("Asia/Jakarta")
	today := time.Now().In(loc).Format("2006-01-02")

	var vouchers []VoucherResult
	if err := validVouchersQuery(today, storeID).
		Where("v.id NOT IN (SELECT vu.voucher_id FROM voucher_usages vu WHERE vu.customer_id = ?)", customerID).
		Find(&vouchers).Error; err != nil {
		utils.ResponseError(c, http.StatusInternalServerError, "Gagal mengambil voucher")
		return
	}

	if c.Query("include_reserved") == "true" {
		reserved, err := reservedVouchers(customerID, today, storeID)
		if err != nil {
			utils.ResponseError(c, http.StatusInternalServerError, "Gagal mengambil voucher")
			return
		}
		vouchers = append(vouchers, reserved...)
	}

	utils.ResponseSuccess(c, http.StatusOK, "OK", vouchers)
}

// reservedVouchers returns valid vouchers whose usage has no booking yet and is
// tied to one of the customer's booking holds. A reservation without a matching
// hold stays hidden, the same as before include_reserved existed.
func reservedVouchers(customerID, today, storeID string) ([]VoucherResult, error) {
	var vouchers []VoucherResult
	if err := validVouchersQuery(today, storeID).
		Where("v.id IN (SELECT vu.voucher_id FROM voucher_usages vu WHERE vu.customer_id = ? AND vu.booking_id IS NULL)", customerID).
		Find(&vouchers).Error; err != nil {
		return nil, err
	}
	if len(vouchers) == 0 {
		return nil, nil
	}

	var holds []models.BookingHold
	if err := config.DB.Select("id", "price_breakdown", "expires_at").
		Where("customer_id = ?", customerID).Find(&holds).Error; err != nil {
		return nil, err
	}
	holdByVoucher := map[string]models.BookingHold{}
	for _, h := range holds {
		vid := customerBookingRepo.HoldVoucherID(h.PriceBreakdown)
		if prev, ok := holdByVoucher[vid]; vid != "" && (!ok || h.ExpiresAt.After(prev.ExpiresAt)) {
			holdByVoucher[vid] = h
		}
	}

	out := make([]VoucherResult, 0, len(vouchers))
	for _, v := range vouchers {
		h, ok := holdByVoucher[v.ID]
		if !ok {
			continue
		}
		until := customerBookingRepo.VoucherReleaseAt(h.ExpiresAt)
		holdID := h.ID
		v.ReservedUntil, v.HoldID = &until, &holdID
		out = append(out, v)
	}
	return out, nil
}
