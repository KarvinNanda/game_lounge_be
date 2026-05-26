package repository

import (
	"time"

	"game_lounge_be/config"
	"game_lounge_be/models"
)

// FindAllAdmin mengambil semua banner (aktif & nonaktif) untuk admin.
func FindAllAdmin() ([]models.Banner, error) {
	var banners []models.Banner
	err := config.DB.Where("deleted_at IS NULL").
		Order("sort_order ASC, id ASC").Find(&banners).Error
	return banners, err
}

// FindAllActive mengambil banner aktif saja untuk customer (public endpoint).
func FindAllActive() ([]models.Banner, error) {
	var banners []models.Banner
	err := config.DB.Where("is_active = true AND deleted_at IS NULL").
		Order("sort_order ASC, id ASC").Find(&banners).Error
	return banners, err
}

// FindByID mengambil satu banner berdasarkan ID (belum dihapus).
func FindByID(id uint) (*models.Banner, error) {
	var banner models.Banner
	err := config.DB.Where("id = ? AND deleted_at IS NULL", id).First(&banner).Error
	return &banner, err
}

// Create menyimpan banner baru.
// Workaround GORM zero-value bool: jika IsActive=false, lakukan explicit update
// setelah create karena GORM akan skip field false saat INSERT (default:true di DB).
func Create(banner *models.Banner) error {
	if err := config.DB.Create(banner).Error; err != nil {
		return err
	}
	if !banner.IsActive {
		return config.DB.Model(banner).Update("is_active", false).Error
	}
	return nil
}

// Update menyimpan seluruh field banner (Save menulis semua kolom — bool aman).
func Update(banner *models.Banner) error {
	return config.DB.Save(banner).Error
}

// SoftDelete mengisi deleted_at dan deleted_by tanpa menghapus baris dari DB.
func SoftDelete(banner *models.Banner, deletedBy string) error {
	return config.DB.Model(banner).Updates(map[string]interface{}{
		"deleted_at": time.Now(),
		"deleted_by": deletedBy,
	}).Error
}

// Reorder mengupdate sort_order banyak banner sekaligus dalam satu transaksi.
func Reorder(orders []struct{ ID, SortOrder uint }) error {
	tx := config.DB.Begin()
	for _, o := range orders {
		if err := tx.Model(&models.Banner{}).
			Where("id = ?", o.ID).
			Update("sort_order", o.SortOrder).Error; err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit().Error
}
