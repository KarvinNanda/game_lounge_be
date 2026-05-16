package repository

import (
	"time"

	"game_lounge_be/config"
	"game_lounge_be/models"
)

// ── Package ───────────────────────────────────────────────────────────────────

// FindAllPackages mengambil list paket dengan filter & pagination.
func FindAllPackages(search, storeID, status string, page, perPage int) ([]models.PlayCreditsPackage, int64, error) {
	var packages []models.PlayCreditsPackage
	var total int64

	query := config.DB.Model(&models.PlayCreditsPackage{}).
		Preload("PackageStores.Store").
		Where("deleted_at IS NULL")

	if search != "" {
		query = query.Where("name LIKE ?", "%"+search+"%")
	}
	if status == "active" {
		query = query.Where("is_active = true")
	} else if status == "inactive" {
		query = query.Where("is_active = false")
	}

	query.Count(&total)
	err := query.Offset((page-1)*perPage).Limit(perPage).
		Order("created_at DESC").Find(&packages).Error

	return packages, total, err
}

// FindAllActivePackages mengambil semua paket aktif (untuk dropdown assign credits).
func FindAllActivePackages() ([]models.PlayCreditsPackage, error) {
	var packages []models.PlayCreditsPackage
	err := config.DB.Preload("PackageStores.Store").
		Where("deleted_at IS NULL AND is_active = true").
		Order("total_hours ASC").Find(&packages).Error
	return packages, err
}

// FindPackageByID mengambil paket berdasarkan ID.
func FindPackageByID(id string) (*models.PlayCreditsPackage, error) {
	var pkg models.PlayCreditsPackage
	err := config.DB.Preload("PackageStores.Store").
		Where("id = ? AND deleted_at IS NULL", id).First(&pkg).Error
	return &pkg, err
}

// CreatePackage menyimpan paket baru.
func CreatePackage(pkg *models.PlayCreditsPackage) error {
	return config.DB.Create(pkg).Error
}

// UpdatePackage menyimpan perubahan paket.
func UpdatePackage(pkg *models.PlayCreditsPackage) error {
	return config.DB.Save(pkg).Error
}

// SoftDeletePackage soft-delete paket.
func SoftDeletePackage(pkg *models.PlayCreditsPackage, deletedBy string) error {
	return config.DB.Model(pkg).Updates(map[string]interface{}{
		"deleted_by": deletedBy,
		"deleted_at": time.Now(),
	}).Error
}

// SyncPackageStores replace store yang berlaku untuk paket.
func SyncPackageStores(packageID string, storeIDs []string) error {
	tx := config.DB.Begin()

	if err := tx.Where("package_id = ?", packageID).
		Delete(&models.PlayCreditsPackageStore{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	for _, sid := range storeIDs {
		row := models.PlayCreditsPackageStore{PackageID: packageID, StoreID: sid}
		if err := tx.Create(&row).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

// ── Member Credits ────────────────────────────────────────────────────────────

// FindAllMemberCredits mengambil list credits member dengan filter.
func FindAllMemberCredits(search, packageID, storeID, status, whatsapp string, page, perPage int) ([]models.CustomerPlayCredit, int64, error) {
	var credits []models.CustomerPlayCredit
	var total int64

	query := config.DB.Model(&models.CustomerPlayCredit{}).
		Preload("Customer").
		Preload("Package.PackageStores.Store").
		Joins("JOIN customers c ON c.id = customer_play_credits.customer_id").
		Where("customer_play_credits.deleted_at IS NULL").
		Where("c.deleted_at IS NULL")

	if search != "" {
		like := "%" + search + "%"
		query = query.Where("c.name LIKE ? OR c.email LIKE ?", like, like)
	}
	if whatsapp != "" {
		query = query.Where("c.whatsapp LIKE ?", "%"+whatsapp+"%")
	}
	if packageID != "" {
		query = query.Where("customer_play_credits.package_id = ?", packageID)
	}
	if status == "active" {
		query = query.Where("customer_play_credits.is_active = true AND customer_play_credits.expires_at > ?", time.Now())
	} else if status == "expired" {
		query = query.Where("customer_play_credits.expires_at <= ? OR customer_play_credits.is_active = false", time.Now())
	}

	query.Count(&total)
	err := query.Offset((page-1)*perPage).Limit(perPage).
		Order("customer_play_credits.purchased_at DESC").Find(&credits).Error

	return credits, total, err
}

// FindCreditByID mengambil credit berdasarkan ID.
func FindCreditByID(id string) (*models.CustomerPlayCredit, error) {
	var credit models.CustomerPlayCredit
	err := config.DB.Preload("Customer").Preload("Package.PackageStores.Store").
		Where("id = ? AND deleted_at IS NULL", id).First(&credit).Error
	return &credit, err
}

// CreateCredit menyimpan credit baru untuk customer.
func CreateCredit(credit *models.CustomerPlayCredit) error {
	return config.DB.Create(credit).Error
}

// UpdateCredit menyimpan perubahan credit.
func UpdateCredit(credit *models.CustomerPlayCredit) error {
	return config.DB.Save(credit).Error
}

// SoftDeleteCredit soft-delete credit.
func SoftDeleteCredit(credit *models.CustomerPlayCredit, deletedBy string) error {
	return config.DB.Model(credit).Updates(map[string]interface{}{
		"deleted_by": deletedBy,
		"deleted_at": time.Now(),
	}).Error
}

// DeductHours mengurangi remaining_hours saat booking menggunakan credits.
// Dipanggil dari modul booking nanti.
func DeductHours(creditID string, hours float64) error {
	return config.DB.Model(&models.CustomerPlayCredit{}).
		Where("id = ?", creditID).
		Update("remaining_hours", config.DB.Raw("remaining_hours - ?", hours)).Error
}
