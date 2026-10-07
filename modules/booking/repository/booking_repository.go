package repository

import (
	"errors"
	"sort"
	"fmt"
	"time"

	"game_lounge_be/config"
	"game_lounge_be/models"
	customerBookingRepo "game_lounge_be/modules/customer_booking/repository"
	eventRepo "game_lounge_be/modules/event_booking/repository"
	"game_lounge_be/utils"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ── Sequence ──────────────────────────────────────────────────────────────────

// EnsureSequenceExists memastikan baris sequence ada di DB (id=1).
func EnsureSequenceExists() {
	config.DB.Exec("INSERT IGNORE INTO booking_sequences (id, last_sequence) VALUES (1, 0)")
}

// GetNextSequence mengambil dan increment global counter secara atomik.
// LAST_INSERT_ID(expr) menyimpan nilai baru per-koneksi, dan transaction
// memastikan UPDATE & SELECT memakai koneksi yang sama. Versi lama membaca
// ulang tabel di koneksi lain, sehingga 2 request bisa mendapat nomor sama.
func GetNextSequence() (uint, error) {
	var seq uint
	err := config.DB.Transaction(func(tx *gorm.DB) error {
		res := tx.Exec("UPDATE booking_sequences SET last_sequence = LAST_INSERT_ID(last_sequence + 1) WHERE id = 1")
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			// Baris belum ada: buat lalu increment.
			if err := tx.Exec("INSERT IGNORE INTO booking_sequences (id, last_sequence) VALUES (1, 0)").Error; err != nil {
				return err
			}
			if err := tx.Exec("UPDATE booking_sequences SET last_sequence = LAST_INSERT_ID(last_sequence + 1) WHERE id = 1").Error; err != nil {
				return err
			}
		}
		return tx.Raw("SELECT LAST_INSERT_ID()").Scan(&seq).Error
	})
	return seq, err
}

// jakartaLoc mengembalikan timezone Asia/Jakarta.
func jakartaLoc() *time.Location {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		return time.UTC
	}
	return loc
}

// GenerateBookingCode membuat kode booking unik.
// Format: BK-{YYMMDD}-{4digit_global_sequence}
func GenerateBookingCode() (string, error) {
	seq, err := GetNextSequence()
	if err != nil {
		return "", err
	}
	dateStr := time.Now().In(jakartaLoc()).Format("060102")
	return fmt.Sprintf("BK-%s-%04d", dateStr, seq), nil
}

// ── Booking CRUD ──────────────────────────────────────────────────────────────

// FindAllBookings mengambil list booking dengan filter & pagination.
func FindAllBookings(storeID, roomID, status, dateFrom, dateTo, search string, page, perPage int) ([]models.Booking, int64, error) {
	var bookings []models.Booking
	var total int64

	query := config.DB.Model(&models.Booking{}).
		Preload("Store").
		Preload("Room.RoomTemplate")

	if storeID != "" {
		query = query.Where("store_id = ?", storeID)
	}
	if roomID != "" {
		query = query.Where("room_id = ?", roomID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if dateFrom != "" {
		query = query.Where("booking_date >= ?", dateFrom)
	}
	if dateTo != "" {
		query = query.Where("booking_date <= ?", dateTo)
	}
	if search != "" {
		like := "%" + search + "%"
		query = query.Where("customer_name LIKE ? OR booking_code LIKE ?", like, like)
	}

	query.Count(&total)
	err := query.Offset((page-1)*perPage).Limit(perPage).
		Order("booking_date DESC, start_time DESC").Find(&bookings).Error

	return bookings, total, err
}

// FindBookingByID mengambil booking berdasarkan ID dengan semua relasi.
func FindBookingByID(id string) (*models.Booking, error) {
	var booking models.Booking
	err := config.DB.
		Preload("Store").
		Preload("Room.RoomTemplate").
		Preload("Customer").
		Where("id = ?", id).First(&booking).Error
	return &booking, err
}

// FindBookingByCode mengambil booking berdasarkan kode.
func FindBookingByCode(code string) (*models.Booking, error) {
	var booking models.Booking
	err := config.DB.Preload("Room.RoomTemplate").
		Where("booking_code = ?", code).First(&booking).Error
	return &booking, err
}

// CreateBooking menyimpan booking baru.
func CreateBooking(booking *models.Booking) error {
	return config.DB.Create(booking).Error
}

// UpdateBooking menyimpan perubahan booking (status, cancel, dll).
func UpdateBooking(booking *models.Booking) error {
	return config.DB.Save(booking).Error
}

// ── Dashboard Grid ────────────────────────────────────────────────────────────

// FindBookingsForGrid mengambil semua booking untuk satu store pada satu tanggal.
func FindBookingsForGrid(storeID, date string) ([]models.Booking, error) {
	var bookings []models.Booking
	err := config.DB.
		Preload("Room.RoomTemplate").
		Where("store_id = ? AND booking_date = ? AND status != 'cancelled'", storeID, date).
		Order("room_id, start_time").
		Find(&bookings).Error
	return bookings, err
}

// FindRoomsForStore mengambil semua ruangan aktif dari satu store.
func FindRoomsForStore(storeID string) ([]models.StoreRoom, error) {
	var rooms []models.StoreRoom
	err := config.DB.Preload("RoomTemplate").
		Where("store_id = ? AND is_active = true AND deleted_at IS NULL", storeID).
		Order("room_template_id, unit_number").
		Find(&rooms).Error
	return rooms, err
}

// ── Overlap Check ─────────────────────────────────────────────────────────────

// CheckOverlap mengecek apakah slot di room sudah terpakai oleh booking aktif,
// hold customer yang belum expire, atau event booking yang memblokir store.
// Jam dinormalisasi relatif ke jam buka store (lihat NormalizeRange) supaya
// sesi lintas tengah malam terdeteksi.
func CheckOverlap(roomID, storeID, bookingDate, startTime, endTime, excludeID string) (bool, error) {
	return overlapIn(config.DB, roomID, storeID, bookingDate, startTime, endTime, excludeID)
}

func overlapIn(db *gorm.DB, roomID, storeID, bookingDate, startTime, endTime, excludeID string) (bool, error) {
	openMins, _ := customerBookingRepo.OperatingWindow(storeID, bookingDate)
	newStart, newEnd := utils.NormalizeRange(startTime, endTime, openMins)
	overlaps := func(s, e string) bool {
		es, ee := utils.NormalizeRange(s, e, openMins)
		return newStart < ee && newEnd > es
	}

	// 1. Booking aktif di room yang sama
	var existing []models.Booking
	q := db.Where("room_id = ? AND booking_date = ? AND status != 'cancelled'", roomID, bookingDate)
	if excludeID != "" {
		q = q.Where("id != ?", excludeID)
	}
	if err := q.Find(&existing).Error; err != nil {
		return false, err
	}
	for _, b := range existing {
		if overlaps(b.StartTime, b.EndTime) {
			return true, nil
		}
	}

	// 2. Hold customer yang sedang menunggu pembayaran
	var holds []models.BookingHold
	if err := db.Where("room_id = ? AND booking_date = ? AND expires_at > ?", roomID, bookingDate, time.Now()).
		Find(&holds).Error; err != nil {
		return false, err
	}
	for _, h := range holds {
		if overlaps(h.StartTime, h.EndTime) {
			return true, nil
		}
	}

	// 3. Event booking yang memblokir seluruh store
	if storeID != "" {
		hasEvent, err := eventRepo.CheckOverlapWithEvent(storeID, bookingDate, startTime, endTime, "")
		if err != nil {
			return false, err
		}
		if hasEvent {
			return true, nil
		}
	}
	return false, nil
}

var (
	ErrSlotTaken          = errors.New("slot waktu sudah terisi oleh booking atau event lain")
	ErrCreditInsufficient = errors.New("sisa jam play credits tidak cukup")
	ErrVoucherUsed        = errors.New("voucher sudah pernah digunakan customer ini")
)

// CreditDeduction = potongan jam play credits yang dilakukan bersama booking.
type CreditDeduction struct {
	CreditID string
	Hours    float64
}

// CreateBookingAtomic menyimpan booking + potongan credits + pemakaian voucher
// dalam 1 transaction. Baris room dikunci (FOR UPDATE) lalu overlap dicek ulang,
// sehingga 2 admin tidak bisa membooking slot yang sama bersamaan.
// Potongan credits memakai guard remaining_hours >= jam → saldo tidak bisa negatif.
func CreateBookingAtomic(b *models.Booking, deduct *CreditDeduction, usage *models.VoucherUsage) error {
	return config.DB.Transaction(func(tx *gorm.DB) error {
		var room models.StoreRoom
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", b.RoomID).First(&room).Error; err != nil {
			return err
		}
		taken, err := overlapIn(tx, b.RoomID, b.StoreID, b.BookingDate.Format("2006-01-02"), b.StartTime, b.EndTime, "")
		if err != nil {
			return err
		}
		if taken {
			return ErrSlotTaken
		}
		if err := tx.Create(b).Error; err != nil {
			return err
		}
		if deduct != nil {
			res := tx.Model(&models.CustomerPlayCredit{}).
				Where("id = ? AND remaining_hours >= ? AND is_active = true AND deleted_at IS NULL AND expires_at > ?",
					deduct.CreditID, deduct.Hours, time.Now()).
				Update("remaining_hours", gorm.Expr("remaining_hours - ?", deduct.Hours))
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected != 1 {
				return ErrCreditInsufficient
			}
		}
		if usage != nil {
			usage.BookingID = &b.ID
			if err := tx.Create(usage).Error; err != nil {
				if utils.IsDuplicateKey(err) {
					return ErrVoucherUsed
				}
				return err
			}
			if err := tx.Model(&models.Voucher{}).Where("id = ?", usage.VoucherID).
				Update("used_count", gorm.Expr("used_count + 1")).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// FindRoomTemplateInStore mengembalikan room_template_id jika room aktif dan milik store.
func FindRoomTemplateInStore(roomID, storeID string) (uint, error) {
	var room models.StoreRoom
	err := config.DB.Select("room_template_id").
		Where("id = ? AND store_id = ? AND is_active = true AND deleted_at IS NULL", roomID, storeID).
		First(&room).Error
	return room.RoomTemplateID, err
}

// ── Sessions Ending Soon ──────────────────────────────────────────────────────

// FindSessionsEndingSoon mengambil sesi yang berakhir dalam X menit ke depan (default 5).
// Booking kemarin ikut dicek: sesi 23:00–02:00 tercatat di tanggal kemarin.
func FindSessionsEndingSoon(storeID string, withinMinutes int) ([]models.Booking, error) {
	if withinMinutes <= 0 {
		withinMinutes = 5
	}
	now := time.Now()
	today := now.In(jakartaLoc())
	limit := now.Add(time.Duration(withinMinutes) * time.Minute)

	var candidates []models.Booking
	query := config.DB.
		Preload("Room.RoomTemplate").
		Where("booking_date IN ? AND status IN ('upcoming','ongoing')",
			[]string{today.AddDate(0, 0, -1).Format("2006-01-02"), today.Format("2006-01-02")})
	if storeID != "" {
		query = query.Where("store_id = ?", storeID)
	}
	if err := query.Find(&candidates).Error; err != nil {
		return nil, err
	}

	opens := map[string]int{}
	var result []models.Booking
	for _, b := range candidates {
		key := b.StoreID + "|" + b.BookingDate.Format("2006-01-02")
		if _, ok := opens[key]; !ok {
			opens[key], _ = customerBookingRepo.OperatingWindow(b.StoreID, b.BookingDate.Format("2006-01-02"))
		}
		_, end := utils.SessionWindow(b.BookingDate, b.StartTime, b.EndTime, opens[key])
		if end.After(now) && !end.After(limit) {
			result = append(result, b)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		_, ei := utils.SessionWindow(result[i].BookingDate, result[i].StartTime, result[i].EndTime, opens[result[i].StoreID+"|"+result[i].BookingDate.Format("2006-01-02")])
		_, ej := utils.SessionWindow(result[j].BookingDate, result[j].StartTime, result[j].EndTime, opens[result[j].StoreID+"|"+result[j].BookingDate.Format("2006-01-02")])
		return ei.Before(ej)
	})
	return result, nil
}

// ── Available Credits ─────────────────────────────────────────────────────────

// FindAvailableCreditsForBooking mengambil play credits customer yang aktif dan
// berlaku di store yang dipilih, dengan sisa jam >= durationHours.
func FindAvailableCreditsForBooking(customerID, storeID, bookingDate string, durationHours float64) ([]models.CustomerPlayCredit, error) {
	var credits []models.CustomerPlayCredit
	// now := time.Now()

	err := config.DB.Preload("Package.PackageStores.Store").
		Joins("JOIN play_credits_packages p ON p.id = customer_play_credits.package_id").
		Where(`customer_play_credits.customer_id = ?
			AND customer_play_credits.is_active = true
			AND customer_play_credits.deleted_at IS NULL
			AND customer_play_credits.expires_at > ?
			AND customer_play_credits.remaining_hours >= ?
			AND (
				p.apply_to_all_stores = true
				OR EXISTS (
					SELECT 1 FROM play_credits_package_stores pcps
					WHERE pcps.package_id = p.id AND pcps.store_id = ?
				)
			)`,
			customerID, bookingDate, durationHours, storeID).
		Find(&credits).Error
	return credits, err
}

// ── Status Auto-Update ────────────────────────────────────────────────────────

// BatchUpdateStatus memperbarui status booking berdasarkan waktu WIB saat ini.
// Dipanggil saat load dashboard untuk menjaga konsistensi DB. Status dihitung
// di Go dengan waktu absolut (utils.SessionStatus): perbandingan jam di SQL
// tidak bisa menangani sesi lintas tengah malam.
func BatchUpdateStatus(storeID string) error {
	now := time.Now()
	today := now.In(jakartaLoc()).Format("2006-01-02")

	var bookings []models.Booking
	if err := config.DB.Select("id", "store_id", "booking_date", "start_time", "end_time", "status").
		Where("store_id = ? AND status IN ('upcoming','ongoing') AND booking_date <= ?", storeID, today).
		Find(&bookings).Error; err != nil {
		return err
	}
	opens := map[string]int{}
	for _, b := range bookings {
		date := b.BookingDate.Format("2006-01-02")
		if _, ok := opens[date]; !ok {
			opens[date], _ = customerBookingRepo.OperatingWindow(storeID, date)
		}
		next := utils.SessionStatus(b.Status, b.BookingDate, b.StartTime, b.EndTime, opens[date], now)
		if next == b.Status {
			continue
		}
		// Bersyarat pada status lama: tidak menimpa pembatalan yang terjadi bersamaan.
		if err := config.DB.Model(&models.Booking{}).Where("id = ? AND status = ?", b.ID, b.Status).
			Update("status", next).Error; err != nil {
			return err
		}
	}
	return nil
}

// GetRoomNameByID mengambil nama room berdasarkan ID, dipakai oleh notification service.
func GetRoomNameByID(roomID string, name *string) {
	config.DB.Table("store_rooms").Select("name").Where("id = ?", roomID).Scan(name)
}

// ErrAlreadyFinal: booking sudah cancelled/completed saat akan dibatalkan.
var ErrAlreadyFinal = errors.New("booking sudah final")

// CancelBookingAtomic membatalkan booking, mengembalikan jam play credits, dan
// melepas voucher dalam 1 transaction. Booking "diklaim" dengan UPDATE bersyarat,
// jadi 2 pembatalan bersamaan tidak mengembalikan credits 2 kali.
func CancelBookingAtomic(b *models.Booking, reason, cancelledBy string) error {
	return config.DB.Transaction(func(tx *gorm.DB) error {
		claim := tx.Model(&models.Booking{}).
			Where("id = ? AND status NOT IN ('cancelled','completed')", b.ID).
			Updates(map[string]interface{}{
				"status":        "cancelled",
				"cancel_reason": reason,
				"cancelled_at":  time.Now(),
				"cancelled_by":  cancelledBy,
				"updated_by":    cancelledBy,
			})
		if claim.Error != nil {
			return claim.Error
		}
		if claim.RowsAffected != 1 {
			return ErrAlreadyFinal
		}

		if b.PaymentMethod == "play_credits" && b.PlayCreditID != nil {
			if err := tx.Model(&models.CustomerPlayCredit{}).Where("id = ?", *b.PlayCreditID).
				Update("remaining_hours", gorm.Expr("remaining_hours + ?", b.DurationHours)).Error; err != nil {
				return err
			}
		}

		if b.VoucherID != nil {
			res := tx.Where("voucher_id = ? AND booking_id = ?", *b.VoucherID, b.ID).Delete(&models.VoucherUsage{})
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected > 0 {
				if err := tx.Model(&models.Voucher{}).Where("id = ? AND used_count > 0", *b.VoucherID).
					Update("used_count", gorm.Expr("used_count - 1")).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}
