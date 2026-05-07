package repository

import (
	"game_lounge_be/config"
	"game_lounge_be/models"

	"gorm.io/gorm/clause"
)

func FindAllStores(search, status string, page, perPage int) ([]models.Store, int64, error) {
	var stores []models.Store
	var total int64

	query := config.DB.Model(&models.Store{}).
		Preload("OperatingHours", "deleted_at IS NULL").
		Preload("HolidaySchedules", "deleted_at IS NULL").
		Where("deleted_at IS NULL")
	if search != "" {
		query = query.Where("name LIKE ? OR address LIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	query.Count(&total)
	err := query.Offset((page-1)*perPage).Limit(perPage).
		Order("created_at DESC").Find(&stores).Error

	return stores, total, err
}

func FindStoreByID(id string) (*models.Store, error) {
	var store models.Store
	err := config.DB.
		Preload("OperatingHours", "deleted_at IS NULL").
		Preload("HolidaySchedules", "deleted_at IS NULL").
		Preload("Rooms", "deleted_at IS NULL").
		Preload("Rooms.RoomTemplate").
		Preload("Rooms.RoomTemplate.Facilities").
		Preload("Rooms.RoomTemplate.Facilities.Category").
		Where("id = ? AND deleted_at IS NULL", id).First(&store).Error
	return &store, err
}

func CreateStore(store *models.Store) error {
	return config.DB.Create(store).Error
}

func UpdateStore(store *models.Store) error {
	// Omit associations so preloaded slices don't cascade into upsert
	return config.DB.Omit(clause.Associations).Save(store).Error
}

func SoftDeleteStore(store *models.Store, deletedBy string) error {
	return config.DB.Model(store).Updates(map[string]interface{}{
		"deleted_by": deletedBy,
		"deleted_at": config.DB.NowFunc(),
	}).Error
}

// Operating Hours
func UpsertOperatingHours(storeID string, hours []models.StoreOperatingHour) error {
	tx := config.DB.Begin()
	tx.Where("store_id = ?", storeID).Delete(&models.StoreOperatingHour{})
	for _, h := range hours {
		h.StoreID = storeID
		if err := tx.Create(&h).Error; err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit().Error
}

func FindOperatingHoursByStore(storeID string) ([]models.StoreOperatingHour, error) {
	var hours []models.StoreOperatingHour
	err := config.DB.Where("store_id = ? AND deleted_at IS NULL", storeID).Find(&hours).Error
	return hours, err
}

// Holiday Schedules
func UpsertHolidaySchedules(storeID string, holidays []models.StoreHolidaySchedule) error {
	tx := config.DB.Begin()
	tx.Where("store_id = ?", storeID).Delete(&models.StoreHolidaySchedule{})
	for _, h := range holidays {
		h.StoreID = storeID
		if err := tx.Create(&h).Error; err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit().Error
}

func FindHolidaysByStore(storeID string) ([]models.StoreHolidaySchedule, error) {
	var holidays []models.StoreHolidaySchedule
	err := config.DB.Where("store_id = ? AND deleted_at IS NULL", storeID).
		Order("date ASC").Find(&holidays).Error
	return holidays, err
}

// Store Rooms
func FindRoomsByStore(storeID string) ([]models.StoreRoom, error) {
	var rooms []models.StoreRoom
	err := config.DB.Preload("RoomTemplate").
		Where("store_id = ? AND deleted_at IS NULL", storeID).
		Order("room_template_id, unit_number").Find(&rooms).Error
	return rooms, err
}

func CreateStoreRooms(rooms []models.StoreRoom) error {
	return config.DB.Create(&rooms).Error
}

func DeleteStoreRoomsByTemplate(storeID string, templateID uint) error {
	return config.DB.Where("store_id = ? AND room_template_id = ?", storeID, templateID).
		Delete(&models.StoreRoom{}).Error
}

func CountRoomsByStore(storeID string) int64 {
	var count int64
	config.DB.Model(&models.StoreRoom{}).
		Where("store_id = ? AND deleted_at IS NULL AND is_active = true", storeID).Count(&count)
	return count
}

func GetStoreStats() map[string]int64 {
	var total, active, inactive int64
	config.DB.Model(&models.Store{}).Where("deleted_at IS NULL").Count(&total)
	config.DB.Model(&models.Store{}).Where("deleted_at IS NULL AND status = 'active'").Count(&active)
	config.DB.Model(&models.Store{}).Where("deleted_at IS NULL AND status = 'inactive'").Count(&inactive)
	var totalRooms int64
	config.DB.Model(&models.StoreRoom{}).Where("deleted_at IS NULL AND is_active = true").Count(&totalRooms)
	return map[string]int64{"total": total, "active": active, "inactive": inactive, "total_rooms": totalRooms}
}
