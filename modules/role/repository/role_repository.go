package repository

import (
	"game_lounge_be/config"
	"game_lounge_be/models"
)

func FindAllRoles(search string, page, perPage int) ([]models.Role, int64, error) {
	var roles []models.Role
	var total int64

	query := config.DB.Model(&models.Role{}).
		Preload("Permissions").
		Where("deleted_at IS NULL")
	if search != "" {
		query = query.Where("name LIKE ?", "%"+search+"%")
	}

	query.Count(&total)
	err := query.Offset((page - 1) * perPage).Limit(perPage).
		Order("created_at DESC").Find(&roles).Error

	return roles, total, err
}

func FindRoleByID(id uint) (*models.Role, error) {
	var role models.Role
	err := config.DB.Preload("Permissions").
		Where("id = ? AND deleted_at IS NULL", id).First(&role).Error
	return &role, err
}

func FindPermissionsByRoleID(roleID uint) ([]models.RolePermission, error) {
	var perms []models.RolePermission
	err := config.DB.Where("role_id = ?", roleID).Find(&perms).Error
	return perms, err
}

func CreateRole(role *models.Role) error {
	return config.DB.Create(role).Error
}

func UpdateRole(role *models.Role) error {
	return config.DB.Save(role).Error
}

func SyncPermissions(roleID uint, permissions []string) error {
	tx := config.DB.Begin()

	if err := tx.Where("role_id = ?", roleID).Delete(&models.RolePermission{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	for _, p := range permissions {
		if err := tx.Create(&models.RolePermission{RoleID: roleID, Permission: p}).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

func SoftDeleteRole(role *models.Role, deletedBy string) error {
	return config.DB.Model(role).Updates(map[string]interface{}{
		"deleted_by": deletedBy,
		"deleted_at": config.DB.NowFunc(),
	}).Error
}
