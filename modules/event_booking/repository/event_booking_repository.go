package repository

import (
	"time"

	"game_lounge_be/config"
	"game_lounge_be/models"
	"game_lounge_be/utils"

	"gorm.io/gorm"
)

// EventPaymentWindow = waktu customer untuk membayar event booking. Invoice Xendit
// diberi durasi yang sama. Setelah lewat, event yang belum dibayar tidak lagi
// memblokir jadwal dan di-cancel otomatis oleh CancelExpiredUnpaid.
const EventPaymentWindow = 30 * time.Minute

// Blocking membatasi query ke event yang benar-benar memblokir jadwal:
// tidak cancelled, dan (event admin / sudah dibayar / masih dalam jendela bayar).
// Tanpa ini customer bisa membuat event full-venue tanpa membayar dan
// memblokir seluruh store tanpa batas.
func Blocking(db *gorm.DB) *gorm.DB {
	return db.Where("status != 'cancelled'").
		Where("(payment_status IS NULL OR payment_status = 'paid' OR (payment_status = 'pending_payment' AND created_at >= ?))",
			time.Now().Add(-EventPaymentWindow))
}

// CancelExpiredUnpaid meng-cancel event customer yang tidak dibayar dalam jendela bayar.
func CancelExpiredUnpaid() (int64, error) {
	res := config.DB.Model(&models.EventBooking{}).
		Where("payment_status = 'pending_payment' AND status != 'cancelled' AND created_at < ?",
			time.Now().Add(-EventPaymentWindow)).
		Updates(map[string]interface{}{
			"status":        "cancelled",
			"cancel_reason": CancelReasonUnpaid,
			"cancelled_at":  time.Now(),
			"cancelled_by":  CancelledBySystem,
		})
	return res.RowsAffected, res.Error
}

const (
	CancelReasonUnpaid = "Pembayaran tidak diterima dalam batas waktu"
	CancelledBySystem  = "system"
)

// ── Event Booking CRUD ────────────────────────────────────────────────────────

// FindAll mengambil list event booking dengan filter.
func FindAll(storeID, status, dateFrom, dateTo string, page, perPage int) ([]models.EventBooking, int64, error) {
	var bookings []models.EventBooking
	var total int64

	q := config.DB.Model(&models.EventBooking{}).Preload("Store")
	if storeID != "" {
		q = q.Where("store_id = ?", storeID)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if dateFrom != "" {
		q = q.Where("booking_date >= ?", dateFrom)
	}
	if dateTo != "" {
		q = q.Where("booking_date <= ?", dateTo)
	}

	q.Count(&total)
	err := q.Offset((page-1)*perPage).Limit(perPage).
		Order("booking_date DESC, start_time DESC").Find(&bookings).Error

	return bookings, total, err
}

// FindByID mengambil satu event booking berdasarkan ID.
func FindByID(id string) (*models.EventBooking, error) {
	var b models.EventBooking
	err := config.DB.Preload("Store").Where("id = ?", id).First(&b).Error
	return &b, err
}

// Create menyimpan event booking baru.
func Create(b *models.EventBooking) error {
	return config.DB.Create(b).Error
}

// Update menyimpan perubahan event booking.
func Update(b *models.EventBooking) error {
	return config.DB.Save(b).Error
}

// FindForDashboard mengambil semua event booking aktif untuk grid pada tanggal & store tertentu.
func FindForDashboard(storeID, date string) ([]models.EventBooking, error) {
	var bookings []models.EventBooking
	err := Blocking(config.DB).
		Where("store_id = ? AND booking_date = ?", storeID, date).
		Find(&bookings).Error
	return bookings, err
}

// ── Overlap Check ─────────────────────────────────────────────────────────────

// CheckOverlapWithRegular mengecek apakah ada regular booking yang conflict dengan event booking.
// Jam dibandingkan sebagai waktu absolut per hari operasional (utils.SessionWindow),
// sehingga booking 00:00–01:00 pada tanggal yang sama ikut terhitung untuk event
// 20:00–02:00. Booking tanggal sebelum/sesudahnya ikut dicek untuk sesi yang melewati jam buka.
func CheckOverlapWithRegular(storeID, date, startTime, endTime string) (bool, error) {
	from, to, dates, err := eventWindow(storeID, date, startTime, endTime)
	if err != nil {
		return false, err
	}

	var bookings []models.Booking
	if err := config.DB.Where("store_id = ? AND booking_date IN ? AND status != 'cancelled'",
		storeID, dates).Find(&bookings).Error; err != nil {
		return false, err
	}

	opens := map[string]int{}
	for _, b := range bookings {
		if windowsOverlap(from, to, storeID, b.BookingDate, b.StartTime, b.EndTime, opens) {
			return true, nil
		}
	}
	return false, nil
}

// CheckOverlapWithEvent mengecek apakah ada event booking lain yang conflict.
// Perbandingan waktu sama dengan CheckOverlapWithRegular.
func CheckOverlapWithEvent(storeID, date, startTime, endTime, excludeID string) (bool, error) {
	from, to, dates, err := eventWindow(storeID, date, startTime, endTime)
	if err != nil {
		return false, err
	}

	var events []models.EventBooking
	q := Blocking(config.DB).Where("store_id = ? AND booking_date IN ?", storeID, dates)
	if excludeID != "" {
		q = q.Where("id != ?", excludeID)
	}
	if err := q.Find(&events).Error; err != nil {
		return false, err
	}

	opens := map[string]int{}
	for _, e := range events {
		if windowsOverlap(from, to, storeID, e.BookingDate, e.StartTime, e.EndTime, opens) {
			return true, nil
		}
	}
	return false, nil
}

// eventWindow menghitung waktu absolut sesi pada tanggal operasional date, beserta
// tanggal (kemarin, hari itu, besok) yang sesinya bisa bersinggungan.
func eventWindow(storeID, date, startTime, endTime string) (from, to time.Time, dates []string, err error) {
	d, err := time.Parse("2006-01-02", date)
	if err != nil {
		return from, to, nil, err
	}
	open, _ := OperatingWindow(storeID, date)
	from, to = utils.SessionWindow(d, startTime, endTime, open)
	for _, off := range []int{-1, 0, 1} {
		dates = append(dates, d.AddDate(0, 0, off).Format("2006-01-02"))
	}
	return from, to, dates, nil
}

// windowsOverlap: apakah sesi (date, start–end) bersinggungan dengan [from, to).
// opens = cache jam buka per tanggal selama 1 pengecekan.
func windowsOverlap(from, to time.Time, storeID string, date time.Time, start, end string, opens map[string]int) bool {
	key := date.Format("2006-01-02")
	open, ok := opens[key]
	if !ok {
		open, _ = OperatingWindow(storeID, key)
		opens[key] = open
	}
	s, e := utils.SessionWindow(date, start, end, open)
	return from.Before(e) && to.After(s)
}

// OperatingWindow mengembalikan jam buka & tutup (menit dari 00:00) untuk store
// pada tanggal tersebut. closeMins > 24*60 jika store tutup setelah tengah malam.
// Ada di package ini (bukan customer_booking) karena customer_booking/repository
// sudah meng-import package ini.
func OperatingWindow(storeID, date string) (openMins, closeMins int) {
	dayType := "weekday"
	if d, err := time.Parse("2006-01-02", date); err == nil {
		if d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
			dayType = "weekend"
		}
	}

	openTime, closeTime := "10:00", "02:00"
	var opHours models.StoreOperatingHour
	config.DB.Where(
		"store_id = ? AND day_type = ? AND is_active = true AND deleted_at IS NULL",
		storeID, dayType,
	).First(&opHours)
	if opHours.ID != 0 {
		openTime, closeTime = opHours.OpenTime, opHours.CloseTime
	}

	openMins, closeMins = utils.MinsOf(openTime), utils.MinsOf(closeTime)
	if closeMins <= openMins {
		closeMins += 24 * 60 // operasional melewati tengah malam
	}
	return openMins, closeMins
}

// ── Status Auto-Update ────────────────────────────────────────────────────────

// BatchUpdateStatus menyinkronkan status event di DB dengan waktu sekarang.
// statusAt dihitung pemanggil (service) dari waktu absolut sesi.
// Dulu memakai tanggal yang dibuka di dashboard sebagai "hari ini", sehingga
// membuka dashboard minggu depan menandai event minggu ini "completed".
func BatchUpdateStatus(storeID string, statusAt func(models.EventBooking) string) {
	today := time.Now().In(jakartaLocation()).Format("2006-01-02")
	var events []models.EventBooking
	config.DB.Select("id", "store_id", "booking_date", "start_time", "end_time", "status").
		Where("store_id = ? AND status IN ('upcoming','ongoing') AND booking_date <= ?", storeID, today).
		Find(&events)
	for _, e := range events {
		if next := statusAt(e); next != e.Status {
			config.DB.Model(&models.EventBooking{}).Where("id = ? AND status = ?", e.ID, e.Status).
				Update("status", next)
		}
	}
}

func jakartaLocation() *time.Location {
	if loc, err := time.LoadLocation("Asia/Jakarta"); err == nil {
		return loc
	}
	return time.FixedZone("WIB", 7*60*60)
}

// ── Event Pricing ─────────────────────────────────────────────────────────────

// GetEventPrice mengambil harga event per hari untuk store.
func GetEventPrice(storeID string) (*models.StoreEventPrice, error) {
	var price models.StoreEventPrice
	err := config.DB.Where("store_id = ?", storeID).First(&price).Error
	return &price, err
}

// UpsertEventPrice menyimpan harga event untuk store (insert or update).
func UpsertEventPrice(storeID string, pricePerDay float64, updatedBy string) (*models.StoreEventPrice, error) {
	var price models.StoreEventPrice
	err := config.DB.Where("store_id = ?", storeID).First(&price).Error
	if err != nil {
		// Belum ada → create
		price = models.StoreEventPrice{
			StoreID:     storeID,
			PricePerDay: pricePerDay,
			CreatedBy:   &updatedBy,
		}
		err = config.DB.Create(&price).Error
	} else {
		// Sudah ada → update
		price.PricePerDay = pricePerDay
		price.UpdatedBy   = &updatedBy
		err = config.DB.Save(&price).Error
	}
	return &price, err
}
