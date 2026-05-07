package service

import (
	"errors"
	"game_lounge_be/modules/auth/dto"
	"game_lounge_be/modules/auth/repository"
	"game_lounge_be/utils"
)

func Login(req dto.LoginRequest) (*dto.LoginResponse, error) {
	staff, err := repository.FindStaffByUsername(req.Username)
	if err != nil {
		return nil, errors.New("username atau password salah")
	}

	if !utils.CheckPassword(req.Password, staff.PasswordHash) {
		return nil, errors.New("username atau password salah")
	}

	var permissions []string
	if !staff.Role.IsSystem {
		permissions, err = repository.FindPermissionsByRoleID(staff.RoleID)
		if err != nil {
			return nil, err
		}
	}

	token, err := utils.GenerateJWT(staff.ID, staff.Username, staff.RoleID, staff.Role.IsSystem, permissions)
	if err != nil {
		return nil, err
	}

	return &dto.LoginResponse{
		Token: token,
		Staff: staff,
	}, nil
}
