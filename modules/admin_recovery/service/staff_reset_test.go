package service

import (
	"strings"
	"testing"
	"time"

	"game_lounge_be/config"
	"game_lounge_be/models"
	"game_lounge_be/utils"

	"github.com/google/uuid"
)

const oldPassword = "Old-password-123"

// newStaff: a non-super-admin staff with a known password.
func newStaff(t *testing.T) *models.Staff {
	t.Helper()
	role := models.Role{Name: "TEST Role " + uuid.NewString()[:6]}
	if err := config.DB.Create(&role).Error; err != nil {
		t.Fatal(err)
	}
	hash, _ := utils.HashPassword(oldPassword)
	s := &models.Staff{
		ID: uuid.NewString(), RoleID: role.ID, Username: "test." + uuid.NewString()[:8],
		Email: "test-" + uuid.NewString()[:8] + "@example.invalid", PasswordHash: hash, TokenVersion: 3,
	}
	if err := config.DB.Create(s).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		config.DB.Where("staff_id = ?", s.ID).Delete(&models.PasswordResetToken{})
		config.DB.Unscoped().Delete(&models.Staff{}, "id = ?", s.ID)
		config.DB.Unscoped().Delete(&models.Role{}, "id = ?", role.ID)
	})
	return s
}

func waitMail(t *testing.T) sentMail {
	t.Helper()
	select {
	case m := <-sent:
		return m
	case <-time.After(3 * time.Second):
		t.Fatal("no email sent")
	}
	return sentMail{}
}

func tokenIn(t *testing.T, body, base string) string {
	t.Helper()
	prefix := base + "/admin-recovery/"
	i := strings.Index(body, prefix)
	if i < 0 {
		t.Fatalf("email has no %s link:\n%s", prefix, body)
	}
	return strings.Fields(body[i+len(prefix):])[0]
}

func TestIssueStaffPasswordReset_KillsOldPasswordAndSendsLink(t *testing.T) {
	skipIfNoDB(t)
	t.Setenv("ADMIN_APP_URL", "https://admin.example.com")
	s := newStaff(t)

	if err := IssueStaffPasswordReset(s.ID); err != nil {
		t.Fatal(err)
	}
	mail := waitMail(t)
	if mail.To != s.Email {
		t.Errorf("sent to %q, want %q", mail.To, s.Email)
	}
	if strings.Contains(mail.Body, oldPassword) {
		t.Error("email must not contain a password")
	}

	var after models.Staff
	config.DB.First(&after, "id = ?", s.ID)
	if utils.CheckPassword(oldPassword, after.PasswordHash) {
		t.Error("old password must stop working immediately")
	}
	if after.TokenVersion != s.TokenVersion+1 {
		t.Errorf("sessions must end: token_version %d -> %d", s.TokenVersion, after.TokenVersion)
	}

	token := tokenIn(t, mail.Body, "https://admin.example.com")
	var row models.PasswordResetToken
	if err := config.DB.Where("staff_id = ? AND used_at IS NULL", s.ID).First(&row).Error; err != nil {
		t.Fatal(err)
	}
	if row.Token != utils.HashToken(token) {
		t.Error("DB must store the hash of the token in the link")
	}
	if ttl := time.Until(row.ExpiresAt); ttl < 23*time.Hour || ttl > 24*time.Hour+time.Minute {
		t.Errorf("link must be valid for 24h, got %v", ttl)
	}

	// The staff (not a super admin) can use this link once.
	if err := ResetPassword(token, "New-password-456"); err != nil {
		t.Fatalf("staff reset link must work for a non-super-admin: %v", err)
	}
	config.DB.First(&after, "id = ?", s.ID)
	if !utils.CheckPassword("New-password-456", after.PasswordHash) {
		t.Error("new password not saved")
	}
	if err := ResetPassword(token, "Another-789"); err == nil {
		t.Error("link must be one-time")
	}
}

func TestIssueStaffPasswordReset_RevokesOlderLinks(t *testing.T) {
	skipIfNoDB(t)
	t.Setenv("ADMIN_APP_URL", "https://admin.example.com")
	s := newStaff(t)
	if err := IssueStaffPasswordReset(s.ID); err != nil {
		t.Fatal(err)
	}
	first := tokenIn(t, waitMail(t).Body, "https://admin.example.com")
	if err := IssueStaffPasswordReset(s.ID); err != nil {
		t.Fatal(err)
	}
	waitMail(t)
	if err := ValidateToken(first); err == nil {
		t.Error("an older link must stop working when a new one is issued")
	}
}

func TestResetPassword_SelfServiceTokenStillSuperAdminOnly(t *testing.T) {
	skipIfNoDB(t)
	s := newStaff(t)
	token := "selfservice-" + uuid.NewString()
	config.DB.Create(&models.PasswordResetToken{
		StaffID: s.ID, Token: utils.HashToken(token), IPAddress: "10.0.0.1", ExpiresAt: time.Now().Add(time.Hour),
	})
	if err := ResetPassword(token, "New-password-456"); err == nil {
		t.Error("a forgot-password token for a non-super-admin must still be rejected")
	}
}

func TestIssueStaffPasswordReset_RequiresAdminAppURL(t *testing.T) {
	skipIfNoDB(t)
	t.Setenv("ADMIN_APP_URL", "")
	s := newStaff(t)
	if err := IssueStaffPasswordReset(s.ID); err == nil {
		t.Error("without ADMIN_APP_URL the link cannot be opened; must fail before touching the password")
	}
	var after models.Staff
	config.DB.First(&after, "id = ?", s.ID)
	if !utils.CheckPassword(oldPassword, after.PasswordHash) {
		t.Error("a failed reset must leave the old password working")
	}
}

func TestIssueStaffInvite_LinkValid72hAndUsableByStaff(t *testing.T) {
	skipIfNoDB(t)
	t.Setenv("ADMIN_APP_URL", "https://admin.example.com")
	s := newStaff(t)

	if err := IssueStaffInvite(s.ID); err != nil {
		t.Fatal(err)
	}
	mail := waitMail(t)
	if !strings.Contains(mail.Subject, "Akun") || strings.Contains(mail.Body, "direset") {
		t.Errorf("invite must read as a new account, not a reset: %q", mail.Subject)
	}
	token := tokenIn(t, mail.Body, "https://admin.example.com")
	var row models.PasswordResetToken
	config.DB.Where("staff_id = ? AND used_at IS NULL", s.ID).First(&row)
	if ttl := time.Until(row.ExpiresAt); ttl < 71*time.Hour || ttl > 72*time.Hour+time.Minute {
		t.Errorf("invite link must be valid for 72h, got %v", ttl)
	}
	if err := ResetPassword(token, "First-password-1"); err != nil {
		t.Fatalf("invite link must work for a non-super-admin: %v", err)
	}
}

func TestStaffLinksConfigured(t *testing.T) {
	t.Setenv("ADMIN_APP_URL", "")
	if StaffLinksConfigured() == nil {
		t.Error("must fail without ADMIN_APP_URL")
	}
	t.Setenv("ADMIN_APP_URL", "https://admin.example.com")
	if err := StaffLinksConfigured(); err != nil {
		t.Error(err)
	}
}
