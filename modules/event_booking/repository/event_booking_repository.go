package repository

import (
	"time"

	"game_lounge_be/config"
	"game_lounge_be/models"
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
	err := config.DB.
		Where("store_id = ? AND booking_date = ? AND status != 'cancelled'", storeID, date).
		Find(&bookings).Error
	return bookings, err
}

// ── Overlap Check ─────────────────────────────────────────────────────────────

// parseToMins mengubah "HH:MM" atau "HH:MM:SS" menjadi total menit dari 00:00.
func parseToMins(t string) int {
	if len(t) < 5 {
		return 0
	}
	h := int(t[0]-'0')*10 + int(t[1]-'0')
	m := int(t[3]-'0')*10 + int(t[4]-'0')
	return h*60 + m
}

// CheckOverlapWithRegular mengecek apakah ada regular booking yang conflict dengan event booking.
// Conflict jika ada regular booking di store yang sama, tanggal yang sama,
// dan jam-nya overlap dengan event.
func CheckOverlapWithRegular(storeID, date, startTime, endTime string) (bool, error) {
	startMins := parseToMins(startTime)
	endMins := parseToMins(endTime)
	if endMins <= startMins {
		endMins += 24 * 60
	}

	var bookings []models.Booking
	if err := config.DB.Where("store_id = ? AND booking_date = ? AND status != 'cancelled'",
		storeID, date).Find(&bookings).Error; err != nil {
		return false, err
	}

	for _, b := range bookings {
		bStart := parseToMins(b.StartTime)
		bEnd := parseToMins(b.EndTime)
		if bEnd <= bStart {
			bEnd += 24 * 60
		}
		if startMins < bEnd && endMins > bStart {
			return true, nil
		}
	}
	return false, nil
}

// CheckOverlapWithEvent mengecek apakah ada event booking lain yang conflict.
func CheckOverlapWithEvent(storeID, date, startTime, endTime, excludeID string) (bool, error) {
	startMins := parseToMins(startTime)
	endMins := parseToMins(endTime)
	if endMins <= startMins {
		endMins += 24 * 60
	}

	var events []models.EventBooking
	q := config.DB.Where("store_id = ? AND booking_date = ? AND status != 'cancelled'", storeID, date)
	if excludeID != "" {
		q = q.Where("id != ?", excludeID)
	}
	if err := q.Find(&events).Error; err != nil {
		return false, err
	}

	for _, e := range events {
		eStart := parseToMins(e.StartTime)
		eEnd := parseToMins(e.EndTime)
		if eEnd <= eStart {
			eEnd += 24 * 60
		}
		if startMins < eEnd && endMins > eStart {
			return true, nil
		}
	}
	return false, nil
}

// ── Status Auto-Update ────────────────────────────────────────────────────────

// BatchUpdateStatus memperbarui status event booking berdasarkan waktu Jakarta.
func BatchUpdateStatus(storeID, date string) {
	jakartaLoc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		jakartaLoc = time.UTC
	}
	now := time.Now().In(jakartaLoc)
	nowTime := now.Format("15:04:05")

	config.DB.Model(&models.EventBooking{}).
		Where("store_id = ? AND booking_date = ? AND status = 'upcoming' AND start_time <= ? AND end_time > ?",
			storeID, date, nowTime, nowTime).
		Update("status", "ongoing")

	config.DB.Model(&models.EventBooking{}).
		Where("store_id = ? AND booking_date = ? AND status = 'ongoing' AND end_time <= ?",
			storeID, date, nowTime).
		Update("status", "completed")

	config.DB.Model(&models.EventBooking{}).
		Where("store_id = ? AND booking_date < ? AND status IN ('upcoming','ongoing')", storeID, date).
		Update("status", "completed")
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
		price.UpdatedBy = &updatedBy
		err = config.DB.Save(&price).Error
	}
	return &price, err
}
