package service

import (
	"errors"
	"game_lounge_be/models"
	"game_lounge_be/modules/facility/dto"
	"game_lounge_be/modules/facility/repository"
	"time"
)

// ── Category ──────────────────────────────────────────────────

func GetAllCategories() ([]models.FacilityCategory, error) {
	return repository.FindAllCategories()
}

func CreateCategory(req dto.CreateCategoryRequest, createdBy string) (*models.FacilityCategory, error) {
	icon := req.IconURL
	cat := &models.FacilityCategory{
		Name:      req.Name,
		IconURL:   &icon,
		CreatedBy: &createdBy,
	}
	if err := repository.CreateCategory(cat); err != nil {
		return nil, err
	}
	return cat, nil
}

func UpdateCategory(id uint, req dto.UpdateCategoryRequest, updatedBy string) (*models.FacilityCategory, error) {
	cat, err := repository.FindCategoryByID(id)
	if err != nil {
		return nil, errors.New("kategori tidak ditemukan")
	}

	cat.Name = req.Name
	cat.UpdatedBy = &updatedBy

	// Only update icon if new value provided
	if req.IconURL != "" {
		cat.IconURL = &req.IconURL
	}

	if err := repository.UpdateCategory(cat); err != nil {
		return nil, err
	}
	return cat, nil
}

func DeleteCategory(id uint, deletedBy string) error {
	cat, err := repository.FindCategoryByID(id)
	if err != nil {
		return errors.New("kategori tidak ditemukan")
	}
	if count := repository.CountFacilitiesByCategory(id); count > 0 {
		return errors.New("kategori masih digunakan oleh fasilitas")
	}
	now := time.Now()
	cat.DeletedAt = &now
	return repository.SoftDeleteCategory(cat, deletedBy)
}

// ── Facility ──────────────────────────────────────────────────

func GetAllFacilities(filter dto.FacilityFilter) ([]models.Facility, int64, map[string]int64, error) {
	facilities, total, err := repository.FindAllFacilities(filter.Search, filter.CategoryID, filter.Page, filter.PerPage)
	stats := repository.GetFacilityStats()
	return facilities, total, stats, err
}

func GetFacilityByID(id uint) (*models.Facility, error) {
	facility, err := repository.FindFacilityByID(id)
	if err != nil {
		return nil, errors.New("fasilitas tidak ditemukan")
	}
	return facility, nil
}

func CreateFacility(req dto.CreateFacilityRequest, createdBy string) (*models.Facility, error) {
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	desc := req.Description
	icon := req.IconURL

	facility := &models.Facility{
		CategoryID:  req.CategoryID,
		Name:        req.Name,
		Description: &desc,
		IconURL:     &icon,
		IsActive:    isActive,
		CreatedBy:   &createdBy,
	}

	if err := repository.CreateFacility(facility); err != nil {
		return nil, errors.New("gagal membuat fasilitas")
	}
	return repository.FindFacilityByID(facility.ID)
}

func UpdateFacility(id uint, req dto.UpdateFacilityRequest, updatedBy string) (*models.Facility, error) {
	facility, err := repository.FindFacilityByID(id)
	if err != nil {
		return nil, errors.New("fasilitas tidak ditemukan")
	}

	isActive := facility.IsActive
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	desc := req.Description
	facility.CategoryID = req.CategoryID
	facility.Name = req.Name
	facility.Description = &desc
	facility.IsActive = isActive
	facility.UpdatedBy = &updatedBy

	// Only update icon if new value provided
	if req.IconURL != "" {
		facility.IconURL = &req.IconURL
	}

	if err := repository.UpdateFacility(facility); err != nil {
		return nil, errors.New("gagal update fasilitas")
	}
	return repository.FindFacilityByID(facility.ID)
}

func DeleteFacility(id uint, deletedBy string) error {
	facility, err := repository.FindFacilityByID(id)
	if err != nil {
		return errors.New("fasilitas tidak ditemukan")
	}
	now := time.Now()
	facility.DeletedAt = &now
	return repository.SoftDeleteFacility(facility, deletedBy)
}
