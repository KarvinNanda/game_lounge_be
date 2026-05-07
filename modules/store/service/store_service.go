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
