package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"game_lounge_be/config"
	"game_lounge_be/models"
	bookingRepo "game_lounge_be/modules/booking/repository"
	"game_lounge_be/modules/customer_booking/repository"
	pricingDto "game_lounge_be/modules/pricing/dto"
	pricingService "game_lounge_be/modules/pricing/service"
	"game_lounge_be/utils"

	"github.com/google/uuid"
)

// InitiateBookingRequest adalah payload dari customer untuk membuat hold + invoice.
type InitiateBookingRequest struct {
	StoreID        string  `json:"store_id" binding:"required"`
	RoomTemplateID uint    `json:"room_template_id" binding:"required"`
	BookingDate    string  `json:"booking_date" binding:"required"`
	StartTime      string  `json:"start_time" binding:"required"`
	DurationHours  float64 `json:"duration_hours" binding:"required,min=1,max=10"`
	PaymentMethod  string  `json:"payment_method" binding:"required"`
	VoucherID      string  `json:"voucher_id"` // opsional
}

// GetAvailability mengambil slot tersedia beserta harga yang dikalkulasi
// secara proper (mempertimbangkan happy hour, flash sale, package pricing).
// Setiap slot memiliki harga sendiri karena happy hour bergantung pada jam mulai.
func GetAvailability(storeID string, roomTemplateID uint, date string, durationHours float64) (map[string]interface{}, error) {
	// Ambil slot dari repository (hanya availability, tanpa harga)
	slots, err := repository.GetAvailableSlots(storeID, roomTemplateID, date, durationHours)
	if err != nil {
		return nil, err
	}

	// Hitung harga proper untuk setiap slot menggunakan pricing service
	type SlotWithPrice struct {
		StartTime string  `json:"start_time"`
		EndTime   string  `json:"end_time"`
		Available bool    `json:"available"`
		Price     float64 `json:"price"`     // harga aktual setelah kalkulasi
		Breakdown string  `json:"breakdown"` // deskripsi breakdown harga
	}

	var slotsWithPrice []SlotWithPrice
	for _, slot := range slots {
		s := SlotWithPrice{
			StartTime: slot.StartTime,
			EndTime:   slot.EndTime,
			Available: slot.Available,
		}

		// Kalkulasi harga hanya untuk slot yang tersedia
		if slot.Available {
			priceResult, priceErr := getPriceForSlot(
				storeID, roomTemplateID, date, slot.StartTime, slot.EndTime,
			)
			if priceErr == nil && priceResult != nil {
				s.Price = priceResult.FinalPrice
				// Ambil deskripsi breakdown pertama sebagai label harga
				if len(priceResult.Breakdown) > 0 {
					s.Breakdown = priceResult.Breakdown[0].Description
				}
			}
		}

		slotsWithPrice = append(slotsWithPrice, s)
	}

	return map[string]interface{}{
		"slots":          slotsWithPrice,
		"duration_hours": durationHours,
	}, nil
}

// InitiateBooking membuat hold + Xendit invoice, lalu mengembalikan URL pembayaran.
func InitiateBooking(req InitiateBookingRequest, customerID string) (map[string]interface{}, error) {
	// 1. Parse tanggal
	bookingDate, err := time.Parse("2006-01-02", req.BookingDate)
	if err != nil {
		return nil, errors.New("format tanggal tidak valid (gunakan YYYY-MM-DD)")
	}

	// 2. Hitung end time dari start time + durasi
	startMins := parseMins(req.StartTime)
	durMins   := int(req.DurationHours * 60)
	endMins   := startMins + durMins
	endTime   := minsToTime(endMins)

	// 3. Hitung harga menggunakan pricing service yang proper
	// (mempertimbangkan happy hour, flash sale, package pricing)
	priceResult, err := getPriceForSlot(req.StoreID, req.RoomTemplateID,
		req.BookingDate, req.StartTime, endTime)
	if err != nil {
		return nil, errors.New("gagal menghitung harga: " + err.Error())
	}
	totalPrice := priceResult.FinalPrice

	// Aplikasikan voucher jika ada
	discountAmount := 0.0
	finalPrice     := totalPrice
	if req.VoucherID != "" {
		var vErr error
		discountAmount, finalPrice, vErr = applyVoucher(req.VoucherID, customerID, totalPrice)
		if vErr != nil {
			return nil, vErr
		}
	}

	// Buat breakdown string untuk disimpan di hold
	breakdownJSON := fmt.Sprintf(
		`{"base":%.0f,"discount":%.0f,"final":%.0f,"voucher_id":"%s","has_flash_sale":%v}`,
		totalPrice, discountAmount, finalPrice, req.VoucherID, priceResult.HasFlashSale,
	)

	// 4. Ambil data customer
	var customer models.Customer
	config.DB.Where("id = ?", customerID).First(&customer)

	// 5. Buat hold (atomic, pakai DB transaction untuk cegah race condition)
	hold := &models.BookingHold{
		ID:             uuid.NewString(),
		CustomerID:     customerID,
		StoreID:        req.StoreID,
		RoomTemplateID: req.RoomTemplateID,
		BookingDate:    bookingDate,
		StartTime:      req.StartTime,
		EndTime:        endTime,
		DurationHours:  req.DurationHours,
		BasePrice:      totalPrice,   // harga sebelum diskon
		TotalPrice:     finalPrice,   // harga setelah diskon
		PriceBreakdown: breakdownJSON,
		PaymentMethod:  req.PaymentMethod,
	}

	createdHold, err := repository.CreateHold(hold)
	if err != nil {
		return nil, err
	}

	// 6. Buat Xendit invoice
	appURL     := os.Getenv("APP_URL")
	externalID := fmt.Sprintf("BK-%s-%s", customerID[:8], createdHold.ID[:8])
	payerEmail := ""
	if customer.Email != nil {
		payerEmail = *customer.Email
	}

	invoice, err := utils.CreateXenditInvoice(utils.XenditInvoiceRequest{
		ExternalID:  externalID,
		Amount:      finalPrice,
		PayerEmail:  payerEmail,
		Description: fmt.Sprintf("Booking %s — %s %s", createdHold.RoomID, req.BookingDate, req.StartTime),
		SuccessURL:  fmt.Sprintf("%s/payment/success?hold_id=%s", appURL, createdHold.ID),
		FailureURL:  fmt.Sprintf("%s/payment/failed?hold_id=%s", appURL, createdHold.ID),
	})
	if err != nil {
		repository.DeleteHold(createdHold.ID)
		return nil, errors.New("gagal membuat invoice pembayaran")
	}

	// 7. Simpan invoice info ke hold
	createdHold.XenditInvoiceID  = &invoice.ID
	createdHold.XenditInvoiceURL = &invoice.InvoiceURL
	repository.UpdateHold(createdHold)

	invoiceURL := invoice.InvoiceURL
	if strings.Contains(invoiceURL, "/payment/mock") {
		invoiceURL = fmt.Sprintf("%s&hold_id=%s", invoiceURL, createdHold.ID)
		createdHold.XenditInvoiceURL = &invoiceURL
		repository.UpdateHold(createdHold)
	}

	return map[string]interface{}{
		"hold_id":         createdHold.ID,
		"invoice_url":     invoiceURL,
		"expires_at":      createdHold.ExpiresAt,
		"base_price":      totalPrice,
		"discount_amount": discountAmount,
		"total_price":     finalPrice,
		"price_breakdown": priceResult.Breakdown,
		"has_flash_sale":  priceResult.HasFlashSale,
		"flash_sale_name": priceResult.FlashSaleName,
		"booking_preview": map[string]interface{}{
			"store_id":        req.StoreID,
			"booking_date":    req.BookingDate,
			"start_time":      req.StartTime,
			"end_time":        endTime,
			"duration_hours":  req.DurationHours,
			"base_price":      totalPrice,
			"discount_amount": discountAmount,
			"total_price":     finalPrice,
			"price_breakdown": priceResult.Breakdown,
		},
	}, nil
}

// ConfirmBookingFromWebhook dipanggil saat Xendit webhook PAID diterima.
// Mengkonversi hold menjadi Booking permanen dan menghapus hold.
func ConfirmBookingFromWebhook(xenditInvoiceID, paymentMethod string) error {
	// 1. Temukan hold berdasarkan invoice ID
	hold, err := repository.FindHoldByXenditInvoice(xenditInvoiceID)
	if err != nil {
		return errors.New("hold tidak ditemukan untuk invoice ini")
	}

	// 2. Cek hold belum expire
	if hold.IsExpired() {
		return errors.New("hold sudah expired — slot tidak lagi direservasi")
	}

	// 3. Generate booking code
	bookingCode, err := bookingRepo.GenerateBookingCode()
	if err != nil {
		return errors.New("gagal generate kode booking")
	}

	// 4. Buat booking record permanen
	customerIDPtr := hold.CustomerID
	customerName  := hold.Customer.Name
	customerWA    := hold.Customer.Whatsapp
	var customerEmail *string
	if hold.Customer.Email != nil {
		customerEmail = hold.Customer.Email
	}

	booking := &models.Booking{
		ID:               uuid.NewString(),
		BookingCode:      bookingCode,
		CustomerID:       &customerIDPtr,
		CustomerName:     customerName,
		CustomerWhatsapp: &customerWA,
		CustomerEmail:    customerEmail,
		StoreID:          hold.StoreID,
		RoomID:           hold.RoomID,
		BookingDate:      hold.BookingDate,
		StartTime:        hold.StartTime,
		EndTime:          hold.EndTime,
		DurationHours:    hold.DurationHours,
		BasePrice:        hold.BasePrice,
		TotalPrice:       hold.TotalPrice,
		PaymentMethod:    "cash", // Xendit dikategorikan cash untuk kompatibilitas enum existing
		Status:           "upcoming",
		CreatedBy:        &customerIDPtr,
	}

	if err := config.DB.Create(booking).Error; err != nil {
		return errors.New("gagal membuat booking")
	}

	// 5. Tandai voucher sebagai terpakai jika ada dalam breakdown
	if strings.Contains(hold.PriceBreakdown, `"voucher_id":`) {
		var breakdown struct {
			VoucherID string `json:"voucher_id"`
		}
		if json.Unmarshal([]byte(hold.PriceBreakdown), &breakdown) == nil && breakdown.VoucherID != "" {
			config.DB.Model(&models.VoucherUsage{}).
				Where("voucher_id = ? AND customer_id = ? AND booking_id IS NULL",
					breakdown.VoucherID, hold.CustomerID).
				Update("booking_id", booking.ID)
		}
	}

	// 6. Hapus hold setelah booking berhasil dibuat
	repository.DeleteHold(hold.ID)

	// 6. Kirim email konfirmasi secara async
	go utils.SendBookingConfirmationEmail(hold.Customer, booking, hold.Store)

	return nil
}

// GetCustomerBookings mengambil list booking milik customer dengan pagination.
func GetCustomerBookings(customerID string, page, perPage int) ([]models.Booking, int64, error) {
	return repository.FindCustomerBookings(customerID, page, perPage)
}

// GetCustomerBookingByID mengambil satu booking milik customer.
func GetCustomerBookingByID(id, customerID string) (*models.Booking, error) {
	b, err := repository.FindCustomerBookingByID(id, customerID)
	if err != nil {
		return nil, errors.New("booking tidak ditemukan")
	}
	return b, nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// getPriceForSlot menghitung harga untuk 1 slot booking menggunakan
// pricing service yang sudah ada. Ini memastikan happy hour, flash sale,
// dan package pricing dihitung dengan benar — sama persis seperti admin booking.
func getPriceForSlot(storeID string, roomTemplateID uint, date, startTime, endTime string) (*pricingDto.CalculatePriceResponse, error) {
	return pricingService.CalculatePrice(pricingDto.CalculatePriceRequest{
		StoreID:        storeID,
		RoomTemplateID: roomTemplateID,
		BookingDate:    date,
		StartTime:      startTime,
		EndTime:        endTime,
	})
}

func parseMins(t string) int {
	if len(t) < 5 {
		return 0
	}
	h := int(t[0]-'0')*10 + int(t[1]-'0')
	m := int(t[3]-'0')*10 + int(t[4]-'0')
	return h*60 + m
}

func minsToTime(m int) string {
	m = m % (24 * 60)
	return fmt.Sprintf("%02d:%02d", m/60, m%60)
}

// applyVoucher memvalidasi voucher dan mengembalikan diskon + harga akhir.
func applyVoucher(voucherID, customerID string, basePrice float64) (discountAmount, finalPrice float64, err error) {
	// Cek voucher dialokasikan ke customer dan belum dipakai
	var vu models.VoucherUsage
	if dbErr := config.DB.
		Where("voucher_id = ? AND customer_id = ? AND booking_id IS NULL", voucherID, customerID).
		First(&vu).Error; dbErr != nil {
		return 0, basePrice, errors.New("voucher tidak valid atau sudah digunakan")
	}

	// Ambil detail voucher
	var voucher models.Voucher
	if dbErr := config.DB.Where("id = ? AND is_active = true AND deleted_at IS NULL", voucherID).
		First(&voucher).Error; dbErr != nil {
		return 0, basePrice, errors.New("voucher tidak ditemukan atau tidak aktif")
	}

	// Cek minimum purchase
	if voucher.MinPurchase != nil && basePrice < *voucher.MinPurchase {
		return 0, basePrice, fmt.Errorf("minimum pembelian Rp %.0f untuk menggunakan voucher ini", *voucher.MinPurchase)
	}

	// Hitung diskon
	var discount float64
	if voucher.DiscountType == "percentage" {
		discount = basePrice * voucher.DiscountValue / 100
		if voucher.MaxDiscount != nil && discount > *voucher.MaxDiscount {
			discount = *voucher.MaxDiscount
		}
	} else {
		discount = voucher.DiscountValue
	}
	if discount > basePrice {
		discount = basePrice
	}

	return discount, basePrice - discount, nil
}
