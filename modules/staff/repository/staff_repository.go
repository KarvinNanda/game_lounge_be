package repository

import (
	"game_lounge_be/config"
	"game_lounge_be/models"
)

func FindAllStaffs(search string, roleID uint, storeID string, page, perPage int) ([]models.Staff, int64, error) {
	var staffs []models.Staff
	var total int64

	query := config.DB.Model(&models.Staff{}).
		Preload("Role").
		Preload("StaffStores.Store").
		Where("staffs.deleted_at IS NULL")

	if search != "" {
		query = query.Where("staffs.username LIKE ? OR staffs.email LIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if roleID > 0 {
		query = query.Where("staffs.role_id = ?", roleID)
	}
	if storeID != "" {
		query = query.Joins("JOIN staff_stores ss ON ss.staff_id = staffs.id").
			Where("ss.store_id = ?", storeID)
	}

	query.Count(&total)
	err := query.Offset((page-1)*perPage).Limit(perPage).
		Order("staffs.created_at DESC").Find(&staffs).Error

	return staffs, total, err
}

func FindStaffByID(id string) (*models.Staff, error) {
	var staff models.Staff
	err := config.DB.Preload("Role").Preload("StaffStores.Store").
		Where("id = ? AND deleted_at IS NULL", id).First(&staff).Error
	return &staff, err
}

func FindStaffByUsername(username string) (*models.Staff, error) {
	var staff models.Staff
	err := config.DB.Preload("Role").
		Where("username = ? AND deleted_at IS NULL", username).First(&staff).Error
	return &staff, err
}

func FindStaffByEmail(email string) (*models.Staff, error) {
	var staff models.Staff
	err := config.DB.Where("email = ? AND deleted_at IS NULL", email).First(&staff).Error
	return &staff, err
}

func FindPermissionsByRoleID(roleID uint) ([]string, error) {
	var perms []models.RolePermission
	err := config.DB.Where("role_id = ?", roleID).Find(&perms).Error
	if err != nil {
		return nil, err
	}
	result := make([]string, len(perms))
	for i, p := range perms {
		result[i] = p.Permission
	}
	return result, nil
}

func CreateStaff(staff *models.Staff) error {
	return config.DB.Create(staff).Error
}

func UpdateStaff(staff *models.Staff) error {
	return config.DB.Save(staff).Error
}

func SyncStaffStores(staffID string, storeIDs []string) error {
	tx := config.DB.Begin()

	if err := tx.Where("staff_id = ?", staffID).Delete(&models.StaffStore{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	for _, storeID := range storeIDs {
		if err := tx.Create(&models.StaffStore{StaffID: staffID, StoreID: storeID}).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

func SoftDeleteStaff(staff *models.Staff, deletedBy string) error {
	return config.DB.Model(staff).Updates(map[string]interface{}{
		"deleted_by": deletedBy,
		"deleted_at": config.DB.NowFunc(),
	}).Error
}


// UpdatePassword menyimpan password hash baru untuk staff.
func UpdatePassword(staffID, hashedPassword string) error {
	return config.DB.Model(&models.Staff{}).
		Where("id = ?", staffID).
		Update("password_hash", hashedPassword).Error
}
