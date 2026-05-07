package service

import (
	"errors"
	"game_lounge_be/models"
	"game_lounge_be/modules/role/dto"
	"game_lounge_be/modules/role/repository"
	"time"
)

type RoleWithPermissions struct {
	models.Role
	Permissions []string `json:"permissions"`
}

func GetAllRoles(filter dto.RoleFilter) ([]RoleWithPermissions, int64, error) {
	roles, total, err := repository.FindAllRoles(filter.Search, filter.Page, filter.PerPage)
	if err != nil {
		return nil, 0, err
	}

	var result []RoleWithPermissions
	for _, r := range roles {
		permStrings := make([]string, len(r.Permissions))
		for i, p := range r.Permissions {
			permStrings[i] = p.Permission
		}
		result = append(result, RoleWithPermissions{Role: r, Permissions: permStrings})
	}
	return result, total, nil
}

func GetRoleByID(id uint) (*RoleWithPermissions, error) {
	role, err := repository.FindRoleByID(id)
	if err != nil {
		return nil, errors.New("role tidak ditemukan")
	}

	permStrings := make([]string, len(role.Permissions))
	for i, p := range role.Permissions {
		permStrings[i] = p.Permission
	}

	return &RoleWithPermissions{Role: *role, Permissions: permStrings}, nil
}

func CreateRole(req dto.CreateRoleRequest, createdBy string) (*models.Role, error) {
	role := &models.Role{
		Name:        req.Name,
		IsSystem:    false,
		CreatedBy:   &createdBy,
	}

	if err := repository.CreateRole(role); err != nil {
		return nil, err
	}

	if len(req.Permissions) > 0 {
		repository.SyncPermissions(role.ID, req.Permissions)
	}

	return role, nil
}

func UpdateRole(id uint, req dto.UpdateRoleRequest, updatedBy string) (*models.Role, error) {
	role, err := repository.FindRoleByID(id)
	if err != nil {
		return nil, errors.New("role tidak ditemukan")
	}
	if role.IsSystem {
		return nil, errors.New("role sistem tidak dapat diubah")
	}

	role.Name = req.Name
	role.UpdatedBy = &updatedBy

	if err := repository.UpdateRole(role); err != nil {
		return nil, err
	}

	repository.SyncPermissions(role.ID, req.Permissions)
	return role, nil
}

func DeleteRole(id uint, deletedBy string) error {
	role, err := repository.FindRoleByID(id)
	if err != nil {
		return errors.New("role tidak ditemukan")
	}
	if role.IsSystem {
		return errors.New("role sistem tidak dapat dihapus")
	}

	now := time.Now()
	role.DeletedAt = &now
	return repository.SoftDeleteRole(role, deletedBy)
}
