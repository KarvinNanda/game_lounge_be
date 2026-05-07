package service

import (
	"errors"
	"game_lounge_be/models"
	"game_lounge_be/modules/room_template/dto"
	"game_lounge_be/modules/room_template/repository"
	"time"
)

func GetAllRoomTemplates(filter dto.RoomTemplateFilter) ([]models.RoomTemplate, int64, map[string]int64, error) {
	templates, total, err := repository.FindAllRoomTemplates(filter.Search, filter.Page, filter.PerPage)
	stats := repository.GetRoomTemplateStats()
	return templates, total, stats, err
}

func GetRoomTemplateByID(id uint) (*models.RoomTemplate, error) {
	tmpl, err := repository.FindRoomTemplateByID(id)
	if err != nil {
		return nil, errors.New("room template tidak ditemukan")
	}
	return tmpl, nil
}

func GetActiveRoomTemplates() ([]models.RoomTemplate, error) {
	return repository.FindAllRoomTemplatesActive()
}

func CreateRoomTemplate(req dto.CreateRoomTemplateRequest, createdBy string) (*models.RoomTemplate, error) {
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	desc := req.Description
	img := req.ImageURL

	tmpl := &models.RoomTemplate{
		Name:        req.Name,
		CapacityMin: req.CapacityMin,
		CapacityMax: req.CapacityMax,
		Description: &desc,
		ImageURL:    &img,
		IsActive:    isActive,
		CreatedBy:   &createdBy,
	}

	if err := repository.CreateRoomTemplate(tmpl); err != nil {
		return nil, errors.New("gagal membuat room template")
	}

	if len(req.FacilityIDs) > 0 {
		repository.SyncFacilities(tmpl, req.FacilityIDs)
	}

	return repository.FindRoomTemplateByID(tmpl.ID)
}

func UpdateRoomTemplate(id uint, req dto.UpdateRoomTemplateRequest, updatedBy string) (*models.RoomTemplate, error) {
	tmpl, err := repository.FindRoomTemplateByID(id)
	if err != nil {
		return nil, errors.New("room template tidak ditemukan")
	}

	isActive := tmpl.IsActive
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	desc := req.Description
	tmpl.Name = req.Name
	tmpl.CapacityMin = req.CapacityMin
	tmpl.CapacityMax = req.CapacityMax
	tmpl.Description = &desc
	tmpl.IsActive = isActive
	tmpl.UpdatedBy = &updatedBy

	// Only update image if new value provided
	if req.ImageURL != "" {
		tmpl.ImageURL = &req.ImageURL
	}

	if err := repository.UpdateRoomTemplate(tmpl); err != nil {
		return nil, errors.New("gagal update room template")
	}

	repository.SyncFacilities(tmpl, req.FacilityIDs)

	return repository.FindRoomTemplateByID(tmpl.ID)
}

func DeleteRoomTemplate(id uint, deletedBy string) error {
	tmpl, err := repository.FindRoomTemplateByID(id)
	if err != nil {
		return errors.New("room template tidak ditemukan")
	}
	now := time.Now()
	tmpl.DeletedAt = &now
	return repository.SoftDeleteRoomTemplate(tmpl, deletedBy)
}
