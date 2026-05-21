package repository

import (
	"game_lounge_be/config"
	"game_lounge_be/models"
	"time"
)

// ── Pricing Config ────────────────────────────────────────────

// FindAllStores mengambil semua store (untuk list pricing).
func FindAllStores(search string, page, perPage int) ([]models.Store, int64, error) {
	var stores []models.Store
	var total int64

	query := config.DB.Model(&models.Store{}).Where("stores.deleted_at IS NULL")
	if search != "" {
		query = query.Where("stores.name LIKE ?", "%"+search+"%")
	}

	query.Count(&total)
	err := query.Offset((page-1)*perPage).Limit(perPage).
		Order("stores.name ASC").Find(&stores).Error

	return stores, total, err
}

// FindPricingByStoreID mengambil config pricing lengkap (dengan semua relasi) untuk 1 store.
func FindPricingByStoreID(storeID string) (*models.StorePricing, error) {
	var pricing models.StorePricing
	err := config.DB.
		Preload("Store").
		Preload("HappyHourSchedules", "deleted_at IS NULL").
		Preload("HappyHourPrices", "deleted_at IS NULL").
		Preload("HappyHourPrices.RoomTemplate").
		Preload("PackagePrices", "deleted_at IS NULL").
		Preload("PackagePrices.RoomTemplate").
		Preload("FlashSales", "deleted_at IS NULL").
		Preload("FlashSales.RoomTemplate").
		Where("store_id = ? AND deleted_at IS NULL", storeID).
		First(&pricing).Error
	return &pricing, err
}

// FindPricingConfigByStoreID mengambil config saja (tanpa relasi berat — untuk internal checks).
func FindPricingConfigByStoreID(storeID string) (*models.StorePricing, error) {
	var pricing models.StorePricing
	err := config.DB.Where("store_id = ? AND deleted_at IS NULL", storeID).First(&pricing).Error
	return &pricing, err
}

// CreatePricing membuat pricing config baru untuk store.
func CreatePricing(pricing *models.StorePricing) error {
	return config.DB.Create(pricing).Error
}

// UpdatePricing menyimpan perubahan pricing config.
func UpdatePricing(pricing *models.StorePricing) error {
	return config.DB.Save(pricing).Error
}

// SoftDeletePricing soft-delete pricing config.
func SoftDeletePricing(pricing *models.StorePricing, deletedBy string) error {
	return config.DB.Model(pricing).Updates(map[string]interface{}{
		"deleted_by": deletedBy,
		"deleted_at": config.DB.NowFunc(),
	}).Error
}

// ── Happy Hour Schedules ──────────────────────────────────────

// FindSchedulesByStoreID mengambil semua jadwal happy hour aktif.
func FindSchedulesByStoreID(storeID string) ([]models.StoreHappyHourSchedule, error) {
	var schedules []models.StoreHappyHourSchedule
	err := config.DB.Where("store_id = ? AND deleted_at IS NULL", storeID).
		Order("start_time ASC").Find(&schedules).Error
	return schedules, err
}

// CreateSchedule menambahkan jadwal happy hour baru.
func CreateSchedule(s *models.StoreHappyHourSchedule) error {
	return config.DB.Create(s).Error
}

// DeleteSchedule soft-delete jadwal happy hour.
func DeleteSchedule(id uint, storeID string, deletedBy string) error {
	return config.DB.Model(&models.StoreHappyHourSchedule{}).
		Where("id = ? AND store_id = ? AND deleted_at IS NULL", id, storeID).
		Updates(map[string]interface{}{
			"deleted_by": deletedBy,
			"deleted_at": time.Now(),
		}).Error
}

// ── Happy Hour Prices ─────────────────────────────────────────

// FindHappyHourPricesByStore mengambil semua harga HH untuk 1 store.
func FindHappyHourPricesByStore(storeID string) ([]models.StoreHappyHourPrice, error) {
	var prices []models.StoreHappyHourPrice
	err := config.DB.Preload("RoomTemplate").
		Where("store_id = ? AND deleted_at IS NULL", storeID).
		Order("room_template_id").Find(&prices).Error
	return prices, err
}

// UpsertHappyHourPrices insert-or-update harga HH (bulk).
func UpsertHappyHourPrices(storeID string, inputs []models.StoreHappyHourPrice) error {
	if len(inputs) == 0 {
		return nil
	}
	tx := config.DB.Begin()
	for _, p := range inputs {
		var existing models.StoreHappyHourPrice
		result := tx.Where("store_id = ? AND room_template_id = ? AND deleted_at IS NULL",
			storeID, p.RoomTemplateID).First(&existing)
		if result.Error != nil {
			// insert baru
			if err := tx.Create(&p).Error; err != nil {
				tx.Rollback()
				return err
			}
		} else {
			// update existing
			if err := tx.Model(&existing).Updates(map[string]interface{}{
				"price_per_hour": p.PricePerHour,
				"updated_by":     p.UpdatedBy,
			}).Error; err != nil {
				tx.Rollback()
				return err
			}
		}
	}
	return tx.Commit().Error
}

// ── Package Prices ────────────────────────────────────────────

// FindPackagePricesByStore mengambil semua harga paket untuk 1 store.
func FindPackagePricesByStore(storeID string) ([]models.StorePackagePrice, error) {
	var prices []models.StorePackagePrice
	err := config.DB.Preload("RoomTemplate").
		Where("store_id = ? AND deleted_at IS NULL", storeID).
		Order("room_template_id, duration_hours").Find(&prices).Error
	return prices, err
}

// UpsertPackagePrices insert-or-update harga paket (bulk).
func UpsertPackagePrices(storeID string, inputs []models.StorePackagePrice) error {
	if len(inputs) == 0 {
		return nil
	}
	tx := config.DB.Begin()
	for _, p := range inputs {
		var existing models.StorePackagePrice
		result := tx.Where("store_id = ? AND room_template_id = ? AND duration_hours = ? AND deleted_at IS NULL",
			storeID, p.RoomTemplateID, p.DurationHours).First(&existing)
		if result.Error != nil {
			if err := tx.Create(&p).Error; err != nil {
				tx.Rollback()
				return err
			}
		} else {
			if err := tx.Model(&existing).Updates(map[string]interface{}{
				"price":      p.Price,
				"is_custom":  p.IsCustom,
				"updated_by": p.UpdatedBy,
			}).Error; err != nil {
				tx.Rollback()
				return err
			}
		}
	}
	return tx.Commit().Error
}

// DeletePackagePrice soft-delete 1 paket harga.
func DeletePackagePrice(id uint, storeID string, deletedBy string) error {
	return config.DB.Model(&models.StorePackagePrice{}).
		Where("id = ? AND store_id = ? AND deleted_at IS NULL", id, storeID).
		Updates(map[string]interface{}{
			"deleted_by": deletedBy,
			"deleted_at": time.Now(),
		}).Error
}

// ── Flash Sales ───────────────────────────────────────────────

// FindFlashSalesByStore mengambil semua flash sale untuk 1 store.
func FindFlashSalesByStore(storeID string) ([]models.StoreFlashSale, error) {
	var sales []models.StoreFlashSale
	err := config.DB.Preload("RoomTemplate").
		Where("store_id = ? AND deleted_at IS NULL", storeID).
		Order("date_from DESC").Find(&sales).Error
	return sales, err
}

// FindActiveFlashSale mencari flash sale aktif untuk (store, room template, date, time).
// Flash sale sekarang menggunakan price_per_hour — irisan waktu dihitung di pricing engine.
func FindActiveFlashSale(storeID string, roomTemplateID uint, date time.Time, startTime, endTime string) (*models.StoreFlashSale, error) {
	var sale models.StoreFlashSale
	dateStr := date.Format("2006-01-02")
	err := config.DB.
		Where("store_id = ? AND room_template_id = ?", storeID, roomTemplateID).
		Where("is_active = true AND deleted_at IS NULL").
		Where("date_from <= ? AND date_to >= ?", dateStr, dateStr).
		Where("time_from < ? AND time_to > ?", endTime, startTime).
		First(&sale).Error
	return &sale, err
}

// IsStoreHoliday mengecek apakah tanggal tertentu adalah hari libur untuk store tertentu.
// Dipakai oleh pricing engine untuk mematikan happy hour di hari libur store.
func IsStoreHoliday(storeID string, date time.Time) bool {
	var count int64
	config.DB.Model(&models.StoreHolidaySchedule{}).
		Where("store_id = ? AND DATE(date) = ? AND deleted_at IS NULL",
			storeID, date.Format("2006-01-02")).
		Count(&count)
	return count > 0
}

// FindFlashSaleByID mencari flash sale berdasarkan id.
func FindFlashSaleByID(id string) (*models.StoreFlashSale, error) {
	var sale models.StoreFlashSale
	err := config.DB.Preload("RoomTemplate").
		Where("id = ? AND deleted_at IS NULL", id).First(&sale).Error
	return &sale, err
}

// CreateFlashSale membuat flash sale baru.
func CreateFlashSale(sale *models.StoreFlashSale) error {
	return config.DB.Create(sale).Error
}

// UpdateFlashSale menyimpan perubahan flash sale.
// Omit("RoomTemplate") penting: mencegah GORM override room_template_id
// dengan ID dari preloaded association (BelongsTo bug GORM).
func UpdateFlashSale(sale *models.StoreFlashSale) error {
	return config.DB.Omit("RoomTemplate").Save(sale).Error
}

// SoftDeleteFlashSale soft-delete flash sale.
func SoftDeleteFlashSale(id string, deletedBy string) error {
	return config.DB.Model(&models.StoreFlashSale{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(map[string]interface{}{
			"deleted_by": deletedBy,
			"deleted_at": time.Now(),
		}).Error
}

// ── Untuk Calculator ──────────────────────────────────────────

// FindPackagePricesForRoom mengambil semua paket harga untuk 1 store + room template.
func FindPackagePricesForRoom(storeID string, roomTemplateID uint) ([]models.StorePackagePrice, error) {
	var prices []models.StorePackagePrice
	err := config.DB.Where("store_id = ? AND room_template_id = ? AND deleted_at IS NULL",
		storeID, roomTemplateID).
		Order("duration_hours ASC").Find(&prices).Error
	return prices, err
}

// FindHappyHourPriceForRoom mengambil harga HH untuk 1 store + room template.
func FindHappyHourPriceForRoom(storeID string, roomTemplateID uint) (*models.StoreHappyHourPrice, error) {
	var price models.StoreHappyHourPrice
	err := config.DB.Where("store_id = ? AND room_template_id = ? AND deleted_at IS NULL",
		storeID, roomTemplateID).First(&price).Error
	return &price, err
}
