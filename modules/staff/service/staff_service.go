package service

import (
	"fmt"
	adminRecoveryService "game_lounge_be/modules/admin_recovery/service"

	"errors"
	"game_lounge_be/models"
	"game_lounge_be/modules/staff/dto"
	"game_lounge_be/modules/staff/repository"
	"game_lounge_be/utils"
	"time"

	"github.com/google/uuid"
)

func GetAllStaffs(filter dto.StaffFilter) ([]models.Staff, int64, error) {
	return repository.FindAllStaffs(filter.Search, filter.RoleID, filter.StoreID, filter.Page, filter.PerPage)
}

func GetStaffByID(id string) (*models.Staff, error) {
	staff, err := repository.FindStaffByID(id)
	if err != nil {
		return nil, errors.New("staff tidak ditemukan")
	}
	return staff, nil
}

func CreateStaff(req dto.CreateStaffRequest, actor Actor) (*models.Staff, error) {
	newRoleIsSystem, err := repository.IsSystemRole(req.RoleID)
	if err != nil {
		return nil, errors.New("role tidak ditemukan")
	}
	if err := checkStaffWrite(staffWrite{actor: actor, newRoleIsSystem: newRoleIsSystem}); err != nil {
		return nil, err
	}

	// Check username uniqueness
	if _, err := repository.FindStaffByUsername(req.Username); err == nil {
		return nil, errors.New("username sudah digunakan")
	}
	// Check email uniqueness
	if _, err := repository.FindStaffByEmail(req.Email); err == nil {
		return nil, errors.New("email sudah digunakan")
	}

	// Without a password the staff sets one through an invite link. Check the
	// link can be built before creating anything.
	invite := req.Password == ""
	var hash string
	if invite {
		if err := adminRecoveryService.StaffLinksConfigured(); err != nil {
			return nil, err
		}
		hash, err = adminRecoveryService.UnusablePasswordHash()
	} else {
		hash, err = utils.HashPassword(req.Password)
	}
	if err != nil {
		return nil, errors.New("gagal memproses password")
	}

	phone := req.Phone
	staff := &models.Staff{
		ID:           uuid.NewString(),
		RoleID:       req.RoleID,
		Username:     req.Username,
		Email:        req.Email,
		Phone:        &phone,
		PasswordHash: hash,
		IsAllStores:  req.IsAllStores,
		CreatedBy:    &actor.Username,
	}

	if err := repository.CreateStaff(staff); err != nil {
		return nil, errors.New("gagal membuat staff")
	}

	if !req.IsAllStores && len(req.StoreIDs) > 0 {
		repository.SyncStaffStores(staff.ID, req.StoreIDs)
	}

	created, err := repository.FindStaffByID(staff.ID)
	if err != nil {
		return nil, err
	}
	if invite {
		if err := adminRecoveryService.IssueStaffInvite(staff.ID); err != nil {
			// The staff exists; only the email failed. The caller still gets it.
			return created, fmt.Errorf("%w: %v", ErrInviteNotSent, err)
		}
	}
	return created, nil
}

// ErrInviteNotSent: the staff was created but the invite email failed.
// The super admin can send a new link with "Reset Password".
var ErrInviteNotSent = errors.New("staff dibuat, tapi link atur password belum terkirim")

func UpdateStaff(id string, req dto.UpdateStaffRequest, actor Actor) (*models.Staff, error) {
	staff, err := repository.FindStaffByID(id)
	if err != nil {
		return nil, errors.New("staff tidak ditemukan")
	}
	newRoleIsSystem, err := repository.IsSystemRole(req.RoleID)
	if err != nil {
		return nil, errors.New("role tidak ditemukan")
	}
	if err := checkStaffWrite(staffWrite{
		actor:           actor,
		targetIsSystem:  staff.Role.IsSystem,
		newRoleIsSystem: newRoleIsSystem,
		isSelf:          staff.ID == actor.ID,
		roleChanged:     staff.RoleID != req.RoleID,
	}); err != nil {
		return nil, err
	}

	// Check username uniqueness (exclude self)
	if existing, err := repository.FindStaffByUsername(req.Username); err == nil && existing.ID != id {
		return nil, errors.New("username sudah digunakan")
	}

	phone := req.Phone
	staff.RoleID = req.RoleID
	staff.Username = req.Username
	staff.Email = req.Email
	staff.Phone = &phone
	staff.IsAllStores = req.IsAllStores
	staff.UpdatedBy = &actor.Username

	if err := repository.UpdateStaff(staff); err != nil {
		return nil, errors.New("gagal update staff")
	}

	if !req.IsAllStores {
		repository.SyncStaffStores(staff.ID, req.StoreIDs)
	} else {
		repository.SyncStaffStores(staff.ID, []string{})
	}

	return repository.FindStaffByID(staff.ID)
}

func DeleteStaff(id string, actor Actor) error {
	staff, err := repository.FindStaffByID(id)
	if err != nil {
		return errors.New("staff tidak ditemukan")
	}
	if err := checkStaffWrite(staffWrite{actor: actor, targetIsSystem: staff.Role.IsSystem}); err != nil {
		return err
	}
	now := time.Now()
	staff.DeletedAt = &now
	return repository.SoftDeleteStaff(staff, actor.Username)
}

// ── Reset Password ────────────────────────────────────────────────────────────

// ResetPasswordResult adalah response setelah reset password staff.
type ResetPasswordResult struct {
	StaffID  string `json:"staff_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Message  string `json:"message"`
}

// ResetStaffPassword: the old password stops working, all sessions end, and the
// staff gets a one-time link (24h) to set a new password. No password is ever
// generated or emailed. See adminRecoveryService.IssueStaffPasswordReset.
func ResetStaffPassword(staffID string) (*ResetPasswordResult, error) {
	staff, err := repository.FindStaffByID(staffID)
	if err != nil {
		return nil, errors.New("staff tidak ditemukan")
	}
	if err := adminRecoveryService.IssueStaffPasswordReset(staff.ID); err != nil {
		return nil, err
	}
	return &ResetPasswordResult{
		StaffID:  staff.ID,
		Username: staff.Username,
		Email:    staff.Email,
		Message:  "Link atur password baru telah dikirim ke " + staff.Email,
	}, nil
}
