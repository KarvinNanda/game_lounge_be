package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"game_lounge_be/models"
	"game_lounge_be/modules/booking/dto"
	"game_lounge_be/modules/booking/repository"
	customerRepo "game_lounge_be/modules/customer/repository"
	ntService "game_lounge_be/modules/notification_template/service"
	creditsRepo "game_lounge_be/modules/play_credits/repository"
	pricingDto "game_lounge_be/modules/pricing/dto"
	pricingService "game_lounge_be/modules/pricing/service"
	voucherDto "game_lounge_be/modules/voucher/dto"
	voucherService "game_lounge_be/modules/voucher/service"
	"game_lounge_be/utils"

	"github.com/google/uuid"
)

// jakartaLoc timezone WIB.
var jakartaLoc *time.Location

func init() {
	var err error
	jakartaLoc, err = time.LoadLocation("Asia/Jakarta")
	if err != nil {
		jakartaLoc = time.UTC
	}
}

// BookingWithComputed adalah booking dengan flag computed tambahan.
type BookingWithComputed struct {
	models.Booking
	IsEndingSoon bool `json:"is_ending_soon"` // true jika sisa < 30 menit
}

// ── Status Computation ────────────────────────────────────────────────────────

// computeStatus menghitung status booking berdasarkan waktu Jakarta saat ini.
func computeStatus(b models.Booking) string {
	if b.Status == "cancelled" || b.Status == "completed" {
		return b.Status
	}
	now := time.Now().In(jakartaLoc)
	today := now.Format("2006-01-02")
	bookingDate := b.BookingDate.Format("2006-01-02")

	startMins := parseTimeToMins(b.StartTime)
	endMins := parseTimeToMins(b.EndTime)
	if endMins <= startMins {
		endMins += 24 * 60
	}
	nowMins := now.Hour()*60 + now.Minute()

	if bookingDate < today {
		return "completed"
	}
	if bookingDate == today {
		if nowMins >= endMins {
			return "completed"
		}
		if nowMins >= startMins {
			return "ongoing"
		}
	}
	return "upcoming"
}

func parseTimeToMins(t string) int {
	if len(t) < 5 {
		return 0
	}
	h := int(t[0]-'0')*10 + int(t[1]-'0')
	m := int(t[3]-'0')*10 + int(t[4]-'0')
	return h*60 + m
}

func enrichBooking(b models.Booking) BookingWithComputed {
	b.Status = computeStatus(b)
	now := time.Now().In(jakartaLoc)
	endMins := parseTimeToMins(b.EndTime)
	nowMins := now.Hour()*60 + now.Minute()
	isEndingSoon := b.Status == "ongoing" && (endMins-nowMins) <= 30
	return BookingWithComputed{Booking: b, IsEndingSoon: isEndingSoon}
}

// ── Booking CRUD ──────────────────────────────────────────────────────────────

// GetAllBookings mengambil list booking dengan filter.
func GetAllBookings(filter dto.BookingFilter) ([]BookingWithComputed, int64, error) {
	bookings, total, err := repository.FindAllBookings(
		filter.StoreID, filter.RoomID, filter.Status,
		filter.DateFrom, filter.DateTo, filter.Search,
		filter.Page, filter.PerPage,
	)
	if err != nil {
		return nil, 0, err
	}

	result := make([]BookingWithComputed, 0, len(bookings))
	for _, b := range bookings {
		result = append(result, enrichBooking(b))
	}
	return result, total, nil
}

// GetBookingByID mengambil detail satu booking.
func GetBookingByID(id string) (*BookingWithComputed, error) {
	b, err := repository.FindBookingByID(id)
	if err != nil {
		return nil, errors.New("booking tidak ditemukan")
	}
	r := enrichBooking(*b)
	return &r, nil
}

// CreateBooking membuat booking baru dengan validasi lengkap.
func CreateBooking(req dto.CreateBookingRequest, createdBy string) (*BookingWithComputed, error) {
	// 1. Parse tanggal
	bookingDate, err := time.Parse("2006-01-02", req.BookingDate)
	if err != nil {
		return nil, errors.New("format booking_date tidak valid (YYYY-MM-DD)")
	}

	// 2. Cek overlap — tidak boleh ada booking lain di room yg sama pada slot yg sama
	hasOverlap, err := repository.CheckOverlap(req.RoomID, req.BookingDate, req.StartTime, req.EndTime, "")
	if err != nil {
		return nil, errors.New("gagal mengecek ketersediaan slot")
	}
	if hasOverlap {
		return nil, errors.New("slot waktu sudah terisi oleh booking lain")
	}

	// 3. Ambil room template ID untuk pricing engine
	roomTemplateID, err := repository.GetRoomTemplateID(req.RoomID)
	if err != nil {
		return nil, errors.New("ruangan tidak ditemukan")
	}

	// 4. Hitung harga dari pricing engine (FinalPrice sudah include flash discount)
	calcResult, err := pricingService.CalculatePrice(pricingDto.CalculatePriceRequest{
		StoreID:        req.StoreID,
		RoomTemplateID: roomTemplateID,
		BookingDate:    req.BookingDate,
		StartTime:      req.StartTime,
		EndTime:        req.EndTime,
	})
	if err != nil {
		return nil, fmt.Errorf("gagal menghitung harga: %v", err)
	}

	// basePrice = FinalPrice (sudah include flash discount jika ada)
	basePrice := calcResult.FinalPrice
	discountAmount := 0.0
	var voucherID *string
	var voucherCodeStr *string

	// 5. Validasi voucher (opsional, hanya untuk member)
	if req.VoucherCode != "" && req.CustomerID != "" {
		customer, cErr := customerRepo.FindCustomerByID(req.CustomerID)
		if cErr == nil && customer.Type == "member" {
			vResult, vErr := voucherService.ValidateVoucher(voucherDto.ValidateVoucherRequest{
				Code:       req.VoucherCode,
				CustomerID: req.CustomerID,
				StoreID:    req.StoreID,
				Amount:     basePrice,
				UseType:    "booking",
			})
			if vErr == nil && vResult.IsValid {
				discountAmount = vResult.DiscountAmount
				voucherID = &vResult.VoucherID
				code := req.VoucherCode
				voucherCodeStr = &code
			}
		}
	}

	// 6. Validasi play credits (jika payment_method = 'play_credits')
	if req.PaymentMethod == "play_credits" {
		if req.PlayCreditID == "" {
			return nil, errors.New("play_credit_id wajib diisi untuk pembayaran credits")
		}
		credit, cErr := creditsRepo.FindCreditByID(req.PlayCreditID)
		if cErr != nil {
			return nil, errors.New("play credits tidak ditemukan")
		}
		if credit.RemainingHours < req.DurationHours {
			return nil, fmt.Errorf("sisa jam credits (%.1f jam) tidak cukup untuk booking ini (%.1f jam)",
				credit.RemainingHours, req.DurationHours)
		}
		if time.Now().After(credit.ExpiresAt) {
			return nil, errors.New("play credits sudah kadaluwarsa")
		}
	}

	// 7. Generate Booking Code (atomik melalui sequence)
	bookingCode, err := repository.GenerateBookingCode()
	if err != nil {
		return nil, errors.New("gagal generate booking code")
	}

	// 8. Serialize price breakdown sebagai JSON
	breakdownJSON, _ := json.Marshal(calcResult)
	breakdownStr := string(breakdownJSON)

	totalPrice := basePrice - discountAmount
	if totalPrice < 0 {
		totalPrice = 0
	}

	// 9. Siapkan pointer fields
	var customerID, playCreditID *string
	var customerWA, customerEmail *string
	var notes *string

	if req.CustomerID != "" {
		customerID = &req.CustomerID
	}
	if req.PlayCreditID != "" && req.PaymentMethod == "play_credits" {
		playCreditID = &req.PlayCreditID
	}
	if req.CustomerWhatsapp != "" {
		customerWA = &req.CustomerWhatsapp
	}
	if req.CustomerEmail != "" {
		customerEmail = &req.CustomerEmail
	}
	if req.Notes != "" {
		notes = &req.Notes
	}

	paymentMethod := req.PaymentMethod
	if paymentMethod == "" {
		paymentMethod = "cash"
	}

	// 10. Buat booking
	booking := &models.Booking{
		ID:               uuid.NewString(),
		BookingCode:      bookingCode,
		StoreID:          req.StoreID,
		RoomID:           req.RoomID,
		CustomerID:       customerID,
		CustomerName:     req.CustomerName,
		CustomerWhatsapp: customerWA,
		CustomerEmail:    customerEmail,
		BookingDate:      bookingDate,
		StartTime:        req.StartTime,
		EndTime:          req.EndTime,
		DurationHours:    req.DurationHours,
		PriceBreakdown:   &breakdownStr,
		BasePrice:        basePrice,
		DiscountAmount:   discountAmount,
		TotalPrice:       totalPrice,
		PaymentMethod:    paymentMethod,
		PlayCreditID:     playCreditID,
		VoucherID:        voucherID,
		VoucherCode:      voucherCodeStr,
		Status:           "upcoming",
		Notes:            notes,
		CreatedBy:        &createdBy,
	}

	if err := repository.CreateBooking(booking); err != nil {
		return nil, errors.New("gagal membuat booking")
	}

	// 11. Deduct play credits (goroutine)
	if req.PaymentMethod == "play_credits" && req.PlayCreditID != "" {
		creditID := req.PlayCreditID
		duration := req.DurationHours
		go func() {
			if err := creditsRepo.DeductHours(creditID, duration); err != nil {
				log.Printf("[Booking] Gagal deduct credits %s: %v", creditID, err)
			}
		}()
	}

	// 12. Redeem voucher jika dipakai
	if voucherID != nil && req.CustomerID != "" {
		vid := *voucherID
		cid := req.CustomerID
		bid := booking.ID
		disc := discountAmount
		go func() {
			if err := voucherService.RedeemVoucher(vid, cid, bid, disc); err != nil {
				log.Printf("[Booking] Gagal redeem voucher %s: %v", vid, err)
			}
		}()
	}

	// 13. Kirim notifikasi ke customer (async)
	go sendBookingNotification(booking)

	return GetBookingByID(booking.ID)
}

// CancelBooking membatalkan booking dengan alasan wajib.
func CancelBooking(id string, req dto.CancelBookingRequest, cancelledBy string) (*BookingWithComputed, error) {
	b, err := repository.FindBookingByID(id)
	if err != nil {
		return nil, errors.New("booking tidak ditemukan")
	}
	if b.Status == "cancelled" {
		return nil, errors.New("booking sudah dibatalkan")
	}
	if b.Status == "completed" {
		return nil, errors.New("booking sudah selesai, tidak bisa dibatalkan")
	}

	now := time.Now()
	b.Status = "cancelled"
	b.CancelReason = &req.Reason
	b.CancelledAt = &now
	b.CancelledBy = &cancelledBy

	// Kembalikan jam play credits jika booking pakai credits
	if b.PaymentMethod == "play_credits" && b.PlayCreditID != nil {
		creditID := *b.PlayCreditID
		duration := b.DurationHours
		go func() {
			// Deduct negatif = kembalikan jam
			if err := creditsRepo.DeductHours(creditID, -duration); err != nil {
				log.Printf("[Booking] Gagal refund credits %s: %v", creditID, err)
			}
		}()
	}

	if err := repository.UpdateBooking(b); err != nil {
		return nil, errors.New("gagal membatalkan booking")
	}
	return GetBookingByID(b.ID)
}

// CompleteBooking menandai booking sebagai selesai (manual oleh admin).
func CompleteBooking(id string, updatedBy string) (*BookingWithComputed, error) {
	b, err := repository.FindBookingByID(id)
	if err != nil {
		return nil, errors.New("booking tidak ditemukan")
	}
	if b.Status == "cancelled" {
		return nil, errors.New("booking sudah dibatalkan")
	}
	if b.Status == "completed" {
		return nil, errors.New("booking sudah selesai")
	}

	b.Status = "completed"
	b.UpdatedBy = &updatedBy

	if err := repository.UpdateBooking(b); err != nil {
		return nil, errors.New("gagal update status booking")
	}
	return GetBookingByID(b.ID)
}

// ── Dashboard ─────────────────────────────────────────────────────────────────

// DashboardRoom adalah room beserta list booking pada tanggal tersebut.
type DashboardRoom struct {
	models.StoreRoom
	Bookings []BookingWithComputed `json:"bookings"`
}

// DashboardData adalah response untuk kalender grid.
type DashboardData struct {
	Date    string          `json:"date"`
	StoreID string          `json:"store_id"`
	Rooms   []DashboardRoom `json:"rooms"`
}

// GetDashboard mengambil data untuk render kalender grid.
func GetDashboard(filter dto.DashboardFilter) (*DashboardData, error) {
	// Update status terlebih dahulu
	_ = repository.BatchUpdateStatus(filter.StoreID, filter.Date)

	rooms, err := repository.FindRoomsForStore(filter.StoreID)
	if err != nil {
		return nil, errors.New("gagal mengambil data ruangan")
	}

	bookings, err := repository.FindBookingsForGrid(filter.StoreID, filter.Date)
	if err != nil {
		return nil, errors.New("gagal mengambil data booking")
	}

	// Map bookings ke room-nya
	bookingMap := make(map[string][]BookingWithComputed)
	for _, b := range bookings {
		enriched := enrichBooking(b)
		bookingMap[b.RoomID] = append(bookingMap[b.RoomID], enriched)
	}

	dashRooms := make([]DashboardRoom, 0, len(rooms))
	for _, r := range rooms {
		if filter.RoomID != "" && r.ID != filter.RoomID {
			continue
		}
		booksForRoom := bookingMap[r.ID]
		if booksForRoom == nil {
			booksForRoom = []BookingWithComputed{}
		}
		dashRooms = append(dashRooms, DashboardRoom{
			StoreRoom: r,
			Bookings:  booksForRoom,
		})
	}

	return &DashboardData{
		Date:    filter.Date,
		StoreID: filter.StoreID,
		Rooms:   dashRooms,
	}, nil
}

// GetSessionsEndingSoon mengambil sesi yang akan berakhir dalam 30 menit.
func GetSessionsEndingSoon(storeID string) ([]BookingWithComputed, error) {
	bookings, err := repository.FindSessionsEndingSoon(storeID, 30)
	if err != nil {
		return nil, err
	}
	result := make([]BookingWithComputed, 0, len(bookings))
	for _, b := range bookings {
		result = append(result, enrichBooking(b))
	}
	return result, nil
}

// GetAvailableCredits mengambil play credits yang tersedia untuk customer di store tertentu.
func GetAvailableCredits(customerID, storeID string, durationHours float64) ([]models.CustomerPlayCredit, error) {
	return repository.FindAvailableCreditsForBooking(customerID, storeID, durationHours)
}

// ── Notification ──────────────────────────────────────────────────────────────

// sendBookingNotification mengirim konfirmasi booking.
// Prioritas: template dari DB → fallback ke HTML hardcode yang sudah didesain.
func sendBookingNotification(b *models.Booking) {
	if b.CustomerEmail == nil || *b.CustomerEmail == "" {
		// Tidak ada email, cek WA saja
		if b.CustomerWhatsapp != nil && *b.CustomerWhatsapp != "" {
			log.Printf("[Booking][WhatsApp placeholder] Kirim kode %s ke %s", b.BookingCode, *b.CustomerWhatsapp)
		}
		return
	}

	startTime := b.StartTime
	if len(startTime) >= 5 {
		startTime = startTime[:5]
	}
	endTime := b.EndTime
	if len(endTime) >= 5 {
		endTime = endTime[:5]
	}
	roomName := getRoomName(b.RoomID)

	subject := fmt.Sprintf("Booking Berhasil! Kode: %s", b.BookingCode)
	var htmlContent string

	// Coba ambil template dari DB
	tmpl, err := ntService.GetRendered("booking_confirmation", map[string]string{
		"nama_customer": b.CustomerName,
		"kode_booking":  b.BookingCode,
		"nama_ruangan":  roomName,
		"tanggal":       b.BookingDate.Format("02 January 2006"),
		"jam_mulai":     startTime,
		"jam_selesai":   endTime,
		"durasi":        fmt.Sprintf("%.0f", b.DurationHours),
		"total_harga":   fmt.Sprintf("%.0f", b.TotalPrice),
	})
	if err == nil && tmpl.IsEmailActive {
		// Template DB tersedia — kirim via SendEmail (plain text → HTML)
		if emailErr := utils.SendEmail(*b.CustomerEmail, b.CustomerName, tmpl.EmailSubject, tmpl.EmailBody); emailErr != nil {
			log.Printf("[Booking] Gagal kirim email %s: %v", b.BookingCode, emailErr)
		}
	} else {
		// Fallback: gunakan HTML hardcode yang sudah didesain
		htmlContent = utils.BuildBookingEmailHTML(
			b.BookingCode, b.CustomerName, roomName,
			b.BookingDate.Format("02 January 2006"),
			startTime, endTime,
			fmt.Sprintf("%.0f", b.DurationHours),
			fmt.Sprintf("%.0f", b.TotalPrice),
		)
		if emailErr := utils.SendHTMLEmail(*b.CustomerEmail, b.CustomerName, subject, htmlContent); emailErr != nil {
			log.Printf("[Booking] Gagal kirim email %s: %v", b.BookingCode, emailErr)
		}
	}

	if b.CustomerWhatsapp != nil && *b.CustomerWhatsapp != "" {
		waBody := ""
		if err == nil {
			waBody = tmpl.WhatsappBody
		}
		log.Printf("[Booking][WhatsApp placeholder] → %s: %s", *b.CustomerWhatsapp, waBody)
	}
}

// getRoomName mengambil nama ruangan dari DB berdasarkan room_id.
func getRoomName(roomID string) string {
	var name string
	repository.GetRoomNameByID(roomID, &name)
	return name
}
