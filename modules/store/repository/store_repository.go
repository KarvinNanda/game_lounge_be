package repository

import (
	"game_lounge_be/config"
	"game_lounge_be/models"

	"gorm.io/gorm"
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

func CreateStore(db *gorm.DB, store *models.Store) error {
	return db.Create(store).Error
}

func UpdateStore(db *gorm.DB, store *models.Store) error {
	// Omit associations so preloaded slices don't cascade into upsert
	return db.Omit(clause.Associations).Save(store).Error
}

// SoftDeleteStore soft-deletes the store and its rooms in one transaction.
// Rooms are soft-deleted (not hard-deleted) because bookings point at them.
func SoftDeleteStore(store *models.Store, deletedBy string) error {
	return config.DB.Transaction(func(tx *gorm.DB) error {
		now := tx.NowFunc()
		if err := tx.Model(store).Updates(map[string]interface{}{
			"deleted_by": deletedBy,
			"deleted_at": now,
		}).Error; err != nil {
			return err
		}
		return tx.Model(&models.StoreRoom{}).
			Where("store_id = ? AND deleted_at IS NULL", store.ID).
			Updates(map[string]interface{}{"is_active": false, "deleted_by": deletedBy, "deleted_at": now}).Error
	})
}

// Operating Hours
// UpsertOperatingHours replaces the store's hours. db may be a transaction; the
// nested Transaction then becomes a savepoint.
func UpsertOperatingHours(db *gorm.DB, storeID string, hours []models.StoreOperatingHour) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("store_id = ?", storeID).Delete(&models.StoreOperatingHour{}).Error; err != nil {
			return err
		}
		for _, h := range hours {
			h.StoreID = storeID
			if err := tx.Create(&h).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func FindOperatingHoursByStore(storeID string) ([]models.StoreOperatingHour, error) {
	var hours []models.StoreOperatingHour
	err := config.DB.Where("store_id = ? AND deleted_at IS NULL", storeID).Find(&hours).Error
	return hours, err
}

func FindOperatingHourByStoreAndDay(storeID string, day string) (*models.StoreOperatingHour, error) {
	var hours models.StoreOperatingHour
	err := config.DB.Where("store_id = ? AND day_type = ? AND is_active = 1 AND deleted_at IS NULL", storeID, day).First(&hours).Error
	return &hours, err
}

// Holiday Schedules
// UpsertHolidaySchedules replaces the store's holidays. db may be a transaction.
func UpsertHolidaySchedules(db *gorm.DB, storeID string, holidays []models.StoreHolidaySchedule) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("store_id = ?", storeID).Delete(&models.StoreHolidaySchedule{}).Error; err != nil {
			return err
		}
		for _, h := range holidays {
			h.StoreID = storeID
			if err := tx.Create(&h).Error; err != nil {
				return err
			}
		}
		return nil
	})
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

func CreateStoreRooms(db *gorm.DB, rooms []models.StoreRoom) error {
	return db.Create(&rooms).Error
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
	// Joined to stores as a second guard: rooms of a deleted store never count.
	config.DB.Model(&models.StoreRoom{}).
		Joins("JOIN stores ON stores.id = store_rooms.store_id AND stores.deleted_at IS NULL").
		Where("store_rooms.deleted_at IS NULL AND store_rooms.is_active = true").Count(&totalRooms)
	return map[string]int64{"total": total, "active": active, "inactive": inactive, "total_rooms": totalRooms}
}
