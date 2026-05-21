package service

import (
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

	if err := repository.CreateStore(store); err != nil {
		return nil, errors.New("gagal membuat store")
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
		repository.UpsertOperatingHours(store.ID, hours)
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
		repository.UpsertHolidaySchedules(store.ID, holidays)
	}

	// Setup rooms — auto-generate unit instances
	if len(req.Rooms) > 0 {
		if err := setupStoreRooms(store.ID, req.Rooms, createdBy); err != nil {
			return nil, err
		}
	}

	return GetStoreByID(store.ID)
}

func setupStoreRooms(storeID string, roomSetups []dto.RoomSetupInput, createdBy string) error {
	var rooms []models.StoreRoom
	for _, setup := range roomSetups {
		if setup.UnitCount <= 0 {
			continue
		}

		var tmpl models.RoomTemplate
		if err := config.DB.First(&tmpl, setup.RoomTemplateID).Error; err != nil {
			continue
		}

		for i := 1; i <= setup.UnitCount; i++ {
			rooms = append(rooms, models.StoreRoom{
				ID:             uuid.NewString(),
				StoreID:        storeID,
				RoomTemplateID: setup.RoomTemplateID,
				UnitNumber:     uint(i),
				Name:           fmt.Sprintf("%s %d", tmpl.Name, i),
				IsActive:       true,
				CreatedBy:      &createdBy,
			})
		}
	}

	if len(rooms) > 0 {
		return repository.CreateStoreRooms(rooms)
	}
	return nil
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

	if err := repository.UpdateStore(store); err != nil {
		return nil, errors.New("gagal update store")
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
		repository.UpsertOperatingHours(id, hours)
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
		repository.UpsertHolidaySchedules(id, holidays)
	}

	return GetStoreByID(id)
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
