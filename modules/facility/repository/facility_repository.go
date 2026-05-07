package repository

import (
	"game_lounge_be/config"
	"game_lounge_be/models"
)

// ── Category ──────────────────────────────────────────────────

func FindAllCategories() ([]models.FacilityCategory, error) {
	var categories []models.FacilityCategory
	err := config.DB.Where("deleted_at IS NULL").Order("name ASC").Find(&categories).Error
	return categories, err
}

func FindCategoryByID(id uint) (*models.FacilityCategory, error) {
	var cat models.FacilityCategory
	err := config.DB.Where("id = ? AND deleted_at IS NULL", id).First(&cat).Error
	return &cat, err
}

func CreateCategory(cat *models.FacilityCategory) error {
	return config.DB.Create(cat).Error
}

func UpdateCategory(cat *models.FacilityCategory) error {
	return config.DB.Save(cat).Error
}

func SoftDeleteCategory(cat *models.FacilityCategory, deletedBy string) error {
	return config.DB.Model(cat).Updates(map[string]interface{}{
		"deleted_by": deletedBy,
		"deleted_at": config.DB.NowFunc(),
	}).Error
}

// ── Facility ──────────────────────────────────────────────────

func FindAllFacilities(search string, categoryID uint, page, perPage int) ([]models.Facility, int64, error) {
	var facilities []models.Facility
	var total int64

	query := config.DB.Model(&models.Facility{}).
		Preload("Category").
		Where("facilities.deleted_at IS NULL")

	if search != "" {
		query = query.Where("facilities.name LIKE ?", "%"+search+"%")
	}
	if categoryID > 0 {
		query = query.Where("facilities.category_id = ?", categoryID)
	}

	query.Count(&total)
	err := query.Offset((page-1)*perPage).Limit(perPage).
		Order("facilities.name ASC").Find(&facilities).Error

	return facilities, total, err
}

func FindAllFacilitiesForTemplate() ([]models.Facility, error) {
	var facilities []models.Facility
	err := config.DB.Preload("Category").
		Where("deleted_at IS NULL AND is_active = true").
		Order("name ASC").Find(&facilities).Error
	return facilities, err
}

func FindFacilityByID(id uint) (*models.Facility, error) {
	var facility models.Facility
	err := config.DB.Preload("Category").
		Where("id = ? AND deleted_at IS NULL", id).First(&facility).Error
	return &facility, err
}

func CreateFacility(facility *models.Facility) error {
	return config.DB.Create(facility).Error
}

func UpdateFacility(facility *models.Facility) error {
	return config.DB.Save(facility).Error
}

func SoftDeleteFacility(facility *models.Facility, deletedBy string) error {
	return config.DB.Model(facility).Updates(map[string]interface{}{
		"deleted_by": deletedBy,
		"deleted_at": config.DB.NowFunc(),
	}).Error
}

func CountFacilitiesByCategory(categoryID uint) int64 {
	var count int64
	config.DB.Model(&models.Facility{}).
		Where("category_id = ? AND deleted_at IS NULL", categoryID).Count(&count)
	return count
}

func GetFacilityStats() map[string]int64 {
	var total, active, inactive int64
	config.DB.Model(&models.Facility{}).Where("deleted_at IS NULL").Count(&total)
	config.DB.Model(&models.Facility{}).Where("deleted_at IS NULL AND is_active = true").Count(&active)
	config.DB.Model(&models.Facility{}).Where("deleted_at IS NULL AND is_active = false").Count(&inactive)
	var catTotal int64
	config.DB.Model(&models.FacilityCategory{}).Where("deleted_at IS NULL").Count(&catTotal)
	return map[string]int64{"total": total, "active": active, "inactive": inactive, "total_categories": catTotal}
}
