package service

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"errors"
	"fmt"
	"game_lounge_be/config"
	"game_lounge_be/models"
	"game_lounge_be/modules/store/dto"
	"game_lounge_be/modules/store/repository"
	"time"

	"github.com/google/uuid"
)

// ── Operating Hours ───────────────────────────────────────────────────────────

// OperatingHoursResult adalah hasil cek jam operasional efektif untuk suatu tanggal.
type OperatingHoursResult struct {
	OpenTime    string `json:"open_time"`
	CloseTime   string `json:"close_time"`
	IsHoliday   bool   `json:"is_holiday"`
	HolidayName string `json:"holiday_name"` // nama hari libur jika holiday
	HolidayType string `json:"holiday_type"` // "global" | "store" | ""
}

// GetEffectiveOperatingHours mengembalikan jam operasional efektif store pada tanggal tertentu.
// Priority: Global Holiday → Store Holiday → Regular Operating Hours (weekday/weekend)
// Jika global/store holiday → is_holiday=true, happy hour tidak berlaku.
func GetEffectiveOperatingHours(storeID string, date time.Time) (*OperatingHoursResult, error) {
	dateStr := date.Format("2006-01-02")

	// 1. Cek global holiday (berlaku untuk semua store)
	var globalHoliday models.GlobalHolidaySchedule
	if err := config.DB.Where("DATE(date) = ? AND deleted_at IS NULL", dateStr).
		First(&globalHoliday).Error; err == nil {
		return &OperatingHoursResult{
			OpenTime:    globalHoliday.OpenTime,
			CloseTime:   globalHoliday.CloseTime,
			IsHoliday:   true,
			HolidayName: globalHoliday.Name,
			HolidayType: "global",
		}, nil
	}

	// 2. Cek store-specific holiday
	var storeHoliday models.StoreHolidaySchedule
	if err := config.DB.Where("store_id = ? AND DATE(date) = ? AND deleted_at IS NULL",
		storeID, dateStr).First(&storeHoliday).Error; err == nil {
		return &OperatingHoursResult{
			OpenTime:    storeHoliday.OpenTime,
			CloseTime:   storeHoliday.CloseTime,
			IsHoliday:   true,
			HolidayName: "Tanggal Merah Cabang",
			HolidayType: "store",
		}, nil
	}

	// 3. Jam operasional reguler (weekday/weekend)
	dayType := "weekend"
	wd := date.Weekday()
	if wd >= time.Monday && wd <= time.Friday {
		dayType = "weekday"
	}

	var opHour models.StoreOperatingHour
	if err := config.DB.Where(
		"store_id = ? AND day_type = ? AND is_active = true AND deleted_at IS NULL",
		storeID, dayType).First(&opHour).Error; err != nil {
		return nil, errors.New("jam operasional tidak ditemukan untuk store ini")
	}

	return &OperatingHoursResult{
		OpenTime:    opHour.OpenTime,
		CloseTime:   opHour.CloseTime,
		IsHoliday:   false,
		HolidayName: "",
		HolidayType: "",
	}, nil
}

// StoreListItem wraps a Store with a precomputed room count for list responses.
type StoreListItem struct {
	models.Store
	RoomCount int64 `json:"room_count"`
}

func GetAllStores(filter dto.StoreFilter) (interface{}, int64, map[string]int64, error) {
	stores, total, err := repository.FindAllStores(filter.Search, filter.Status, filter.Page, filter.PerPage)
	if err != nil {
		return nil, 0, nil, err
	}

	var result []StoreListItem
	for _, s := range stores {
		count := repository.CountRoomsByStore(s.ID)
		result = append(result, StoreListItem{Store: s, RoomCount: count})
	}

	stats := repository.GetStoreStats()
	return result, total, stats, nil
}

func GetStoreByID(id string) (*models.Store, error) {
	store, err := repository.FindStoreByID(id)
	if err != nil {
		return nil, errors.New("store tidak ditemukan")
	}
	return store, nil
}

func CreateStore(req dto.CreateStoreRequest, createdBy string) (*models.Store, error) {
	status := req.Status
	if status == "" {
		status = "draft"
	}

	whatsapp := req.Whatsapp
	postal := req.PostalCode
	desc := req.Description
	link := req.LinkGmaps
	photo := req.PhotoURL

	store := &models.Store{
		ID:          uuid.NewString(),
		Name:        req.Name,
		Address:     req.Address,
		Whatsapp:    &whatsapp,
		PostalCode:  &postal,
		Description: &desc,
		PhotoURL:    &photo,
		LinkGmaps:   &link,
		Status:      status,
		CreatedBy:   &createdBy,
	}

	// Validate rooms before inserting anything, so a rejected request creates no store.
	newRooms, err := planRoomAdditions(config.DB, store.ID, req.Rooms, createdBy)
	if err != nil {
		return nil, err
	}

	// Every write below commits together or not at all.
	err = config.DB.Transaction(func(tx *gorm.DB) error {
		if err := repository.CreateStore(tx, store); err != nil {
			return errors.New("gagal membuat store")
		}

		// Save operating hours
		if len(req.OperatingHours) > 0 {
			var hours []models.StoreOperatingHour
			for _, h := range req.OperatingHours {
				hours = append(hours, models.StoreOperatingHour{
					StoreID:   store.ID,
					DayType:   h.DayType,
					OpenTime:  h.OpenTime,
					CloseTime: h.CloseTime,
					IsActive:  h.IsActive,
					CreatedBy: &createdBy,
				})
			}
			if err := repository.UpsertOperatingHours(tx, store.ID, hours); err != nil {
				return errors.New("gagal menyimpan jam operasional")
			}
		}

		// Save holidays
		if len(req.Holidays) > 0 {
			var holidays []models.StoreHolidaySchedule
			for _, h := range req.Holidays {
				date, _ := time.Parse("2006-01-02", h.Date)
				holidays = append(holidays, models.StoreHolidaySchedule{
					StoreID:   store.ID,
					Date:      date,
					OpenTime:  h.OpenTime,
					CloseTime: h.CloseTime,
					CreatedBy: &createdBy,
				})
			}
			if err := repository.UpsertHolidaySchedules(tx, store.ID, holidays); err != nil {
				return errors.New("gagal menyimpan jadwal libur")
			}
		}

		if len(newRooms) > 0 {
			if err := repository.CreateStoreRooms(tx, newRooms); err != nil {
				return errors.New("gagal membuat unit ruangan")
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return GetStoreByID(store.ID)
}

func UpdateStore(id string, req dto.UpdateStoreRequest, updatedBy string) (*models.Store, error) {
	store, err := repository.FindStoreByID(id)
	if err != nil {
		return nil, errors.New("store tidak ditemukan")
	}

	whatsapp := req.Whatsapp
	postal := req.PostalCode
	desc := req.Description
	link := req.LinkGmaps

	store.Name = req.Name
	store.Address = req.Address
	store.Whatsapp = &whatsapp
	store.PostalCode = &postal
	store.Description = &desc
	store.LinkGmaps = &link
	store.Status = req.Status
	store.UpdatedBy = &updatedBy

	// Only update photo if new value provided
	if req.PhotoURL != "" {
		store.PhotoURL = &req.PhotoURL
	}

	// Every write below commits together or not at all.
	err = config.DB.Transaction(func(tx *gorm.DB) error {
		// Lock the store row, then count units: a second save of the same form
		// waits here and sees the units the first one added (no duplicates).
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Select("id").First(&models.Store{}, "id = ?", id).Error; err != nil {
			return errors.New("store tidak ditemukan")
		}
		// Validated before any write, so a rejected request has no side effects.
		newRooms, err := planRoomAdditions(tx, id, req.Rooms, updatedBy)
		if err != nil {
			return err
		}

		if err := repository.UpdateStore(tx, store); err != nil {
			return errors.New("gagal update store")
		}

		// Update operating hours
		if len(req.OperatingHours) > 0 {
			var hours []models.StoreOperatingHour
			for _, h := range req.OperatingHours {
				hours = append(hours, models.StoreOperatingHour{
					StoreID:   id,
					DayType:   h.DayType,
					OpenTime:  h.OpenTime,
					CloseTime: h.CloseTime,
					IsActive:  h.IsActive,
					UpdatedBy: &updatedBy,
				})
			}
			if err := repository.UpsertOperatingHours(tx, id, hours); err != nil {
				return errors.New("gagal update jam operasional")
			}
		}

		// Update holidays
		if len(req.Holidays) > 0 {
			var holidays []models.StoreHolidaySchedule
			for _, h := range req.Holidays {
				date, _ := time.Parse("2006-01-02", h.Date)
				holidays = append(holidays, models.StoreHolidaySchedule{
					StoreID:   id,
					Date:      date,
					OpenTime:  h.OpenTime,
					CloseTime: h.CloseTime,
					UpdatedBy: &updatedBy,
				})
			}
			if err := repository.UpsertHolidaySchedules(tx, id, holidays); err != nil {
				return errors.New("gagal update jadwal libur")
			}
		}

		if len(newRooms) > 0 {
			if err := repository.CreateStoreRooms(tx, newRooms); err != nil {
				return errors.New("gagal menambah unit ruangan")
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return GetStoreByID(id)
}

// ErrRoomSetup marks a rejected rooms payload (client error, HTTP 400).
// Test with errors.Is(err, ErrRoomSetup).
var ErrRoomSetup = errors.New("room setup tidak valid")

type roomSetupError struct{ msg string }

func (e roomSetupError) Error() string        { return e.msg }
func (e roomSetupError) Is(target error) bool { return target == ErrRoomSetup }

func roomErr(format string, args ...any) error {
	return roomSetupError{msg: fmt.Sprintf(format, args...)}
}

// planRoomAdditions turns the requested unit count per room template into the new
// units to create. It only adds: units are never removed here, because bookings,
// holds and FnB orders point at store_rooms.id. To take a unit out of service,
// deactivate it with PATCH /admin/store-rooms/:id/toggle.
// The target is a count, so sending the same request twice creates nothing new.
func planRoomAdditions(db *gorm.DB, storeID string, setups []dto.RoomSetupInput, actor string) ([]models.StoreRoom, error) {
	var out []models.StoreRoom
	seen := map[uint]bool{}
	for _, s := range setups {
		if s.UnitCount < 0 {
			return nil, roomErr("unit_count untuk room template %d tidak boleh negatif", s.RoomTemplateID)
		}
		if seen[s.RoomTemplateID] {
			return nil, roomErr("room template %d dikirim lebih dari 1 kali", s.RoomTemplateID)
		}
		seen[s.RoomTemplateID] = true

		var tmpl models.RoomTemplate
		if err := db.First(&tmpl, s.RoomTemplateID).Error; err != nil {
			return nil, roomErr("room template %d tidak ditemukan", s.RoomTemplateID)
		}

		var current int64
		if err := db.Model(&models.StoreRoom{}).
			Where("store_id = ? AND room_template_id = ? AND deleted_at IS NULL", storeID, s.RoomTemplateID).
			Count(&current).Error; err != nil {
			return nil, err
		}
		if int64(s.UnitCount) < current {
			return nil, roomErr("jumlah unit %s tidak bisa dikurangi dari %d ke %d; nonaktifkan unit lewat toggle", tmpl.Name, current, s.UnitCount)
		}
		// New units of an inactive or deleted type would never show in the customer
		// app. The same count is still accepted, because the admin form sends every
		// type the store has, including types that were deactivated later.
		if int64(s.UnitCount) > current && (!tmpl.IsActive || tmpl.DeletedAt != nil) {
			return nil, roomErr("tipe ruangan %s tidak aktif, unit baru tidak bisa ditambahkan", tmpl.Name)
		}

		// Number after the highest unit ever used for this template (including
		// soft-deleted rows), so a unit number is never reused.
		var maxUnit uint
		if err := db.Model(&models.StoreRoom{}).
			Where("store_id = ? AND room_template_id = ?", storeID, s.RoomTemplateID).
			Select("COALESCE(MAX(unit_number), 0)").Scan(&maxUnit).Error; err != nil {
			return nil, err
		}
		for i := int64(0); i < int64(s.UnitCount)-current; i++ {
			n := maxUnit + uint(i) + 1
			out = append(out, models.StoreRoom{
				ID:             uuid.NewString(),
				StoreID:        storeID,
				RoomTemplateID: s.RoomTemplateID,
				UnitNumber:     n,
				Name:           fmt.Sprintf("%s %d", tmpl.Name, n),
				IsActive:       true,
				CreatedBy:      &actor,
			})
		}
	}
	return out, nil
}

func DeleteStore(id string, deletedBy string) error {
	store, err := repository.FindStoreByID(id)
	if err != nil {
		return errors.New("store tidak ditemukan")
	}
	now := time.Now()
	store.DeletedAt = &now
	return repository.SoftDeleteStore(store, deletedBy)
}
