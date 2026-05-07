package repository

import (
	"game_lounge_be/config"
	"game_lounge_be/models"

	"gorm.io/gorm/clause"
)

func FindAllRoomTemplates(search string, page, perPage int) ([]models.RoomTemplate, int64, error) {
	var templates []models.RoomTemplate
	var total int64

	query := config.DB.Model(&models.RoomTemplate{}).
		Preload("Facilities").
		Preload("Facilities.Category").
		Where("deleted_at IS NULL")

	if search != "" {
		query = query.Where("name LIKE ?", "%"+search+"%")
	}

	query.Count(&total)
	err := query.Offset((page-1)*perPage).Limit(perPage).
		Order("created_at DESC").Find(&templates).Error

	return templates, total, err
}

func FindAllRoomTemplatesActive() ([]models.RoomTemplate, error) {
	var templates []models.RoomTemplate
	err := config.DB.Preload("Facilities").
		Preload("Facilities.Category").
		Where("deleted_at IS NULL AND is_active = true").
		Order("name ASC").Find(&templates).Error
	return templates, err
}

func FindRoomTemplateByID(id uint) (*models.RoomTemplate, error) {
	var tmpl models.RoomTemplate
	err := config.DB.Preload("Facilities").
		Preload("Facilities.Category").
		Where("id = ? AND deleted_at IS NULL", id).First(&tmpl).Error
	return &tmpl, err
}

func CreateRoomTemplate(tmpl *models.RoomTemplate) error {
	return config.DB.Create(tmpl).Error
}

func UpdateRoomTemplate(tmpl *models.RoomTemplate) error {
	// Omit associations — facilities are managed separately via SyncFacilities
	return config.DB.Omit(clause.Associations).Save(tmpl).Error
}

func SyncFacilities(tmpl *models.RoomTemplate, facilityIDs []uint) error {
	var facilities []models.Facility
	if len(facilityIDs) > 0 {
		config.DB.Where("id IN ?", facilityIDs).Find(&facilities)
	}
	return config.DB.Model(tmpl).Association("Facilities").Replace(facilities)
}

func SoftDeleteRoomTemplate(tmpl *models.RoomTemplate, deletedBy string) error {
	return config.DB.Model(tmpl).Updates(map[string]interface{}{
		"deleted_by": deletedBy,
		"deleted_at": config.DB.NowFunc(),
	}).Error
}

func GetRoomTemplateStats() map[string]int64 {
	var total, active, inactive int64
	config.DB.Model(&models.RoomTemplate{}).Where("deleted_at IS NULL").Count(&total)
	config.DB.Model(&models.RoomTemplate{}).Where("deleted_at IS NULL AND is_active = true").Count(&active)
	config.DB.Model(&models.RoomTemplate{}).Where("deleted_at IS NULL AND is_active = false").Count(&inactive)
	return map[string]int64{"total": total, "active": active, "inactive": inactive}
}
