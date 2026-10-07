package service

import (
	"encoding/json"
	"errors"
	"log"
	"math"
	"time"

	"game_lounge_be/config"
	"game_lounge_be/models"
	customerBookingRepo "game_lounge_be/modules/customer_booking/repository"
	"game_lounge_be/modules/event_booking/dto"
	"game_lounge_be/modules/event_booking/repository"
	storeRepository "game_lounge_be/modules/store/repository"
	"game_lounge_be/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ── Helpers ───────────────────────────────────────────────────────────────────

// isHHMM menerima "HH:MM" atau "HH:MM:SS" dengan jam 00-23 dan menit 00-59.
func isHHMM(t string) bool {
	if len(t) < 5 {
		return false
	}
	_, err := time.Parse("15:04", t[:5])
	return err == nil
}

// computeStatus menghitung status event booking dari waktu sekarang (WIB).
func computeStatus(b models.EventBooking) string {
	return computeStatusAt(b, eventOpenMins(b.StoreID, b.BookingDate), time.Now())
}

// computeStatusAt: waktu absolut (utils.SessionStatus) supaya event lintas tengah
// malam (mis. full day 09:00–02:00) tetap "ongoing" sampai 02:00.
func computeStatusAt(b models.EventBooking, openMins int, now time.Time) string {
	return utils.SessionStatus(b.Status, b.BookingDate, b.StartTime, b.EndTime, openMins, now)
}

func eventOpenMins(storeID string, date time.Time) int {
	open, _ := customerBookingRepo.OperatingWindow(storeID, date.Format("2006-01-02"))
	return open
}

// calculateTotalPrice menghitung harga proporsional.
// Formula: round((price_per_day / 24 × duration_hours) / 1000) × 1000
// Dibulatkan ke Rp 1.000 terdekat.
func calculateTotalPrice(pricePerDay, durationHours float64) float64 {
	raw := pricePerDay / 24 * durationHours
	return math.Round(raw/1000) * 1000
}

// eventDuration memvalidasi jam lalu menghitung durasi (jam). end <= start = lewat tengah malam.
// Dipakai Create & QuoteCustomerEvent supaya quote selalu sama dengan harga yang ditagih.
func eventDuration(startTime, endTime string) (float64, error) {
	if !isHHMM(startTime) || !isHHMM(endTime) {
		return 0, errors.New("format jam tidak valid (HH:MM)")
	}
	// start == end dulu dihitung sebagai event 24 jam.
	if startTime[:5] == endTime[:5] {
		return 0, errors.New("jam mulai dan jam selesai tidak boleh sama")
	}
	startMins := utils.MinsOf(startTime)
	endMins := utils.MinsOf(endTime)
	if endMins <= startMins {
		endMins += 24 * 60
	}
	return float64(endMins-startMins) / 60.0, nil
}

// configuredEventPrice = harga event store; error jika belum dikonfigurasi.
func configuredEventPrice(storeID string) (*models.StoreEventPrice, error) {
	eventPrice, err := repository.GetEventPrice(storeID)
	if err != nil || eventPrice.PricePerDay == 0 {
		return nil, errors.New("harga event untuk cabang ini belum dikonfigurasi. Silakan atur di menu Pricing")
	}
	return eventPrice, nil
}

// ── CRUD ──────────────────────────────────────────────────────────────────────

// GetAll mengambil list event booking.
func GetAll(filter dto.EventBookingFilter) ([]models.EventBooking, int64, error) {
	bookings, total, err := repository.FindAll(
		filter.StoreID, filter.Status, filter.DateFrom, filter.DateTo,
		filter.Page, filter.PerPage,
	)
	if err != nil {
		return nil, 0, err
	}
	for i := range bookings {
		bookings[i].Status = computeStatus(bookings[i])
	}
	return bookings, total, nil
}

// GetByID mengambil satu event booking.
func GetByID(id string) (*models.EventBooking, error) {
	b, err := repository.FindByID(id)
	if err != nil {
		return nil, errors.New("event booking tidak ditemukan")
	}
	b.Status = computeStatus(*b)
	return b, nil
}

// Create membuat event booking baru dengan validasi lengkap.
func Create(req dto.CreateEventBookingRequest, createdBy string) (*models.EventBooking, error) {
	bookingDate, err := time.Parse("2006-01-02", req.BookingDate)
	if err != nil {
		return nil, errors.New("format booking_date tidak valid (YYYY-MM-DD)")
	}

	// ── Hitung start/end time berdasarkan duration_type ───────────────
	startTime := req.StartTime
	endTime := req.EndTime

	if req.DurationType == "full_day" {
		var opErr error
		startTime, endTime, opErr = getStoreOperatingHours(req.StoreID, req.BookingDate)
		if opErr != nil {
			return nil, errors.New("gagal mengambil jam operasional store")
		}
	} else {
		// hourly: start dan end wajib ada
		if startTime == "" || endTime == "" {
			return nil, errors.New("start_time dan end_time wajib diisi untuk booking per jam")
		}
	}

	durationHours, err := eventDuration(startTime, endTime)
	if err != nil {
		return nil, err
	}

	// ── Validasi overlap ──────────────────────────────────────────────
	hasRegularOverlap, _ := repository.CheckOverlapWithRegular(
		req.StoreID, req.BookingDate, startTime, endTime,
	)
	if hasRegularOverlap {
		return nil, errors.New("ada booking reguler yang conflict pada jam tersebut. Batalkan booking reguler tersebut terlebih dahulu")
	}

	hasEventOverlap, _ := repository.CheckOverlapWithEvent(
		req.StoreID, req.BookingDate, startTime, endTime, "",
	)
	if hasEventOverlap {
		return nil, errors.New("sudah ada event booking lain pada jam tersebut")
	}

	// ── Harga ─────────────────────────────────────────────────────────
	eventPrice, err := configuredEventPrice(req.StoreID)
	if err != nil {
		return nil, err
	}

	var totalPrice float64
	if req.DurationType == "full_day" {
		totalPrice = eventPrice.PricePerDay // full day = harga penuh
	} else {
		totalPrice = calculateTotalPrice(eventPrice.PricePerDay, durationHours)
	}

	// ── Simpan selected_room_template_ids (per_room_type) ─────────────
	var selectedRoomIDs *string
	if req.BookingScope == "per_room_type" && len(req.SelectedRoomTemplateIDs) > 0 {
		idsJSON, _ := json.Marshal(req.SelectedRoomTemplateIDs)
		idsStr := string(idsJSON)
		selectedRoomIDs = &idsStr
	}

	var wa, email, notes, desc *string
	if req.CustomerWhatsapp != "" {
		wa = &req.CustomerWhatsapp
	}
	if req.CustomerEmail != "" {
		email = &req.CustomerEmail
	}
	if req.Notes != "" {
		notes = &req.Notes
	}
	if req.Description != "" {
		desc = &req.Description
	}

	// Default durationType/bookingScope jika kosong (misal panggilan internal)
	durationType := req.DurationType
	bookingScope := req.BookingScope
	if durationType == "" {
		durationType = "hourly"
	}
	if bookingScope == "" {
		bookingScope = "full_venue"
	}

	booking := &models.EventBooking{
		ID:                      uuid.NewString(),
		StoreID:                 req.StoreID,
		EventName:               req.EventName,
		Description:             desc,
		CustomerName:            req.CustomerName,
		CustomerWhatsapp:        wa,
		CustomerEmail:           email,
		BookingDate:             bookingDate,
		StartTime:               startTime,
		EndTime:                 endTime,
		DurationHours:           durationHours,
		PricePerDay:             eventPrice.PricePerDay,
		TotalPrice:              totalPrice,
		Status:                  "upcoming",
		DurationType:            durationType,
		BookingScope:            bookingScope,
		SelectedRoomTemplateIDs: selectedRoomIDs,
		Notes:                   notes,
		CreatedBy:               &createdBy,
	}

	if err := repository.Create(booking); err != nil {
		return nil, errors.New("gagal membuat event booking")
	}

	return GetByID(booking.ID)
}

// getStoreOperatingHours mengambil jam buka-tutup store pada tanggal tertentu.
// Fallback ke 10:00–02:00 jika data belum dikonfigurasi.
func getStoreOperatingHours(storeID, date string) (openTime, closeTime string, err error) {
	bookingDate, _ := time.Parse("2006-01-02", date)
	dayType := "weekday"
	if bookingDate.Weekday() == time.Saturday || bookingDate.Weekday() == time.Sunday {
		dayType = "weekend"
	}

	var opHour models.StoreOperatingHour
	if dbErr := config.DB.Where(
		"store_id = ? AND day_type = ? AND is_active = true AND deleted_at IS NULL",
		storeID, dayType,
	).First(&opHour).Error; dbErr != nil {
		return "10:00", "02:00", nil // default fallback
	}
	return opHour.OpenTime[:5], opHour.CloseTime[:5], nil
}

// Cancel membatalkan event booking dengan alasan wajib.
func Cancel(id string, req dto.CancelEventBookingRequest, cancelledBy string) (*models.EventBooking, error) {
	b, err := repository.FindByID(id)
	if err != nil {
		return nil, errors.New("event booking tidak ditemukan")
	}
	if b.Status == "cancelled" {
		return nil, errors.New("event booking sudah dibatalkan")
	}
	if b.Status == "completed" {
		return nil, errors.New("event booking sudah selesai")
	}

	now := time.Now()
	b.Status = "cancelled"
	b.CancelReason = &req.Reason
	b.CancelledAt = &now
	b.CancelledBy = &cancelledBy

	if err := repository.Update(b); err != nil {
		return nil, errors.New("gagal membatalkan event booking")
	}
	return GetByID(b.ID)
}

// GetForDashboard mengambil event booking untuk grid pada store & tanggal tertentu.
func GetForDashboard(storeID, date string) ([]models.EventBooking, error) {
	repository.BatchUpdateStatus(storeID, computeStatus)
	bookings, err := repository.FindForDashboard(storeID, date)
	if err != nil {
		return nil, err
	}
	for i := range bookings {
		bookings[i].Status = computeStatus(bookings[i])
	}
	return bookings, nil
}

// ── Pricing ───────────────────────────────────────────────────────────────────

// GetEventPrice mengambil harga event untuk store.
// Mengembalikan struct kosong (bukan error) jika belum dikonfigurasi.
func GetEventPrice(storeID string) (*models.StoreEventPrice, error) {
	price, err := repository.GetEventPrice(storeID)
	if err != nil {
		return &models.StoreEventPrice{StoreID: storeID, PricePerDay: 0}, nil
	}
	return price, nil
}

// UpsertEventPrice menyimpan harga event untuk store.
func UpsertEventPrice(storeID string, pricePerDay float64, updatedBy string) (*models.StoreEventPrice, error) {
	if pricePerDay < 0 {
		return nil, errors.New("harga tidak boleh negatif")
	}
	return repository.UpsertEventPrice(storeID, pricePerDay, updatedBy)
}

// PreviewPrice menghitung preview harga event sebelum booking dibuat.
func PreviewPrice(storeID, startTime, endTime string) (map[string]interface{}, error) {
	startMins := utils.MinsOf(startTime)
	endMins := utils.MinsOf(endTime)
	if endMins <= startMins {
		endMins += 24 * 60
	}
	durationHours := float64(endMins-startMins) / 60.0
	isFullDay := false

	eventPrice, err := repository.GetEventPrice(storeID)
	if err != nil || eventPrice.PricePerDay == 0 {
		return nil, errors.New("harga event belum dikonfigurasi untuk cabang ini")
	}

	// cek hari ni weekend or
	var dayCategory string
	now := time.Now() // Pastikan timezone server sudah sesuai (misal WIB)

	switch now.Weekday() {
	case time.Monday, time.Tuesday, time.Wednesday, time.Thursday:
		dayCategory = "weekday"
	case time.Friday, time.Saturday, time.Sunday:
		dayCategory = "weekend"
	}

	// get data store sesuai store id & weekend/weekday nya
	storeOperatingHours, err := storeRepository.FindOperatingHourByStoreAndDay(storeID, dayCategory)
	if err != nil {
		return nil, errors.New("gagal mengambil data store")
	}

	if storeOperatingHours.OpenTime == startTime && storeOperatingHours.CloseTime == endTime {
		isFullDay = true
	} else {
		isFullDay = false
	}

	totalPrice := 0.0
	if isFullDay {
		totalPrice = eventPrice.PricePerDay
	} else {
		totalPrice = calculateTotalPrice(eventPrice.PricePerDay, durationHours)
	}

	return map[string]interface{}{
		"price_per_day":  eventPrice.PricePerDay,
		"duration_hours": durationHours,
		"total_price":    totalPrice,
	}, nil
}

var (
	// ErrEventPaymentNotFound: tidak ada event dengan invoice ini.
	ErrEventPaymentNotFound = errors.New("event booking untuk invoice ini tidak ditemukan")
	// ErrEventPaymentProcessed: webhook ulang untuk event yang sudah dibayar.
	ErrEventPaymentProcessed = errors.New("pembayaran event sudah diproses")
	// ErrEventPaidButSlotTaken: dibayar setelah event dibatalkan dan slot sudah
	// dipakai / event dibatalkan admin. Butuh refund manual.
	ErrEventPaidButSlotTaken = errors.New("pembayaran event diterima tetapi event tidak bisa diaktifkan — perlu refund manual")
)

// ConfirmCustomerPayment menandai event booking customer sebagai lunas.
//
// Idempotent: baris event dikunci (SELECT ... FOR UPDATE), jadi webhook ganda
// diproses bergantian dan yang kedua mendapat ErrEventPaymentProcessed.
// Jika event sudah di-cancel otomatis karena telat bayar, event dihidupkan lagi
// selama slotnya masih kosong; selain itu dicatat sebagai orphan.
func ConfirmCustomerPayment(invoiceID string) error {
	return config.DB.Transaction(func(tx *gorm.DB) error {
		var e models.EventBooking
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("xendit_invoice_id = ?", invoiceID).First(&e).Error; err != nil {
			return ErrEventPaymentNotFound
		}
		if e.PaymentStatus != nil && *e.PaymentStatus == "paid" {
			return ErrEventPaymentProcessed
		}

		updates := map[string]interface{}{"payment_status": "paid"}
		if e.Status == "cancelled" {
			date := e.BookingDate.Format("2006-01-02")
			byAdmin := e.CancelledBy == nil || *e.CancelledBy != repository.CancelledBySystem
			regular, err1 := repository.CheckOverlapWithRegular(e.StoreID, date, e.StartTime, e.EndTime)
			event, err2 := repository.CheckOverlapWithEvent(e.StoreID, date, e.StartTime, e.EndTime, e.ID)
			if err := errors.Join(err1, err2); err != nil {
				return err
			}
			if byAdmin || regular || event {
				log.Printf("[PAYMENT ORPHAN] event=%s invoice=%s customer=%v amount=%.0f: %v",
					e.ID, invoiceID, e.CustomerID, e.TotalPrice, ErrEventPaidButSlotTaken)
				return ErrEventPaidButSlotTaken
			}
			updates["status"] = "upcoming"
			updates["cancel_reason"] = nil
			updates["cancelled_at"] = nil
			updates["cancelled_by"] = nil
		}
		return tx.Model(&models.EventBooking{}).Where("id = ?", e.ID).Updates(updates).Error
	})
}

// EventQuote = harga event customer sebelum booking dibuat.
type EventQuote struct {
	StoreID       string  `json:"store_id"`
	BookingDate   string  `json:"booking_date"`
	StartTime     string  `json:"start_time"`
	EndTime       string  `json:"end_time"`
	DurationHours float64 `json:"duration_hours"`
	PricePerDay   float64 `json:"price_per_day"`
	TotalPrice    float64 `json:"total_price"`
	Available     bool    `json:"available"`
}

// QuoteCustomerEvent menghitung harga event customer tanpa membuat booking.
// Customer selalu booking per jam (lihat customer_app.InitiateEventBooking), jadi
// harga = calculateTotalPrice, sama persis dengan yang ditagih Create.
// Available = tidak bentrok dengan booking reguler maupun event lain saat ini.
func QuoteCustomerEvent(storeID, date, startTime, endTime string) (*EventQuote, error) {
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return nil, errors.New("format booking_date tidak valid (YYYY-MM-DD)")
	}
	if date < time.Now().In(utils.JakartaLoc()).Format("2006-01-02") {
		return nil, errors.New("tanggal event sudah lewat")
	}
	durationHours, err := eventDuration(startTime, endTime)
	if err != nil {
		return nil, err
	}
	eventPrice, err := configuredEventPrice(storeID)
	if err != nil {
		return nil, err
	}

	regular, err := repository.CheckOverlapWithRegular(storeID, date, startTime, endTime)
	if err != nil {
		return nil, err
	}
	event, err := repository.CheckOverlapWithEvent(storeID, date, startTime, endTime, "")
	if err != nil {
		return nil, err
	}

	return &EventQuote{
		StoreID:       storeID,
		BookingDate:   date,
		StartTime:     startTime[:5],
		EndTime:       endTime[:5],
		DurationHours: durationHours,
		PricePerDay:   eventPrice.PricePerDay,
		TotalPrice:    calculateTotalPrice(eventPrice.PricePerDay, durationHours),
		Available:     !regular && !event,
	}, nil
}
