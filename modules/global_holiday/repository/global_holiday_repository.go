package repository

import (
	"time"

	"game_lounge_be/config"
	"game_lounge_be/models"
)

// FindAll mengambil global holiday dengan filter search, year, dan pagination.
func FindAll(search string, year, page, perPage int) ([]models.GlobalHolidaySchedule, int64, error) {
	var holidays []models.GlobalHolidaySchedule
	var total int64

	query := config.DB.Model(&models.GlobalHolidaySchedule{}).Where("deleted_at IS NULL")

	if search != "" {
		query = query.Where("name LIKE ?", "%"+search+"%")
	}
	if year > 0 {
		query = query.Where("YEAR(date) = ?", year)
	}

	query.Count(&total)
	err := query.Order("date ASC").
		Offset((page - 1) * perPage).Limit(perPage).
		Find(&holidays).Error

	return holidays, total, err
}

// FindByDate mengecek apakah tanggal tertentu adalah global holiday.
// Dipakai oleh pricing engine.
func FindByDate(date time.Time) (*models.GlobalHolidaySchedule, error) {
	var h models.GlobalHolidaySchedule
	err := config.DB.Where("DATE(date) = ? AND deleted_at IS NULL", date.Format("2006-01-02")).
		First(&h).Error
	return &h, err
}

// FindByID mengambil satu holiday berdasarkan ID.
func FindByID(id uint) (*models.GlobalHolidaySchedule, error) {
	var h models.GlobalHolidaySchedule
	err := config.DB.Where("id = ? AND deleted_at IS NULL", id).First(&h).Error
	return &h, err
}

// Create menyimpan global holiday baru.
func Create(h *models.GlobalHolidaySchedule) error {
	return config.DB.Create(h).Error
}

// Update menyimpan perubahan holiday.
func Update(h *models.GlobalHolidaySchedule) error {
	return config.DB.Save(h).Error
}

// SoftDelete soft-delete global holiday.
func SoftDelete(h *models.GlobalHolidaySchedule, deletedBy string) error {
	return config.DB.Model(h).Updates(map[string]interface{}{
		"deleted_by": deletedBy,
		"deleted_at": time.Now(),
	}).Error
}
