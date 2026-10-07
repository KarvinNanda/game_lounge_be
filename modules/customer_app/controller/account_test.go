package controller

import (
	"net/http"
	"testing"

	"game_lounge_be/config"
	"game_lounge_be/models"
	"game_lounge_be/utils"

	"github.com/google/uuid"
)

// Password lama salah adalah kesalahan input, bukan sesi invalid: 401 membuat
// FE mengira sesi habis dan me-logout customer.
func TestChangePassword_OldPasswordSalah_400(t *testing.T) {
	skipIfNoDB(t)
	hash, err := utils.HashPassword("benar-123456")
	if err != nil {
		t.Fatal(err)
	}
	cust := &models.Customer{ID: uuid.NewString(), Name: "TEST", Whatsapp: "0", PasswordHash: &hash}
	if err := config.DB.Create(cust).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { config.DB.Unscoped().Delete(&models.Customer{}, "id = ?", cust.ID) })

	w := serve(http.MethodPut, "/change-password", cust.ID,
		`{"old_password":"salah-123456","new_password":"baru-1234567","confirm_password":"baru-1234567"}`,
		ChangePassword)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d: %s", w.Code, w.Body.String())
	}
}
