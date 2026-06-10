package repository

import (
	"errors"
	"fmt"
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

// ValidateAndDeductPlayCredits memvalidasi play credits sebelum digunakan untuk booking.
// Cek: credits masih aktif, belum expired pada tanggal booking, sisa jam cukup.
// Menggunakan FIFO — kredit yang mau expired paling awal dipakai duluan.
func ValidateAndDeductPlayCredits(customerID, bookingDate string, durationHours float64) (*models.CustomerPlayCredit, error) {
	bookingDateParsed, err := time.Parse("2006-01-02", bookingDate)
	if err != nil {
		return nil, errors.New("format booking_date tidak valid")
	}

	// Cari play credits yang valid: aktif, belum expired pada tanggal booking, sisa jam cukup
	var credit models.CustomerPlayCredit
	dbErr := config.DB.
		Where(`customer_id = ?
            AND is_active = true
            AND remaining_hours >= ?
            AND expires_at >= ?
            AND deleted_at IS NULL`,
			customerID, durationHours, bookingDateParsed).
		Order("expires_at ASC"). // FIFO — yang mau expired duluan dipakai lebih awal
		First(&credit).Error

	if dbErr != nil {
		// Cek apakah ada credits tapi sudah expired
		var expiredCount int64
		config.DB.Model(&models.CustomerPlayCredit{}).
			Where("customer_id = ? AND is_active = true AND expires_at < ? AND deleted_at IS NULL",
				customerID, bookingDateParsed).
			Count(&expiredCount)
		if expiredCount > 0 {
			return nil, fmt.Errorf(
				"play credits kamu sudah expired untuk tanggal booking %s. Silakan beli paket credits baru",
				bookingDateParsed.Format("02 January 2006"),
			)
		}

		// Cek apakah ada credits aktif tapi jam tidak cukup
		var insufficientCount int64
		config.DB.Model(&models.CustomerPlayCredit{}).
			Where(`customer_id = ? AND is_active = true
                AND expires_at >= ? AND remaining_hours < ?
                AND deleted_at IS NULL`,
				customerID, bookingDateParsed, durationHours).
			Count(&insufficientCount)
		if insufficientCount > 0 {
			return nil, fmt.Errorf(
				"sisa jam play credits tidak cukup untuk booking %.0f jam",
				durationHours,
			)
		}

		return nil, errors.New("tidak ada play credits yang valid. Silakan beli paket credits terlebih dahulu")
	}

	// Deduct jam dari credits
	newRemaining := credit.RemainingHours - durationHours
	if err := config.DB.Model(&credit).Update("remaining_hours", newRemaining).Error; err != nil {
		return nil, errors.New("gagal menggunakan play credits")
	}

	// Non-aktifkan jika jam sudah habis
	if newRemaining <= 0 {
		config.DB.Model(&credit).Update("is_active", false) //nolint:errcheck
	}

	return &credit, nil
}
