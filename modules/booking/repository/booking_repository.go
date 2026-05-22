package repository

import (
	"fmt"
	"time"

	"game_lounge_be/config"
	"game_lounge_be/models"
	eventRepo "game_lounge_be/modules/event_booking/repository"
)

// ── Sequence ──────────────────────────────────────────────────────────────────

// EnsureSequenceExists memastikan baris sequence ada di DB (id=1).
func EnsureSequenceExists() {
	config.DB.Exec("INSERT IGNORE INTO booking_sequences (id, last_sequence) VALUES (1, 0)")
}

// GetNextSequence mengambil dan increment global counter secara atomik.
func GetNextSequence() (uint, error) {
	result := config.DB.Exec("UPDATE booking_sequences SET last_sequence = last_sequence + 1 WHERE id = 1")
	if result.Error != nil {
		return 0, result.Error
	}
	if result.RowsAffected == 0 {
		// Baris belum ada, init dulu
		EnsureSequenceExists()
		config.DB.Exec("UPDATE booking_sequences SET last_sequence = last_sequence + 1 WHERE id = 1")
	}
	var seq models.BookingSequence
	if err := config.DB.First(&seq, 1).Error; err != nil {
		return 0, err
	}
	return seq.LastSequence, nil
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

// GetRoomTemplateID mengambil room_template_id dari store_room berdasarkan room ID.
func GetRoomTemplateID(roomID string) (uint, error) {
	var room models.StoreRoom
	err := config.DB.Select("room_template_id").Where("id = ?", roomID).First(&room).Error
	return room.RoomTemplateID, err
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

// CheckOverlap memvalidasi apakah slot waktu sudah terisi booking lain.
// Overlap: newStart < existEnd AND newEnd > existStart.
// Selain mengecek regular booking per room, juga mengecek event booking
// yang memblokir seluruh store pada slot yang sama.
func CheckOverlap(roomID, storeID, bookingDate, startTime, endTime, excludeID string) (bool, error) {
	startMins := parseTimeToMinutes(startTime)
	endMins := parseTimeToMinutes(endTime)
	if endMins <= startMins {
		endMins += 24 * 60 // lintas tengah malam
	}

	// 1. Cek regular booking di room yang sama
	var existingBookings []models.Booking
	query := config.DB.Where("room_id = ? AND booking_date = ? AND status != 'cancelled'", roomID, bookingDate)
	if excludeID != "" {
		query = query.Where("id != ?", excludeID)
	}
	if err := query.Find(&existingBookings).Error; err != nil {
		return false, err
	}

	for _, b := range existingBookings {
		eStart := parseTimeToMinutes(b.StartTime)
		eEnd := parseTimeToMinutes(b.EndTime)
		if eEnd <= eStart {
			eEnd += 24 * 60
		}
		if startMins < eEnd && endMins > eStart {
			return true, nil
		}
	}

	// 2. Cek event booking yang memblokir seluruh store
	if storeID != "" {
		hasEvent, _ := eventRepo.CheckOverlapWithEvent(storeID, bookingDate, startTime, endTime, "")
		if hasEvent {
			return true, nil
		}
	}

	return false, nil
}

// parseTimeToMinutes mengubah "HH:MM" atau "HH:MM:SS" menjadi total menit dari 00:00.
func parseTimeToMinutes(t string) int {
	if len(t) < 5 {
		return 0
	}
	h := int(t[0]-'0')*10 + int(t[1]-'0')
	m := int(t[3]-'0')*10 + int(t[4]-'0')
	return h*60 + m
}

// ── Sessions Ending Soon ──────────────────────────────────────────────────────

// FindSessionsEndingSoon mengambil sesi yang akan berakhir dalam X menit ke depan (default 5).
func FindSessionsEndingSoon(storeID string, withinMinutes int) ([]models.Booking, error) {
	if withinMinutes <= 0 {
		withinMinutes = 5
	}
	now := time.Now().In(jakartaLoc())
	today := now.Format("2006-01-02")
	nowTime := now.Format("15:04:05")
	endLimit := now.Add(time.Duration(withinMinutes) * time.Minute).Format("15:04:05")

	var bookings []models.Booking
	query := config.DB.
		Preload("Room.RoomTemplate").
		Where("booking_date = ? AND status IN ('upcoming','ongoing')", today).
		Where("end_time > ? AND end_time <= ?", nowTime, endLimit)

	if storeID != "" {
		query = query.Where("store_id = ?", storeID)
	}

	err := query.Order("end_time ASC").Find(&bookings).Error
	return bookings, err
}

// ── Available Credits ─────────────────────────────────────────────────────────

// FindAvailableCreditsForBooking mengambil play credits customer yang aktif dan
// berlaku di store yang dipilih, dengan sisa jam >= durationHours.
func FindAvailableCreditsForBooking(customerID, storeID string, durationHours float64) ([]models.CustomerPlayCredit, error) {
	var credits []models.CustomerPlayCredit
	now := time.Now()

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
			customerID, now, durationHours, storeID).
		Find(&credits).Error
	return credits, err
}

// ── Status Auto-Update ────────────────────────────────────────────────────────

// BatchUpdateStatus memperbarui status booking berdasarkan waktu Jakarta saat ini.
// Dipanggil saat load dashboard untuk menjaga konsistensi DB.
func BatchUpdateStatus(storeID, date string) error {
	now := time.Now().In(jakartaLoc())
	nowTime := now.Format("15:04:05")

	// upcoming → ongoing: booking_date = today AND start <= now AND end > now
	config.DB.Model(&models.Booking{}).
		Where("store_id = ? AND booking_date = ? AND status = 'upcoming' AND start_time <= ? AND end_time > ?",
			storeID, date, nowTime, nowTime).
		Update("status", "ongoing")

	// ongoing → completed: booking_date = today AND end <= now
	config.DB.Model(&models.Booking{}).
		Where("store_id = ? AND booking_date = ? AND status = 'ongoing' AND end_time <= ?",
			storeID, date, nowTime).
		Update("status", "completed")

	// upcoming → completed: booking_date < today (terlewat)
	config.DB.Model(&models.Booking{}).
		Where("store_id = ? AND booking_date < ? AND status = 'upcoming'", storeID, date).
		Update("status", "completed")

	return nil
}


// GetRoomNameByID mengambil nama room berdasarkan ID, dipakai oleh notification service.
func GetRoomNameByID(roomID string, name *string) {
	config.DB.Table("store_rooms").Select("name").Where("id = ?", roomID).Scan(name)
}
