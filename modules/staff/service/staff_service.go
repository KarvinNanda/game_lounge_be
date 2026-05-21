package service

import (
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

func CreateStaff(req dto.CreateStaffRequest, createdBy string) (*models.Staff, error) {
	// Check username uniqueness
	if _, err := repository.FindStaffByUsername(req.Username); err == nil {
		return nil, errors.New("username sudah digunakan")
	}
	// Check email uniqueness
	if _, err := repository.FindStaffByEmail(req.Email); err == nil {
		return nil, errors.New("email sudah digunakan")
	}

	hash, err := utils.HashPassword(req.Password)
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
		CreatedBy:    &createdBy,
	}

	if err := repository.CreateStaff(staff); err != nil {
		return nil, errors.New("gagal membuat staff")
	}

	if !req.IsAllStores && len(req.StoreIDs) > 0 {
		repository.SyncStaffStores(staff.ID, req.StoreIDs)
	}

	return repository.FindStaffByID(staff.ID)
}

func UpdateStaff(id string, req dto.UpdateStaffRequest, updatedBy string) (*models.Staff, error) {
	staff, err := repository.FindStaffByID(id)
	if err != nil {
		return nil, errors.New("staff tidak ditemukan")
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
	staff.UpdatedBy = &updatedBy

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

func DeleteStaff(id string, deletedBy string) error {
	staff, err := repository.FindStaffByID(id)
	if err != nil {
		return errors.New("staff tidak ditemukan")
	}
	now := time.Now()
	staff.DeletedAt = &now
	return repository.SoftDeleteStaff(staff, deletedBy)
}

// ── Reset Password ────────────────────────────────────────────────────────────

// ResetPasswordResult adalah response setelah reset password staff.
type ResetPasswordResult struct {
	StaffID  string `json:"staff_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Message  string `json:"message"`
}

// ResetStaffPassword generate password baru dari username, simpan ke DB, kirim ke email staff.
func ResetStaffPassword(staffID string) (*ResetPasswordResult, error) {
	staff, err := repository.FindStaffByID(staffID)
	if err != nil {
		return nil, errors.New("staff tidak ditemukan")
	}

	newPassword := utils.GeneratePasswordFromName(staff.Username)

	hashedPassword, err := utils.HashPassword(newPassword)
	if err != nil {
		return nil, errors.New("gagal memproses password baru")
	}

	if err := repository.UpdatePassword(staff.ID, hashedPassword); err != nil {
		return nil, errors.New("gagal menyimpan password baru")
	}

	// Kirim email ke staff (async)
	username := staff.Username
	email := staff.Email
	go func() {
		subject := "Password Akun Quantum Anda Telah Direset"
		body := "Halo " + username + "!\n\nPassword akun Quantum Playstation Rental kamu telah direset oleh Super Admin.\n\nBerikut password baru kamu:\nUsername : " + username + "\nPassword : " + newPassword + "\n\nSegera login dan ganti password kamu setelah masuk.\n\nSalam,\nTim Quantum Gaming Center"
		if err := utils.SendEmail(email, username, subject, body); err != nil {
			_ = err // log sudah ada di SendEmail
		}
	}()

	return &ResetPasswordResult{
		StaffID:  staff.ID,
		Username: staff.Username,
		Email:    staff.Email,
		Message:  "Password baru telah dikirim ke " + staff.Email,
	}, nil
}
