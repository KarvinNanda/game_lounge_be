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
	customerBookingRepo "game_lounge_be/modules/customer_booking/repository"
	ntService "game_lounge_be/modules/notification_template/service"
	creditsRepo "game_lounge_be/modules/play_credits/repository"
	pricingDto "game_lounge_be/modules/pricing/dto"
	pricingService "game_lounge_be/modules/pricing/service"
	storeService "game_lounge_be/modules/store/service"
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

// computeStatus menghitung status booking dari waktu sekarang (WIB).
func computeStatus(b models.Booking) string {
	return computeStatusAt(b, storeOpenMins(b.StoreID, b.BookingDate), time.Now())
}

// computeStatusAt = computeStatus dengan jam buka & waktu "sekarang" eksplisit (untuk test).
// Sesi dihitung absolut (lihat utils.SessionStatus) supaya booking lintas tengah
// malam, mis. 23:00–02:00, tetap "ongoing" sampai 02:00 hari berikutnya.
func computeStatusAt(b models.Booking, openMins int, now time.Time) string {
	return utils.SessionStatus(b.Status, b.BookingDate, b.StartTime, b.EndTime, openMins, now)
}

// storeOpenMins = jam buka store (menit) pada tanggal tersebut.
func storeOpenMins(storeID string, date time.Time) int {
	open, _ := customerBookingRepo.OperatingWindow(storeID, date.Format("2006-01-02"))
	return open
}

// openCache menyimpan jam buka per store+tanggal selama 1 request list,
// supaya status N booking tidak memicu N query jam operasional.
type openCache map[string]int

func (oc openCache) get(storeID string, date time.Time) int {
	key := storeID + "|" + date.Format("2006-01-02")
	if v, ok := oc[key]; ok {
		return v
	}
	v := storeOpenMins(storeID, date)
	oc[key] = v
	return v
}

func enrichBooking(b models.Booking) BookingWithComputed {
	return enrichBookingWith(b, openCache{}, time.Now())
}

func enrichBookingWith(b models.Booking, oc openCache, now time.Time) BookingWithComputed {
	open := oc.get(b.StoreID, b.BookingDate)
	b.Status = computeStatusAt(b, open, now)
	_, end := utils.SessionWindow(b.BookingDate, b.StartTime, b.EndTime, open)
	isEndingSoon := b.Status == "ongoing" && end.Sub(now) <= 30*time.Minute
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
	oc, now := openCache{}, time.Now()
	for _, b := range bookings {
		result = append(result, enrichBookingWith(b, oc, now))
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

	// Durasi selalu dihitung server — duration_hours dari request diabaikan.
	durationHours, err := durationFromRange(req.StartTime, req.EndTime)
	if err != nil {
		return nil, err
	}

	// 2. Cek overlap awal (dicek ulang di dalam transaction saat menyimpan).
	hasOverlap, err := repository.CheckOverlap(req.RoomID, req.StoreID, req.BookingDate, req.StartTime, req.EndTime, "")
	if err != nil {
		return nil, errors.New("gagal mengecek ketersediaan slot")
	}
	if hasOverlap {
		return nil, repository.ErrSlotTaken
	}

	// 3. Room wajib milik store ini — kalau tidak, harga bisa diambil dari store lain.
	roomTemplateID, err := repository.FindRoomTemplateInStore(req.RoomID, req.StoreID)
	if err != nil {
		return nil, errors.New("ruangan tidak ditemukan di cabang ini")
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

	// 5. Validasi voucher (opsional). Kode yang tidak valid → error, bukan
	// diam-diam diabaikan. Cek member ada di ValidateVoucher.
	if req.VoucherCode != "" {
		if req.CustomerID == "" {
			return nil, errors.New("voucher hanya bisa dipakai oleh customer terdaftar")
		}
		vResult, vErr := voucherService.ValidateVoucher(voucherDto.ValidateVoucherRequest{
			Code:           req.VoucherCode,
			CustomerID:     req.CustomerID,
			StoreID:        req.StoreID,
			Amount:         basePrice,
			UseType:        "booking",
			RoomTemplateID: roomTemplateID,
		})
		if vErr != nil {
			return nil, vErr
		}
		if !vResult.IsValid {
			return nil, errors.New(vResult.Message)
		}
		discountAmount = vResult.DiscountAmount
		voucherID = &vResult.VoucherID
		code := req.VoucherCode
		voucherCodeStr = &code
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
		if err := checkCreditUsable(credit, req.CustomerID, req.StoreID, durationHours, time.Now()); err != nil {
			return nil, err
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
		DurationHours:    durationHours,
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

	// 11. Simpan booking + potong credits + pakai voucher secara atomik.
	var deduct *repository.CreditDeduction
	if playCreditID != nil {
		deduct = &repository.CreditDeduction{CreditID: *playCreditID, Hours: durationHours}
	}
	var usage *models.VoucherUsage
	if voucherID != nil {
		usage = &models.VoucherUsage{VoucherID: *voucherID, CustomerID: req.CustomerID, DiscountAmount: discountAmount}
	}
	if err := repository.CreateBookingAtomic(booking, deduct, usage); err != nil {
		switch {
		case errors.Is(err, repository.ErrSlotTaken),
			errors.Is(err, repository.ErrCreditInsufficient),
			errors.Is(err, repository.ErrVoucherUsed):
			return nil, err
		}
		log.Printf("[Booking] gagal membuat booking: %v", err)
		return nil, errors.New("gagal membuat booking")
	}

	// 13. Kirim notifikasi ke customer (async)
	utils.SafeGo(func() { sendBookingNotification(booking) })

	return GetBookingByID(booking.ID)
}

// checkCancellable: hanya booking yang BELUM mulai yang boleh dibatalkan.
// Status di DB bisa masih "upcoming" untuk booking yang sudah lewat (status
// dihitung saat dibaca), jadi yang dipakai adalah status terhitung.
func checkCancellable(b models.Booking, openMins int, now time.Time) error {
	switch computeStatusAt(b, openMins, now) {
	case "upcoming":
		return nil
	case "cancelled":
		return errors.New("booking sudah dibatalkan")
	case "ongoing":
		return errors.New("booking sedang berjalan, tidak bisa dibatalkan")
	default:
		return errors.New("booking sudah selesai, tidak bisa dibatalkan")
	}
}

// isPaidOnline: booking dari customer app (dibuat oleh webhook Xendit) disimpan
// dengan payment_method "cash" dan created_by = customer_id.
func isPaidOnline(b *models.Booking) bool {
	return b.CreatedBy != nil && b.CustomerID != nil && *b.CreatedBy == *b.CustomerID
}

// CancelBooking membatalkan booking dengan alasan wajib. Pembatalan, pengembalian
// jam play credits, dan pelepasan voucher terjadi dalam 1 transaction.
func CancelBooking(id string, req dto.CancelBookingRequest, cancelledBy string) (*BookingWithComputed, error) {
	b, err := repository.FindBookingByID(id)
	if err != nil {
		return nil, errors.New("booking tidak ditemukan")
	}
	if err := checkCancellable(*b, storeOpenMins(b.StoreID, b.BookingDate), time.Now()); err != nil {
		return nil, err
	}

	if err := repository.CancelBookingAtomic(b, req.Reason, cancelledBy); err != nil {
		if errors.Is(err, repository.ErrAlreadyFinal) {
			return nil, errors.New("booking sudah dibatalkan atau selesai")
		}
		log.Printf("[Booking] gagal membatalkan %s: %v", b.ID, err)
		return nil, errors.New("gagal membatalkan booking")
	}
	if isPaidOnline(b) {
		// Refund Xendit belum otomatis — tandai di log untuk diproses manual.
		log.Printf("[REFUND MANUAL] booking %s (%s) dibatalkan; dibayar online Rp %.0f", b.BookingCode, b.ID, b.TotalPrice)
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
	Date        string          `json:"date"`
	StoreID     string          `json:"store_id"`
	OpenTime    string          `json:"open_time"`    // jam buka efektif
	CloseTime   string          `json:"close_time"`   // jam tutup efektif
	IsHoliday   bool            `json:"is_holiday"`   // apakah hari libur
	HolidayName string          `json:"holiday_name"` // nama hari libur
	HolidayType string          `json:"holiday_type"` // "global" | "store" | ""
	Rooms       []DashboardRoom `json:"rooms"`
}

// GetDashboard mengambil data untuk render kalender grid.
func GetDashboard(filter dto.DashboardFilter) (*DashboardData, error) {
	// Update status terlebih dahulu
	_ = repository.BatchUpdateStatus(filter.StoreID)

	// Ambil jam operasional efektif (cek global holiday + store holiday)
	openTime := "10:00:00"
	closeTime := "02:00:00"
	isHoliday := false
	holidayName := ""
	holidayType := ""

	if date, err := time.Parse("2006-01-02", filter.Date); err == nil {
		if opHours, err := storeService.GetEffectiveOperatingHours(filter.StoreID, date); err == nil && opHours != nil {
			openTime = opHours.OpenTime
			closeTime = opHours.CloseTime
			isHoliday = opHours.IsHoliday
			holidayName = opHours.HolidayName
			holidayType = opHours.HolidayType
		}
	}

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
	oc, now := openCache{}, time.Now()
	for _, b := range bookings {
		enriched := enrichBookingWith(b, oc, now)
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
		Date:        filter.Date,
		StoreID:     filter.StoreID,
		OpenTime:    openTime,
		CloseTime:   closeTime,
		IsHoliday:   isHoliday,
		HolidayName: holidayName,
		HolidayType: holidayType,
		Rooms:       dashRooms,
	}, nil
}

// GetSessionsEndingSoon mengambil sesi yang akan berakhir dalam 5 menit.
func GetSessionsEndingSoon(storeID string) ([]BookingWithComputed, error) {
	bookings, err := repository.FindSessionsEndingSoon(storeID, 5)
	if err != nil {
		return nil, err
	}
	result := make([]BookingWithComputed, 0, len(bookings))
	oc, now := openCache{}, time.Now()
	for _, b := range bookings {
		result = append(result, enrichBookingWith(b, oc, now))
	}
	return result, nil
}

// GetAvailableCredits mengambil play credits yang tersedia untuk customer di store tertentu.
func GetAvailableCredits(customerID, storeID,bookingDate string, durationHours float64) ([]models.CustomerPlayCredit, error) {
	return repository.FindAvailableCreditsForBooking(customerID, storeID,bookingDate, durationHours)
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
