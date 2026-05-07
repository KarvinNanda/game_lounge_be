package repository

import (
	"game_lounge_be/config"
	"game_lounge_be/models"
)

func FindStaffByUsername(username string) (*models.Staff, error) {
	var staff models.Staff
	err := config.DB.Preload("Role").
		Where("username = ? AND deleted_at IS NULL", username).
		First(&staff).Error
	return &staff, err
}

func FindStaffByID(id string) (*models.Staff, error) {
	var staff models.Staff
	err := config.DB.Preload("Role").Preload("StaffStores.Store").
		Where("id = ? AND deleted_at IS NULL", id).
		First(&staff).Error
	return &staff, err
}

func FindPermissionsByRoleID(roleID uint) ([]string, error) {
	var perms []models.RolePermission
	if err := config.DB.Where("role_id = ?", roleID).Find(&perms).Error; err != nil {
		return nil, err
	}
	result := make([]string, len(perms))
	for i, p := range perms {
		result[i] = p.Permission
	}
	return result, nil
}
