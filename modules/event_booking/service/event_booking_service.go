package service

import (
	"encoding/json"
	"errors"
	"math"
	"time"

	"game_lounge_be/config"
	"game_lounge_be/models"
	"game_lounge_be/modules/event_booking/dto"
	"game_lounge_be/modules/event_booking/repository"
	storeRepository "game_lounge_be/modules/store/repository"

	"github.com/google/uuid"
)

// ── Helpers ───────────────────────────────────────────────────────────────────

func parseMins(t string) int {
	if len(t) < 5 {
		return 0
	}
	h := int(t[0]-'0')*10 + int(t[1]-'0')
	m := int(t[3]-'0')*10 + int(t[4]-'0')
	return h*60 + m
}

// computeStatus menghitung status event booking berdasarkan waktu Jakarta.
func computeStatus(b models.EventBooking) string {
	if b.Status == "cancelled" || b.Status == "completed" {
		return b.Status
	}
	jakartaLoc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		jakartaLoc = time.UTC
	}
	now := time.Now().In(jakartaLoc)
	startMins := parseMins(b.StartTime)
	endMins := parseMins(b.EndTime)
	if endMins <= startMins {
		endMins += 24 * 60
	}
	nowMins := now.Hour()*60 + now.Minute()
	today := now.Format("2006-01-02")
	bDate := b.BookingDate.Format("2006-01-02")

	if bDate < today {
		return "completed"
	}
	if bDate == today {
		if nowMins >= endMins {
			return "completed"
		}
		if nowMins >= startMins {
			return "ongoing"
		}
	}
	return "upcoming"
}

// calculateTotalPrice menghitung harga proporsional.
// Formula: round((price_per_day / 24 × duration_hours) / 1000) × 1000
// Dibulatkan ke Rp 1.000 terdekat.
func calculateTotalPrice(pricePerDay, durationHours float64) float64 {
	raw := pricePerDay / 24 * durationHours
	return math.Round(raw/1000) * 1000
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
	endTime   := req.EndTime

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

	// Hitung durasi
	startMins := parseMins(startTime)
	endMins   := parseMins(endTime)
	if endMins <= startMins {
		endMins += 24 * 60
	}
	durationHours := float64(endMins-startMins) / 60.0

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
	eventPrice, err := repository.GetEventPrice(req.StoreID)
	if err != nil || eventPrice.PricePerDay == 0 {
		return nil, errors.New("harga event untuk cabang ini belum dikonfigurasi. Silakan atur di menu Pricing")
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
	repository.BatchUpdateStatus(storeID, date)
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
	startMins := parseMins(startTime)
	endMins   := parseMins(endTime)
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

	if storeOperatingHours.OpenTime == startTime && storeOperatingHours.CloseTime == endTime{
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
