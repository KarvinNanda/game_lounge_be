package service

import (
	"errors"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"game_lounge_be/models"
	"game_lounge_be/modules/voucher/dto"
	"game_lounge_be/modules/voucher/repository"
	ntService "game_lounge_be/modules/notification_template/service"
	"game_lounge_be/utils"

	"github.com/google/uuid"
)

// ── Enriched Response ─────────────────────────────────────────────────────────

type VoucherWithStatus struct {
	models.Voucher
	Status       string `json:"status"`
	TotalMembers int64  `json:"total_members"`
}

func enrichVoucher(v models.Voucher) VoucherWithStatus {
	status := "active"
	now := time.Now()
	if !v.IsActive {
		status = "inactive"
	} else if v.EndDate != nil && v.EndDate.Before(now) {
		status = "expired"
	}
	total := repository.CountAllMembers()
	return VoucherWithStatus{Voucher: v, Status: status, TotalMembers: total}
}

// ── CRUD ──────────────────────────────────────────────────────────────────────

func GetAllVouchers(filter dto.VoucherFilter) ([]VoucherWithStatus, int64, map[string]interface{}, error) {
	vouchers, total, err := repository.FindAllVouchers(
		filter.Search, filter.Status, filter.Type, filter.StoreID,
		filter.Page, filter.PerPage,
	)
	if err != nil {
		return nil, 0, nil, err
	}

	result := make([]VoucherWithStatus, 0, len(vouchers))
	for _, v := range vouchers {
		result = append(result, enrichVoucher(v))
	}

	stats := repository.GetVoucherStats()
	return result, total, stats, nil
}

func GetVoucherByID(id string) (*VoucherWithStatus, error) {
	voucher, err := repository.FindVoucherByID(id)
	if err != nil {
		return nil, errors.New("voucher tidak ditemukan")
	}
	r := enrichVoucher(*voucher)
	return &r, nil
}

// GetAvailableVouchersForCustomer mengambil voucher yang bisa dipakai customer
// pada store dan tipe transaksi tertentu. Digunakan oleh frontend saat create booking.
func GetAvailableVouchersForCustomer(customerID, storeID, useType string) ([]VoucherWithStatus, error) {
	if useType == "" {
		useType = "booking"
	}
	vouchers, err := repository.FindAvailableVouchersForCustomer(customerID, storeID, useType)
	if err != nil {
		return nil, err
	}
	result := make([]VoucherWithStatus, 0, len(vouchers))
	for _, v := range vouchers {
		result = append(result, enrichVoucher(v))
	}
	return result, nil
}

// GenerateCode menghasilkan kode voucher dari nama voucher (akronim + timestamp).
func GenerateCode(name string) string {
	words := strings.Fields(strings.ToUpper(name))
	prefix := ""
	for _, w := range words {
		if len(w) > 0 {
			prefix += string(w[0])
		}
		if len(prefix) >= 4 {
			break
		}
	}
	if len(prefix) < 2 {
		nameUp := strings.ToUpper(name)
		end := 4
		if len(nameUp) < end {
			end = len(nameUp)
		}
		prefix = nameUp[:end]
	}
	suffix := fmt.Sprintf("%04d", time.Now().UnixNano()%10000)
	return prefix + suffix
}

func CreateVoucher(req dto.CreateVoucherRequest, createdBy string) (*VoucherWithStatus, error) {
	// Cek duplikasi kode
	if repository.IsCodeExists(req.Code, "") {
		return nil, errors.New("kode voucher sudah digunakan")
	}

	// Parse tanggal
	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		return nil, errors.New("format start_date tidak valid (YYYY-MM-DD)")
	}
	var endDate *time.Time
	if req.EndDate != "" {
		d, err := time.Parse("2006-01-02", req.EndDate)
		if err != nil {
			return nil, errors.New("format end_date tidak valid (YYYY-MM-DD)")
		}
		endDate = &d
	}

	var desc, channel *string
	if req.Description != "" {
		desc = &req.Description
	}
	if req.SendChannel != "" {
		channel = &req.SendChannel
	}
	var maxDiscount, minPurchase *float64
	if req.MaxDiscount > 0 {
		maxDiscount = &req.MaxDiscount
	}
	if req.MinPurchase > 0 {
		minPurchase = &req.MinPurchase
	}

	code := strings.ToUpper(req.Code)
	voucher := &models.Voucher{
		ID:            uuid.NewString(),
		Name:          req.Name,
		Code:          code,
		Description:   desc,
		Type:          req.Type,
		DiscountType:  req.DiscountType,
		DiscountValue: req.DiscountValue,
		MaxDiscount:   maxDiscount,
		MinPurchase:   minPurchase,
		StartDate:     startDate,
		EndDate:       endDate,
		SendChannel:   channel,
		IsAllStores:   req.IsAllStores,
		IsActive:      true,
		CreatedBy:     &createdBy,
	}

	if err := repository.CreateVoucher(voucher); err != nil {
		return nil, errors.New("gagal membuat voucher")
	}

	if !req.IsAllStores && len(req.StoreIDs) > 0 {
		_ = repository.SyncVoucherStores(voucher.ID, req.StoreIDs)
	}

	// Kirim notifikasi ke member secara async (tidak memblokir response)
	if req.SendChannel != "" {
		go sendVoucherNotification(
			voucher.ID, voucher.Name, voucher.Code,
			req.SendChannel, endDate, req.Description,
		)
	}

	return GetVoucherByID(voucher.ID)
}

// sendVoucherNotification mengirim notifikasi voucher ke semua member aktif secara async.
// Prioritas: template dari DB → fallback ke HTML hardcode yang sudah didesain.
func sendVoucherNotification(voucherID, name, code, channel string, endDate *time.Time, desc string) {
	members, err := repository.FindAllMembers()
	if err != nil || len(members) == 0 {
		return
	}

	expiry := "Tanpa batas"
	if endDate != nil {
		expiry = endDate.Format("02 Januari 2006")
	}

	for _, m := range members {
		// ── Email ────────────────────────────────────────────────
		if (channel == "email" || channel == "all") && m.Email != nil && *m.Email != "" {
			subject := fmt.Sprintf("Voucher Spesial untuk Kamu: %s", code)

			tmpl, tmplErr := ntService.GetRendered("voucher_notification", map[string]string{
				"nama_customer":     m.Name,
				"nama_voucher":      name,
				"kode_voucher":      code,
				"berlaku_sampai":    expiry,
				"deskripsi_voucher": desc,
			})

			if tmplErr == nil && tmpl.IsEmailActive {
				// Template DB tersedia
				if emailErr := utils.SendEmail(*m.Email, m.Name, tmpl.EmailSubject, tmpl.EmailBody); emailErr != nil {
					log.Printf("[Voucher] Gagal kirim email ke %s: %v", *m.Email, emailErr)
				}
			} else {
				// Fallback: HTML hardcode yang sudah didesain
				htmlContent := utils.BuildVoucherEmailHTML(m.Name, name, code, expiry, desc)
				if emailErr := utils.SendHTMLEmail(*m.Email, m.Name, subject, htmlContent); emailErr != nil {
					log.Printf("[Voucher] Gagal kirim email ke %s: %v", *m.Email, emailErr)
				}
			}
		}

		// ── WhatsApp (placeholder) ────────────────────────────────
		if channel == "whatsapp" || channel == "all" {
			log.Printf("[Voucher][WhatsApp placeholder] → %s: voucher %s", m.Whatsapp, code)
		}
	}

	_ = repository.UpdateTotalSent(voucherID, len(members))
}

func UpdateVoucher(id string, req dto.UpdateVoucherRequest, updatedBy string) (*VoucherWithStatus, error) {
	voucher, err := repository.FindVoucherByID(id)
	if err != nil {
		return nil, errors.New("voucher tidak ditemukan")
	}

	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		return nil, errors.New("format start_date tidak valid (YYYY-MM-DD)")
	}
	var endDate *time.Time
	if req.EndDate != "" {
		d, _ := time.Parse("2006-01-02", req.EndDate)
		endDate = &d
	}

	var desc *string
	if req.Description != "" {
		desc = &req.Description
	}
	var maxDiscount, minPurchase *float64
	if req.MaxDiscount > 0 {
		maxDiscount = &req.MaxDiscount
	}
	if req.MinPurchase > 0 {
		minPurchase = &req.MinPurchase
	}

	isActive := voucher.IsActive
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	voucher.Name = req.Name
	voucher.Description = desc
	voucher.Type = req.Type
	voucher.DiscountType = req.DiscountType
	voucher.DiscountValue = req.DiscountValue
	voucher.MaxDiscount = maxDiscount
	voucher.MinPurchase = minPurchase
	voucher.StartDate = startDate
	voucher.EndDate = endDate
	voucher.IsAllStores = req.IsAllStores
	voucher.IsActive = isActive
	voucher.UpdatedBy = &updatedBy

	if err := repository.UpdateVoucher(voucher); err != nil {
		return nil, errors.New("gagal update voucher")
	}

	if !req.IsAllStores {
		_ = repository.SyncVoucherStores(voucher.ID, req.StoreIDs)
	} else {
		_ = repository.SyncVoucherStores(voucher.ID, []string{})
	}

	return GetVoucherByID(voucher.ID)
}

func DeleteVoucher(id string, deletedBy string) error {
	voucher, err := repository.FindVoucherByID(id)
	if err != nil {
		return errors.New("voucher tidak ditemukan")
	}
	return repository.SoftDeleteVoucher(voucher, deletedBy)
}

// ── Validate & Redeem ─────────────────────────────────────────────────────────

// ValidateVoucher memvalidasi voucher sebelum digunakan saat booking.
// Cek: aktif, belum expired, store match, customer belum pernah pakai.
func ValidateVoucher(req dto.ValidateVoucherRequest) (*dto.ValidateVoucherResponse, error) {
	voucher, err := repository.FindVoucherByCode(req.Code)
	if err != nil {
		return &dto.ValidateVoucherResponse{IsValid: false, Message: "Kode voucher tidak ditemukan"}, nil
	}

	now := time.Now()

	if !voucher.IsActive {
		return &dto.ValidateVoucherResponse{IsValid: false, Message: "Voucher tidak aktif"}, nil
	}
	if voucher.StartDate.After(now) {
		return &dto.ValidateVoucherResponse{IsValid: false, Message: "Voucher belum berlaku"}, nil
	}
	if voucher.EndDate != nil && voucher.EndDate.Before(now) {
		return &dto.ValidateVoucherResponse{IsValid: false, Message: "Voucher sudah kadaluwarsa"}, nil
	}
	if voucher.Type != "both" && voucher.Type != req.UseType {
		return &dto.ValidateVoucherResponse{
			IsValid: false,
			Message: "Voucher tidak berlaku untuk jenis transaksi ini",
		}, nil
	}

	// Cek store
	if !voucher.IsAllStores {
		storeValid := false
		for _, s := range voucher.Stores {
			if s.StoreID == req.StoreID {
				storeValid = true
				break
			}
		}
		if !storeValid {
			return &dto.ValidateVoucherResponse{IsValid: false, Message: "Voucher tidak berlaku di cabang ini"}, nil
		}
	}

	// Cek minimum pembelian
	if voucher.MinPurchase != nil && req.Amount < *voucher.MinPurchase {
		return &dto.ValidateVoucherResponse{
			IsValid: false,
			Message: fmt.Sprintf("Minimum pembelian Rp %.0f", *voucher.MinPurchase),
		}, nil
	}

	// Cek 1x per user
	if repository.CheckUsageExists(voucher.ID, req.CustomerID) {
		return &dto.ValidateVoucherResponse{IsValid: false, Message: "Voucher sudah pernah digunakan"}, nil
	}

	discountAmount := calculateDiscount(req.Amount, voucher)

	return &dto.ValidateVoucherResponse{
		IsValid:        true,
		VoucherID:      voucher.ID,
		Code:           voucher.Code,
		DiscountAmount: discountAmount,
		Message:        "Voucher valid",
	}, nil
}

// calculateDiscount menghitung nominal diskon berdasarkan tipe.
func calculateDiscount(amount float64, v *models.Voucher) float64 {
	var discount float64
	if v.DiscountType == "percentage" {
		discount = math.Round(amount * v.DiscountValue / 100)
		if v.MaxDiscount != nil && discount > *v.MaxDiscount {
			discount = *v.MaxDiscount
		}
	} else {
		discount = v.DiscountValue
	}
	if discount > amount {
		discount = amount
	}
	return discount
}

// RedeemVoucher catat pemakaian voucher (dipanggil dari modul Booking).
func RedeemVoucher(voucherID, customerID, bookingID string, discountAmount float64) error {
	usage := &models.VoucherUsage{
		VoucherID:      voucherID,
		CustomerID:     customerID,
		DiscountAmount: discountAmount,
	}
	if bookingID != "" {
		usage.BookingID = &bookingID
	}
	if err := repository.CreateUsage(usage); err != nil {
		return errors.New("gagal mencatat pemakaian voucher")
	}
	return repository.IncrementUsedCount(voucherID)
}
