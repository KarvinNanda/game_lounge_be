package repository

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"game_lounge_be/config"
	"game_lounge_be/models"
	eventRepo "game_lounge_be/modules/event_booking/repository"
	"game_lounge_be/utils"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

	openMins, closeMins := OperatingWindow(storeID, date)
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

	// Event (full venue) memblokir semua room di store.
	events, _ := eventRepo.FindForDashboard(storeID, date)

	// Generate slots dari openTime sampai closeTime - durasiSlot
	var slots []SlotInfo
	for startMins := openMins; startMins+durMins <= closeMins; startMins += 60 {
		endMins   := startMins + durMins
		slotStart := utils.MinsToClock(startMins)
		slotEnd   := utils.MinsToClock(endMins)
		available := false

		for _, room := range rooms {
			conflict := false

			// Cek terhadap bookings
			for _, b := range bookings {
				if b.RoomID != room.ID {
					continue
				}
				bS, bE := utils.NormalizeRange(b.StartTime, b.EndTime, openMins)
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
					hS, hE := utils.NormalizeRange(h.StartTime, h.EndTime, openMins)
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

		for _, e := range events {
			if eS, eE := utils.NormalizeRange(e.StartTime, e.EndTime, openMins); startMins < eE && endMins > eS {
				available = false
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

// OperatingWindow mengembalikan jam buka & tutup (menit dari 00:00) untuk store
// pada tanggal tersebut. closeMins > 24*60 jika store tutup setelah tengah malam.
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

// ── Hold ──────────────────────────────────────────────────────────────────────

// CreateHold mencari unit room tersedia lalu buat hold dalam satu DB transaction.
// Baris store_rooms dikunci (SELECT ... FOR UPDATE), sehingga 2 request untuk
// room type yang sama diproses bergantian dan tidak bisa hold unit yang sama.
//
// reserve (opsional) = reservasi voucher. Dibuat di transaction yang sama:
// UNIQUE(voucher_id, customer_id) menolak hold paralel dengan voucher yang sama.
func CreateHold(h *models.BookingHold, reserve *models.VoucherUsage) (*models.BookingHold, error) {
	bookingDateStr := h.BookingDate.Format("2006-01-02")
	openMins, _ := OperatingWindow(h.StoreID, bookingDateStr)
	newStart, newEnd := utils.NormalizeRange(h.StartTime, h.EndTime, openMins)

	// Event booking memblokir seluruh store.
	if hasEvent, err := eventRepo.CheckOverlapWithEvent(h.StoreID, bookingDateStr, h.StartTime, h.EndTime, ""); err != nil {
		return nil, err
	} else if hasEvent {
		return nil, fmt.Errorf("store sedang dipakai untuk event pada jam tersebut")
	}

	err := config.DB.Transaction(func(tx *gorm.DB) error {
		var rooms []models.StoreRoom
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("store_id = ? AND room_template_id = ? AND is_active = true AND deleted_at IS NULL",
				h.StoreID, h.RoomTemplateID).
			Order("unit_number").Find(&rooms).Error; err != nil {
			return err
		}

		roomIDs := make([]string, 0, len(rooms))
		for _, r := range rooms {
			roomIDs = append(roomIDs, r.ID)
		}
		var bookings []models.Booking
		var holds []models.BookingHold
		if len(roomIDs) > 0 {
			if err := tx.Where("room_id IN ? AND booking_date = ? AND status != 'cancelled'", roomIDs, bookingDateStr).
				Find(&bookings).Error; err != nil {
				return err
			}
			if err := tx.Where("room_id IN ? AND booking_date = ? AND expires_at > ?", roomIDs, bookingDateStr, time.Now()).
				Find(&holds).Error; err != nil {
				return err
			}
		}

		busy := map[string]bool{}
		for _, b := range bookings {
			if s, e := utils.NormalizeRange(b.StartTime, b.EndTime, openMins); newStart < e && newEnd > s {
				busy[b.RoomID] = true
			}
		}
		for _, x := range holds {
			if s, e := utils.NormalizeRange(x.StartTime, x.EndTime, openMins); newStart < e && newEnd > s {
				busy[x.RoomID] = true
			}
		}

		for _, r := range rooms {
			if !busy[r.ID] {
				h.RoomID = r.ID
				h.ExpiresAt = time.Now().Add(HoldDuration)
				if err := tx.Create(h).Error; err != nil {
					return err
				}
				if reserve != nil {
					if err := tx.Create(reserve).Error; err != nil {
						if utils.IsDuplicateKey(err) {
							return ErrVoucherAlreadyUsed
						}
						return err
					}
				}
				return nil
			}
		}
		return errNoRoomAvailable
	})
	if err != nil {
		return nil, err
	}
	return h, nil
}

// HoldDuration = waktu customer untuk membayar. Invoice Xendit diberi durasi
// yang sama supaya customer tidak bisa membayar setelah hold habis.
const HoldDuration = 15 * time.Minute

var errNoRoomAvailable = fmt.Errorf("tidak ada unit ruangan yang tersedia pada jam tersebut")

// ErrVoucherAlreadyUsed: voucher sudah dipakai / sedang direservasi customer ini.
var ErrVoucherAlreadyUsed = errors.New("voucher sudah digunakan")

// HoldVoucherID membaca voucher_id dari price_breakdown hold ("" jika tidak ada).
func HoldVoucherID(priceBreakdown string) string {
	var b struct {
		VoucherID string `json:"voucher_id"`
	}
	_ = json.Unmarshal([]byte(priceBreakdown), &b)
	return b.VoucherID
}

// releaseVoucherReservation menghapus reservasi voucher (usage tanpa booking) milik hold.
func releaseVoucherReservation(tx *gorm.DB, h *models.BookingHold) error {
	voucherID := HoldVoucherID(h.PriceBreakdown)
	if voucherID == "" {
		return nil
	}
	return tx.Where("voucher_id = ? AND customer_id = ? AND booking_id IS NULL", voucherID, h.CustomerID).
		Delete(&models.VoucherUsage{}).Error
}

// ReleaseHold menghapus hold beserta reservasi vouchernya (mis. invoice gagal dibuat).
func ReleaseHold(h *models.BookingHold) error {
	return config.DB.Transaction(func(tx *gorm.DB) error {
		if err := releaseVoucherReservation(tx, h); err != nil {
			return err
		}
		return tx.Delete(&models.BookingHold{}, "id = ?", h.ID).Error
	})
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

// holdGracePeriod: hold expired tetap disimpan sebentar supaya webhook PAID yang
// datang terlambat masih bisa menemukan hold-nya (lihat ConfirmBookingFromWebhook).
const holdGracePeriod = time.Hour

// CleanExpiredHolds menghapus hold yang sudah expire lebih dari holdGracePeriod.
// Dipanggil periodik oleh jobs.StartCleanup.
// Reservasi voucher milik hold tersebut ikut dilepas supaya voucher bisa dipakai lagi.
func CleanExpiredHolds() error {
	var holds []models.BookingHold
	if err := config.DB.Where("expires_at < ?", time.Now().Add(-holdGracePeriod)).Find(&holds).Error; err != nil {
		return err
	}
	for i := range holds {
		if err := ReleaseHold(&holds[i]); err != nil {
			return err
		}
	}
	return nil
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
