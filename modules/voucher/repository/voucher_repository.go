package repository

import (
	"time"

	"game_lounge_be/config"
	"game_lounge_be/models"
)

// ── List & Detail ─────────────────────────────────────────────────────────────

// FindAllVouchers mengambil list voucher dengan filter & pagination.
func FindAllVouchers(search, status, vtype, storeID string, page, perPage int) ([]models.Voucher, int64, error) {
	var vouchers []models.Voucher
	var total int64

	query := config.DB.Model(&models.Voucher{}).
		Preload("Stores.Store").
		Where("deleted_at IS NULL")

	if search != "" {
		like := "%" + search + "%"
		query = query.Where("name LIKE ? OR code LIKE ?", like, like)
	}
	if vtype != "" {
		query = query.Where("type = ?", vtype)
	}

	now := time.Now().Format("2006-01-02")
	switch status {
	case "active":
		query = query.Where("is_active = true AND (end_date IS NULL OR end_date >= ?)", now)
	case "expired":
		query = query.Where("end_date < ?", now)
	case "inactive":
		query = query.Where("is_active = false")
	}

	query.Count(&total)
	err := query.Offset((page-1)*perPage).Limit(perPage).
		Order("created_at DESC").Find(&vouchers).Error

	return vouchers, total, err
}

// FindVoucherByID mengambil voucher berdasarkan ID dengan semua relasi.
func FindVoucherByID(id string) (*models.Voucher, error) {
	var voucher models.Voucher
	err := config.DB.Preload("Stores.Store").Preload("Usages.Customer").
		Where("id = ? AND deleted_at IS NULL", id).First(&voucher).Error
	return &voucher, err
}

// FindVoucherByCode mengambil voucher berdasarkan kode.
func FindVoucherByCode(code string) (*models.Voucher, error) {
	var voucher models.Voucher
	err := config.DB.Preload("Stores.Store").
		Where("code = ? AND deleted_at IS NULL", code).First(&voucher).Error
	return &voucher, err
}

// IsCodeExists cek apakah kode sudah dipakai voucher lain.
func IsCodeExists(code, excludeID string) bool {
	var count int64
	q := config.DB.Model(&models.Voucher{}).Where("code = ? AND deleted_at IS NULL", code)
	if excludeID != "" {
		q = q.Where("id != ?", excludeID)
	}
	q.Count(&count)
	return count > 0
}

// ── Write ─────────────────────────────────────────────────────────────────────

// CreateVoucher menyimpan voucher baru.
func CreateVoucher(voucher *models.Voucher) error {
	return config.DB.Create(voucher).Error
}

// UpdateVoucher menyimpan perubahan voucher.
func UpdateVoucher(voucher *models.Voucher) error {
	return config.DB.Save(voucher).Error
}

// SoftDeleteVoucher soft-delete voucher.
func SoftDeleteVoucher(voucher *models.Voucher, deletedBy string) error {
	return config.DB.Model(voucher).Updates(map[string]interface{}{
		"deleted_by": deletedBy,
		"deleted_at": time.Now(),
	}).Error
}

// SyncVoucherStores replace cabang yang berlaku untuk voucher.
func SyncVoucherStores(voucherID string, storeIDs []string) error {
	tx := config.DB.Begin()
	if err := tx.Where("voucher_id = ?", voucherID).Delete(&models.VoucherStore{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	for _, sid := range storeIDs {
		if err := tx.Create(&models.VoucherStore{VoucherID: voucherID, StoreID: sid}).Error; err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit().Error
}

// ── Counter helpers ───────────────────────────────────────────────────────────

// IncrementUsedCount tambah used_count saat voucher dipakai.
func IncrementUsedCount(voucherID string) error {
	return config.DB.Model(&models.Voucher{}).Where("id = ?", voucherID).
		Update("used_count", config.DB.Raw("used_count + 1")).Error
}

// UpdateTotalSent update jumlah notifikasi terkirim.
func UpdateTotalSent(voucherID string, count int) error {
	return config.DB.Model(&models.Voucher{}).Where("id = ?", voucherID).
		Update("total_sent", count).Error
}

// ── Usage ─────────────────────────────────────────────────────────────────────

// CheckUsageExists cek apakah customer sudah pernah pakai voucher ini.
func CheckUsageExists(voucherID, customerID string) bool {
	var count int64
	config.DB.Model(&models.VoucherUsage{}).
		Where("voucher_id = ? AND customer_id = ?", voucherID, customerID).Count(&count)
	return count > 0
}

// CreateUsage catat pemakaian voucher.
func CreateUsage(usage *models.VoucherUsage) error {
	return config.DB.Create(usage).Error
}

// ── Stats & Member list ───────────────────────────────────────────────────────

// GetVoucherStats mengambil statistik voucher keseluruhan.
func GetVoucherStats() map[string]interface{} {
	var totalVoucher, totalActive, totalExpired int64
	var totalSent, totalUsed uint
	var totalDiscount float64

	now := time.Now().Format("2006-01-02")
	config.DB.Model(&models.Voucher{}).Where("deleted_at IS NULL").Count(&totalVoucher)
	config.DB.Model(&models.Voucher{}).
		Where("deleted_at IS NULL AND is_active = true AND (end_date IS NULL OR end_date >= ?)", now).
		Count(&totalActive)
	config.DB.Model(&models.Voucher{}).
		Where("deleted_at IS NULL AND end_date < ?", now).Count(&totalExpired)
	config.DB.Model(&models.Voucher{}).Where("deleted_at IS NULL").
		Select("COALESCE(SUM(total_sent), 0)").Scan(&totalSent)
	config.DB.Model(&models.Voucher{}).Where("deleted_at IS NULL").
		Select("COALESCE(SUM(used_count), 0)").Scan(&totalUsed)
	config.DB.Model(&models.VoucherUsage{}).
		Select("COALESCE(SUM(discount_amount), 0)").Scan(&totalDiscount)

	return map[string]interface{}{
		"total_voucher":  totalVoucher,
		"total_active":   totalActive,
		"total_expired":  totalExpired,
		"total_sent":     totalSent,
		"total_used":     totalUsed,
		"total_discount": totalDiscount,
	}
}

// FindAvailableVouchersForCustomer mengambil voucher yang bisa dipakai oleh customer
// pada store dan tipe transaksi tertentu.
// Filter:
//   - active + belum expired + sudah start
//   - belum pernah dipakai customer ini
//   - berlaku di store (is_all_stores OR store_id masuk VoucherStores)
//   - type cocok (voucher.type = useType OR voucher.type = 'both')
func FindAvailableVouchersForCustomer(customerID, storeID, useType string) ([]models.Voucher, error) {
	var vouchers []models.Voucher
	now := time.Now().Format("2006-01-02")

	query := config.DB.Preload("Stores.Store").
		Where("deleted_at IS NULL").
		Where("is_active = true").
		Where("start_date <= ?", now).
		Where("end_date IS NULL OR end_date >= ?", now).
		// Tipe harus cocok
		Where("type = ? OR type = 'both'", useType).
		// Belum pernah dipakai customer ini
		Where("id NOT IN (SELECT voucher_id FROM voucher_usages WHERE customer_id = ?)", customerID).
		// Berlaku di store ini
		Where("is_all_stores = true OR id IN (SELECT voucher_id FROM voucher_stores WHERE store_id = ?)", storeID).
		Order("created_at DESC")

	err := query.Find(&vouchers).Error
	return vouchers, err
}

// FindAllMembers mengambil semua customer bertipe 'member' yang aktif
// (untuk pengiriman notifikasi voucher).
func FindAllMembers() ([]models.Customer, error) {
	var customers []models.Customer
	err := config.DB.
		Where("type = 'member' AND status = 'active' AND deleted_at IS NULL").
		Find(&customers).Error
	return customers, err
}

// CountAllMembers menghitung total member aktif.
func CountAllMembers() int64 {
	var count int64
	config.DB.Model(&models.Customer{}).
		Where("type = 'member' AND status = 'active' AND deleted_at IS NULL").Count(&count)
	return count
}
