package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"strings"
	"time"

	"game_lounge_be/config"
	"game_lounge_be/models"
	bookingRepo "game_lounge_be/modules/booking/repository"
	"game_lounge_be/modules/customer_booking/repository"
	pricingDto "game_lounge_be/modules/pricing/dto"
	pricingService "game_lounge_be/modules/pricing/service"
	voucherDto "game_lounge_be/modules/voucher/dto"
	voucherRepo "game_lounge_be/modules/voucher/repository"
	voucherService "game_lounge_be/modules/voucher/service"
	"game_lounge_be/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// InitiateBookingRequest adalah payload dari customer untuk membuat hold + invoice.
// SelectedSlots berisi list jam mulai per-1-jam yang BERURUTAN: ["10:00","11:00","12:00"].
// Lihat resolveSlotRange.
type InitiateBookingRequest struct {
	StoreID        string   `json:"store_id" binding:"required"`
	RoomTemplateID uint     `json:"room_template_id" binding:"required"`
	BookingDate    string   `json:"booking_date" binding:"required"`
	SelectedSlots  []string `json:"selected_slots" binding:"required,min=1"` // e.g. ["10:00","11:00"]
	PaymentMethod  string   `json:"payment_method" binding:"required"`
	VoucherID      string   `json:"voucher_id"` // opsional
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

	// 2–3. Validasi slot & hitung harga (sama persis dengan endpoint quote)
	quote, err := quoteSlots(req.StoreID, req.RoomTemplateID, req.BookingDate, req.SelectedSlots)
	if err != nil {
		return nil, err
	}
	startTime, endTime := quote.StartTime, quote.EndTime
	durationHours := float64(quote.Hours)
	totalPrice := quote.Price.FinalPrice
	hasFlashSale := quote.Price.HasFlashSale

	// 4. Aplikasikan voucher jika ada — validasi lengkap yang sama dengan admin
	// (tanggal, tipe, store, room type, minimum, member, 1x per customer).
	discountAmount := 0.0
	voucherCode := ""
	var reservation *models.VoucherUsage
	if req.VoucherID != "" {
		voucher, err := voucherRepo.FindVoucherByID(req.VoucherID)
		if err != nil {
			return nil, errors.New("voucher tidak ditemukan")
		}
		check, err := voucherService.ValidateVoucher(voucherDto.ValidateVoucherRequest{
			Code:           voucher.Code,
			CustomerID:     customerID,
			StoreID:        req.StoreID,
			Amount:         totalPrice,
			UseType:        "booking",
			RoomTemplateID: req.RoomTemplateID,
		})
		if err != nil {
			return nil, err
		}
		if !check.IsValid {
			return nil, errors.New(check.Message)
		}
		discountAmount = math.Round(check.DiscountAmount) // IDR tanpa desimal
		voucherCode = voucher.Code
		reservation = &models.VoucherUsage{
			VoucherID:      voucher.ID,
			CustomerID:     customerID,
			DiscountAmount: discountAmount,
		}
	}
	finalPrice := math.Round(totalPrice - discountAmount)

	// Buat breakdown string untuk disimpan di hold
	breakdownBytes, _ := json.Marshal(map[string]any{
		"base": totalPrice, "discount": discountAmount, "final": finalPrice,
		"voucher_id": req.VoucherID, "voucher_code": voucherCode, "has_flash_sale": hasFlashSale,
	})
	breakdownJSON := string(breakdownBytes)

	// 5. Ambil data customer
	var customer models.Customer
	config.DB.Where("id = ?", customerID).First(&customer)

	// 6. Buat hold (atomic, pakai DB transaction untuk cegah race condition)
	hold := &models.BookingHold{
		ID:             uuid.NewString(),
		CustomerID:     customerID,
		StoreID:        req.StoreID,
		RoomTemplateID: req.RoomTemplateID,
		BookingDate:    bookingDate,
		StartTime:      startTime,
		EndTime:        endTime,
		DurationHours:  durationHours,
		BasePrice:      totalPrice,   // harga sebelum diskon
		TotalPrice:     finalPrice,   // harga setelah diskon
		PriceBreakdown: breakdownJSON,
		PaymentMethod:  req.PaymentMethod,
	}

	createdHold, err := repository.CreateHold(hold, reservation)
	if err != nil {
		return nil, err
	}

	// 7. Buat Xendit invoice
	appURL     := os.Getenv("APP_URL")
	externalID := fmt.Sprintf("BK-%s-%s", customerID[:8], createdHold.ID[:8])
	payerEmail := ""
	if customer.Email != nil {
		payerEmail = *customer.Email
	}

	invoice, err := utils.CreateXenditInvoice(utils.XenditInvoiceRequest{
		ExternalID:      externalID,
		Amount:          finalPrice,
		InvoiceDuration: int(repository.HoldDuration.Seconds()),
		PayerEmail:  payerEmail,
		Description: fmt.Sprintf("Booking %s — %s %s", createdHold.RoomID, req.BookingDate, startTime),
		SuccessURL:  fmt.Sprintf("%s/payment/success?hold_id=%s", appURL, createdHold.ID),
		FailureURL:  fmt.Sprintf("%s/payment/failed?hold_id=%s", appURL, createdHold.ID),
	})
	if err != nil {
		if relErr := repository.ReleaseHold(createdHold); relErr != nil {
			log.Printf("gagal melepas hold %s: %v", createdHold.ID, relErr)
		}
		return nil, errors.New("gagal membuat invoice pembayaran")
	}

	// 8. Simpan invoice info ke hold
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
		"expires_at":      createdHold.ExpiresAt.Format(time.RFC3339),
		"base_price":      totalPrice,
		"discount_amount": discountAmount,
		"total_price":     finalPrice,
		"has_flash_sale":  hasFlashSale,
		"booking_preview": map[string]interface{}{
			"store_id":        req.StoreID,
			"booking_date":    req.BookingDate,
			"start_time":      startTime,
			"end_time":        endTime,
			"duration_hours":  durationHours,
			"selected_slots":  req.SelectedSlots,
			"base_price":      totalPrice,
			"discount_amount": discountAmount,
			"total_price":     finalPrice,
		},
	}, nil
}

// slotQuote = hasil validasi slot + harga untuk 1 rentang booking.
type slotQuote struct {
	StartTime, EndTime string
	Hours              int
	Available          bool
	Price              *pricingDto.CalculatePriceResponse
}

// quoteSlots dipakai oleh InitiateBooking dan endpoint quote publik, supaya
// harga yang ditampilkan sebelum bayar = harga yang ditagih.
func quoteSlots(storeID string, roomTemplateID uint, bookingDate string, selected []string) (*slotQuote, error) {
	if _, err := time.Parse("2006-01-02", bookingDate); err != nil {
		return nil, errors.New("format tanggal tidak valid (gunakan YYYY-MM-DD)")
	}
	loc, _ := time.LoadLocation("Asia/Jakarta")
	if bookingDate < time.Now().In(loc).Format("2006-01-02") {
		return nil, errors.New("tanggal booking sudah lewat")
	}

	// Slot resmi hari itu (urut dari jam buka store)
	daySlots, err := repository.GetAvailableSlots(storeID, roomTemplateID, bookingDate, 1)
	if err != nil {
		return nil, errors.New("gagal memuat jadwal ruangan")
	}
	dayOrder := make([]string, 0, len(daySlots))
	free := make(map[string]bool, len(daySlots))
	for _, sl := range daySlots {
		dayOrder = append(dayOrder, sl.StartTime)
		free[sl.StartTime] = sl.Available
	}
	start, end, hours, err := resolveSlotRange(selected, dayOrder)
	if err != nil {
		return nil, err
	}
	available := true
	for _, sl := range selected {
		available = available && free[sl]
	}

	// Harga untuk SELURUH rentang sekaligus — paket multi-jam ikut diperhitungkan.
	price, err := pricingService.CalculatePrice(pricingDto.CalculatePriceRequest{
		StoreID:        storeID,
		RoomTemplateID: roomTemplateID,
		BookingDate:    bookingDate,
		StartTime:      start,
		EndTime:        end,
	})
	if err != nil {
		return nil, errors.New("gagal menghitung harga")
	}
	return &slotQuote{StartTime: start, EndTime: end, Hours: hours, Available: available, Price: price}, nil
}

// QuoteBooking = harga booking tanpa membuat hold (read-only, publik).
func QuoteBooking(storeID string, roomTemplateID uint, bookingDate string, selected []string) (map[string]interface{}, error) {
	q, err := quoteSlots(storeID, roomTemplateID, bookingDate, selected)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"store_id":         storeID,
		"room_template_id": roomTemplateID,
		"booking_date":     bookingDate,
		"start_time":       q.StartTime,
		"end_time":         q.EndTime,
		"duration_hours":   q.Hours,
		"available":        q.Available,
		"base_price":       q.Price.BasePrice,
		"flash_discount":   q.Price.FlashDiscount,
		"total_price":      math.Round(q.Price.FinalPrice),
		"has_flash_sale":   q.Price.HasFlashSale,
		"flash_sale_name":  q.Price.FlashSaleName,
		"breakdown":        q.Price.Breakdown, // paket / happy hour / flash sale yang dipakai
	}, nil
}

var (
	// ErrHoldNotFound: hold untuk invoice ini tidak ada — sudah diproses oleh
	// webhook sebelumnya (hold dihapus saat booking dibuat) atau invoice asing.
	ErrHoldNotFound = errors.New("hold tidak ditemukan untuk invoice ini")
	// ErrPaidButSlotTaken: customer membayar setelah hold habis dan slot sudah
	// diambil orang lain. Butuh refund manual — jangan di-retry.
	ErrPaidButSlotTaken = errors.New("pembayaran diterima tetapi slot sudah terpakai — perlu refund manual")
)

// ConfirmBookingFromWebhook dipanggil saat Xendit webhook PAID diterima.
// Mengkonversi hold menjadi Booking permanen dan menghapus hold.
//
// Idempotent: hold "diklaim" dengan DELETE di dalam transaction yang sama dengan
// INSERT booking. Webhook yang dikirim ulang/bersamaan akan mendapat 0 baris
// dan tidak membuat booking kedua.
func ConfirmBookingFromWebhook(xenditInvoiceID, paymentMethod string) error {
	hold, err := repository.FindHoldByXenditInvoice(xenditInvoiceID)
	if err != nil {
		return ErrHoldNotFound
	}

	// Hold habis tapi uang sudah masuk: tetap buat booking jika slot masih kosong.
	if hold.IsExpired() {
		taken, err := bookingRepo.CheckOverlap(hold.RoomID, hold.StoreID,
			hold.BookingDate.Format("2006-01-02"), hold.StartTime, hold.EndTime, "")
		if err != nil {
			return err
		}
		if taken {
			log.Printf("[PAYMENT ORPHAN] invoice=%s hold=%s customer=%s amount=%.0f: %v",
				xenditInvoiceID, hold.ID, hold.CustomerID, hold.TotalPrice, ErrPaidButSlotTaken)
			return ErrPaidButSlotTaken
		}
	}

	bookingCode, err := bookingRepo.GenerateBookingCode()
	if err != nil {
		return errors.New("gagal generate kode booking")
	}

	customerID := hold.CustomerID
	customerWA := hold.Customer.Whatsapp
	booking := &models.Booking{
		ID:               uuid.NewString(),
		BookingCode:      bookingCode,
		CustomerID:       &customerID,
		CustomerName:     hold.Customer.Name,
		CustomerWhatsapp: &customerWA,
		CustomerEmail:    hold.Customer.Email,
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
		CreatedBy:        &customerID,
		HoldID:           &hold.ID,
	}
	voucher := holdVoucher(hold.PriceBreakdown)
	if voucher.ID != "" {
		booking.VoucherID = &voucher.ID
		booking.VoucherCode = &voucher.Code
		booking.DiscountAmount = hold.BasePrice - hold.TotalPrice
	}

	errAlreadyProcessed := errors.New("already processed")
	err = config.DB.Transaction(func(tx *gorm.DB) error {
		claim := tx.Delete(&models.BookingHold{}, "id = ?", hold.ID)
		if claim.Error != nil {
			return claim.Error
		}
		if claim.RowsAffected != 1 {
			return errAlreadyProcessed
		}
		if err := tx.Create(booking).Error; err != nil {
			return err
		}
		return markVoucherUsed(tx, voucher.ID, hold.CustomerID, booking.ID, booking.DiscountAmount)
	})
	if errors.Is(err, errAlreadyProcessed) {
		return ErrHoldNotFound
	}
	if err != nil {
		return fmt.Errorf("gagal membuat booking: %w", err)
	}

	utils.SafeGo(func() { utils.SendBookingConfirmationEmail(hold.Customer, booking, hold.Store) })
	return nil
}

type holdVoucherInfo struct {
	ID   string `json:"voucher_id"`
	Code string `json:"voucher_code"`
}

func holdVoucher(priceBreakdown string) holdVoucherInfo {
	var v holdVoucherInfo
	_ = json.Unmarshal([]byte(priceBreakdown), &v)
	return v
}

// markVoucherUsed menautkan reservasi voucher ke booking dan menaikkan used_count.
// Jika reservasi sudah hilang (mis. hold dibersihkan sebelum webhook datang),
// usage dibuat ulang; bila UNIQUE menolak, booking tetap dibuat karena customer
// sudah membayar harga diskon — kejadian ini dicatat di log.
func markVoucherUsed(tx *gorm.DB, voucherID, customerID, bookingID string, discount float64) error {
	if voucherID == "" {
		return nil
	}
	res := tx.Model(&models.VoucherUsage{}).
		Where("voucher_id = ? AND customer_id = ? AND booking_id IS NULL", voucherID, customerID).
		Updates(map[string]interface{}{"booking_id": bookingID, "used_at": time.Now()})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		usage := &models.VoucherUsage{VoucherID: voucherID, CustomerID: customerID, BookingID: &bookingID, DiscountAmount: discount}
		if err := tx.Create(usage).Error; err != nil {
			if !utils.IsDuplicateKey(err) {
				return err
			}
			log.Printf("[VOUCHER] voucher %s sudah dipakai customer %s; booking %s tetap dibuat", voucherID, customerID, bookingID)
			return nil
		}
	}
	return tx.Model(&models.Voucher{}).Where("id = ?", voucherID).
		Update("used_count", gorm.Expr("used_count + 1")).Error
}

// GetCustomerBookings mengambil list booking milik customer dengan pagination.
func GetCustomerBookings(customerID string, statuses []string, page, perPage int) ([]models.Booking, int64, error) {
	if err := repository.SyncCustomerBookingStatus(customerID); err != nil {
		return nil, 0, err
	}
	return repository.FindCustomerBookings(customerID, statuses, page, perPage)
}

// GetCustomerBookingByID mengambil satu booking milik customer.
func GetCustomerBookingByID(id, customerID string) (*models.Booking, error) {
	b, err := repository.FindCustomerBookingByID(id, customerID)
	if err != nil {
		return nil, errors.New("booking tidak ditemukan")
	}
	// Status di DB bisa basi (lihat SyncCustomerBookingStatus): hitung dari waktu sekarang.
	open, _ := repository.OperatingWindow(b.StoreID, b.BookingDate.Format("2006-01-02"))
	b.Status = utils.SessionStatus(b.Status, b.BookingDate, b.StartTime, b.EndTime, open, time.Now())
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


// HoldStatus = status pembayaran sebuah hold untuk halaman payment success.
type HoldStatus struct {
	Status      string  `json:"status"` // pending | confirmed | expired
	BookingCode *string `json:"booking_code,omitempty"`
	BookingID   *string `json:"booking_id,omitempty"`
}

// GetHoldStatus memetakan hold ke booking untuk halaman payment success.
// Webhook PAID menghapus hold dan membuat booking ber-hold_id, jadi booking dicek dulu.
// Hold expired bisa masih jadi booking jika webhook terlambat (lihat ConfirmBookingFromWebhook),
// jadi "expired" belum final selama hold masih ada.
// Selalu dibatasi customerID: hold/booking milik customer lain = ErrHoldNotFound.
func GetHoldStatus(holdID, customerID string) (*HoldStatus, error) {
	if b, err := repository.FindBookingByHold(holdID, customerID); err == nil {
		return &HoldStatus{Status: "confirmed", BookingCode: &b.BookingCode, BookingID: &b.ID}, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	h, err := repository.FindCustomerHold(holdID, customerID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrHoldNotFound
	}
	if err != nil {
		return nil, err
	}
	if h.IsExpired() {
		return &HoldStatus{Status: "expired"}, nil
	}
	return &HoldStatus{Status: "pending"}, nil
}
