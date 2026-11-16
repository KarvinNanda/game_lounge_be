package service

import (
	"testing"

	"game_lounge_be/config"
	"game_lounge_be/models"
	"game_lounge_be/modules/staff/dto"
	"game_lounge_be/utils"

	"github.com/google/uuid"
)

var superAdmin = Actor{ID: "test-actor", Username: "test", IsSystem: true}

func createReq(t *testing.T, password string) dto.CreateStaffRequest {
	t.Helper()
	u := "test.create." + uuid.NewString()[:8]
	t.Cleanup(func() {
		var ids []string
		config.DB.Model(&models.Staff{}).Where("username = ?", u).Pluck("id", &ids)
		for _, id := range ids {
			config.DB.Where("staff_id = ?", id).Delete(&models.PasswordResetToken{})
			config.DB.Unscoped().Delete(&models.Staff{}, "id = ?", id)
		}
	})
	return dto.CreateStaffRequest{RoleID: 6, Username: u, Email: u + "@example.invalid", Password: password, IsAllStores: true}
}

func countStaff(username string) int64 {
	var n int64
	config.DB.Model(&models.Staff{}).Where("username = ?", username).Count(&n)
	return n
}

func TestCreateStaff_WithoutPasswordNeedsAdminAppURLFirst(t *testing.T) {
	skipIfNoDB(t)
	t.Setenv("ADMIN_APP_URL", "")
	req := createReq(t, "")
	if _, err := CreateStaff(req, superAdmin); err == nil {
		t.Fatal("without ADMIN_APP_URL the invite link cannot be built; must fail")
	}
	if n := countStaff(req.Username); n != 0 {
		t.Errorf("a rejected request must not create the staff, found %d", n)
	}
}

func TestCreateStaff_ExplicitPasswordStillWorks(t *testing.T) {
	skipIfNoDB(t)
	req := createReq(t, "Typed-pass-123")
	s, err := CreateStaff(req, superAdmin)
	if err != nil {
		t.Fatal(err)
	}
	var saved models.Staff
	config.DB.First(&saved, "id = ?", s.ID)
	if !utils.CheckPassword("Typed-pass-123", saved.PasswordHash) {
		t.Error("explicit password must be saved as before")
	}
	var links int64
	config.DB.Model(&models.PasswordResetToken{}).Where("staff_id = ?", s.ID).Count(&links)
	if links != 0 {
		t.Error("no invite link when the admin set a password")
	}
}
