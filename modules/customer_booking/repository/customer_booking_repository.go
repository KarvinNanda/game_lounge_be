package repository

import (
	"fmt"
	"time"

	"game_lounge_be/config"
	"game_lounge_be/models"
)

// ── Availability ──────────────────────────────────────────────────────────────

type SlotInfo struct {
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	Available bool   `json:"available"`
}

// GetAvailableSlots mengembalikan semua slot per jam untuk room type pada tanggal tertentu.
// Slot available = minimal 1 unit room tidak konflik dengan booking atau hold aktif.
func GetAvailableSlots(storeID string, roomTemplateID uint, date string, durationHours float64) ([]SlotInfo, error) {
	// Ambil semua unit room aktif untuk store + template ini
	var rooms []models.StoreRoom
	config.DB.Where(
		"store_id = ? AND room_template_id = ? AND is_active = true AND deleted_at IS NULL",
		storeID, roomTemplateID,
	).Find(&rooms)

	if len(rooms) == 0 {
		return []SlotInfo{}, nil
	}

	// Tentukan day_type berdasarkan hari dalam seminggu
	parsedDate, err := time.Parse("2006-01-02", date)
	dayType := "weekday"
	if err == nil {
		if parsedDate.Weekday() == time.Saturday || parsedDate.Weekday() == time.Sunday {
			dayType = "weekend"
		}
	}

	// Ambil jam operasional efektif
	var openTime, closeTime string
	var opHours models.StoreOperatingHour
	config.DB.Where(
		"store_id = ? AND day_type = ? AND is_active = true AND deleted_at IS NULL",
		storeID, dayType,
	).First(&opHours)

	if opHours.ID == 0 {
		openTime, closeTime = "10:00", "02:00"
	} else {
		openTime  = opHours.OpenTime
		closeTime = opHours.CloseTime
		// Potong ke HH:MM jika format HH:MM:SS
		if len(openTime) > 5 {
			openTime = openTime[:5]
		}
		if len(closeTime) > 5 {
			closeTime = closeTime[:5]
		}
	}

	openMins  := parseMins(openTime)
	closeMins := parseMins(closeTime)
	if closeMins <= openMins {
		closeMins += 24 * 60 // operasional melewati tengah malam
	}
	durMins := int(durationHours * 60)

	// Ambil ID semua unit room
	var roomIDs []string
	for _, r := range rooms {
		roomIDs = append(roomIDs, r.ID)
	}

	// Ambil semua booking aktif pada date + rooms ini
	var bookings []models.Booking
	config.DB.Where(
		"room_id IN ? AND booking_date = ? AND status NOT IN ('cancelled')",
		roomIDs, date,
	).Find(&bookings)

	// Ambil hold aktif (belum expire)
	var holds []models.BookingHold
	config.DB.Where(
		"room_id IN ? AND booking_date = ? AND expires_at > ?",
		roomIDs, date, time.Now(),
	).Find(&holds)

	// Generate slots dari openTime sampai closeTime - durasiSlot
	var slots []SlotInfo
	for startMins := openMins; startMins+durMins <= closeMins; startMins += 60 {
		endMins   := startMins + durMins
		slotStart := minsToTime(startMins)
		slotEnd   := minsToTime(endMins)
		available := false

		for _, room := range rooms {
			conflict := false

			// Cek terhadap bookings
			for _, b := range bookings {
				if b.RoomID != room.ID {
					continue
				}
				bS := parseMins(b.StartTime[:5])
				bE := parseMins(b.EndTime[:5])
				if bE <= bS {
					bE += 24 * 60
				}
				if startMins < bE && endMins > bS {
					conflict = true
					break
				}
			}

			// Cek terhadap holds
			if !conflict {
				for _, h := range holds {
					if h.RoomID != room.ID {
						continue
					}
					hS := parseMins(h.StartTime[:5])
					hE := parseMins(h.EndTime[:5])
					if hE <= hS {
						hE += 24 * 60
					}
					if startMins < hE && endMins > hS {
						conflict = true
						break
					}
				}
			}

			if !conflict {
				available = true
				break
			}
		}

		slots = append(slots, SlotInfo{
			StartTime: slotStart,
			EndTime:   slotEnd,
			Available: available,
		})
	}

	return slots, nil
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

// ── Hold ──────────────────────────────────────────────────────────────────────

// CreateHold mencari unit room tersedia lalu buat hold dalam satu DB transaction.
// Mencegah race condition: dua customer tidak bisa hold unit yang sama secara bersamaan.
func CreateHold(h *models.BookingHold) (*models.BookingHold, error) {
	tx := config.DB.Begin()

	// Lock & cari room tersedia untuk store + template yang diminta
	var rooms []models.StoreRoom
	tx.Set("gorm:query_option", "FOR UPDATE").
		Where(
			"store_id = ? AND room_template_id = ? AND is_active = true AND deleted_at IS NULL",
			h.StoreID, h.RoomTemplateID,
		).Find(&rooms)

	availableRoomID := ""
	bookingDateStr  := h.BookingDate.Format("2006-01-02")

	for _, room := range rooms {
		// Cek booking conflict
		var bCount int64
		tx.Model(&models.Booking{}).
			Where(
				"room_id = ? AND booking_date = ? AND status NOT IN ('cancelled') AND start_time < ? AND end_time > ?",
				room.ID, bookingDateStr, h.EndTime, h.StartTime,
			).Count(&bCount)
		if bCount > 0 {
			continue
		}

		// Cek hold conflict
		var hCount int64
		tx.Model(&models.BookingHold{}).
			Where(
				"room_id = ? AND booking_date = ? AND expires_at > NOW() AND start_time < ? AND end_time > ?",
				room.ID, bookingDateStr, h.EndTime, h.StartTime,
			).Count(&hCount)
		if hCount > 0 {
			continue
		}

		availableRoomID = room.ID
		break
	}

	if availableRoomID == "" {
		tx.Rollback()
		return nil, fmt.Errorf("tidak ada unit ruangan yang tersedia pada jam tersebut")
	}

	h.RoomID    = availableRoomID
	h.ExpiresAt = time.Now().Add(15 * time.Minute)

	if err := tx.Create(h).Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	tx.Commit()
	return h, nil
}

func FindHoldByID(id string) (*models.BookingHold, error) {
	var h models.BookingHold
	err := config.DB.
		Preload("Customer").
		Preload("Store").
		Preload("Room.RoomTemplate").
		Preload("RoomTemplate").
		Where("id = ?", id).First(&h).Error
	return &h, err
}

func FindHoldByXenditInvoice(invoiceID string) (*models.BookingHold, error) {
	var h models.BookingHold
	err := config.DB.
		Preload("Customer").
		Preload("Store").
		Preload("Room.RoomTemplate").
		Preload("RoomTemplate").
		Where("xendit_invoice_id = ?", invoiceID).First(&h).Error
	return &h, err
}

func UpdateHold(h *models.BookingHold) error {
	return config.DB.Save(h).Error
}

func DeleteHold(id string) error {
	return config.DB.Delete(&models.BookingHold{}, "id = ?", id).Error
}

// CleanExpiredHolds menghapus hold yang sudah expire.
// Dapat dipanggil secara periodik (cron/goroutine).
func CleanExpiredHolds() {
	config.DB.Delete(&models.BookingHold{}, "expires_at < NOW()")
}

// ── Customer Bookings ─────────────────────────────────────────────────────────

// FindCustomerBookings mengambil list booking milik customer dengan pagination.
func FindCustomerBookings(customerID string, page, perPage int) ([]models.Booking, int64, error) {
	var bookings []models.Booking
	var total int64
	q := config.DB.
		Preload("Room.RoomTemplate").
		Preload("Store").
		Where("customer_id = ?", customerID)

	q.Model(&models.Booking{}).Count(&total)

	err := q.Order("booking_date DESC, start_time DESC").
		Offset((page-1)*perPage).Limit(perPage).Find(&bookings).Error
	return bookings, total, err
}

// FindCustomerBookingByID mengambil satu booking milik customer tertentu.
func FindCustomerBookingByID(id, customerID string) (*models.Booking, error) {
	var b models.Booking
	err := config.DB.
		Preload("Room.RoomTemplate").
		Preload("Store").
		Where("id = ? AND customer_id = ?", id, customerID).First(&b).Error
	return &b, err
}
