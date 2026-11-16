package service

import (
	"errors"
	"strings"
	"testing"
	"time"

	"game_lounge_be/config"
	"game_lounge_be/models"
	"game_lounge_be/modules/customer/dto"
	ntService "game_lounge_be/modules/notification_template/service"
	"game_lounge_be/utils"

	"github.com/google/uuid"
)

func waitWelcome(t *testing.T) welcomeMessage {
	t.Helper()
	select {
	case msg := <-sentWelcome:
		return msg
	case <-time.After(3 * time.Second):
		t.Fatal("no welcome message sent")
	}
	return welcomeMessage{}
}

func cleanupCustomer(t *testing.T, id string) {
	t.Cleanup(func() {
		config.DB.Where("customer_id = ?", id).Delete(&models.CustomerPasswordReset{})
		config.DB.Unscoped().Delete(&models.Customer{}, "id = ?", id)
	})
}

// tokenFromLink returns the raw token at the end of {APP_URL}/reset-password/{token}.
func tokenFromLink(t *testing.T, link string) string {
	t.Helper()
	i := strings.LastIndex(link, "/reset-password/")
	if i < 0 {
		t.Fatalf("link %q has no /reset-password/ path", link)
	}
	return link[i+len("/reset-password/"):]
}

func TestCreateCustomer_SendsSetPasswordLinkNotPassword(t *testing.T) {
	skipIfNoDB(t)
	suffix := uuid.NewString()[:8]
	c, err := CreateCustomer(dto.CreateCustomerRequest{
		Name: "TEST", Whatsapp: "0899" + suffix, Email: "test-" + suffix + "@example.invalid", Type: "member",
	}, "test")
	if err != nil {
		t.Fatal(err)
	}
	cleanupCustomer(t, c.ID)

	var saved models.Customer
	config.DB.First(&saved, "id = ?", c.ID)
	if saved.PasswordHash != nil {
		t.Error("customer must not get a generated password; they set it through the link")
	}

	msg := waitWelcome(t)
	if msg.Email != *c.Email {
		t.Errorf("sent to %q, want %q", msg.Email, *c.Email)
	}
	var reset models.CustomerPasswordReset
	if err := config.DB.Where("customer_id = ? AND used_at IS NULL", c.ID).First(&reset).Error; err != nil {
		t.Fatalf("no unused set-password token: %v", err)
	}
	if reset.Token != utils.HashToken(tokenFromLink(t, msg.Link)) {
		t.Error("DB must store the hash of the token that is in the link")
	}
	if ttl := time.Until(reset.ExpiresAt); ttl < 71*time.Hour || ttl > 72*time.Hour+time.Minute {
		t.Errorf("link must be valid for 72h, got %v", ttl)
	}
}

func TestResendPassword_NewLinkKeepsPasswordAndRevokesOldLink(t *testing.T) {
	skipIfNoDB(t)
	hash := "existing-hash"
	cust := &models.Customer{
		ID: uuid.NewString(), Name: "TEST", Whatsapp: "0898" + uuid.NewString()[:8],
		Email: ptrStr("test-" + uuid.NewString()[:8] + "@example.invalid"), PasswordHash: &hash,
		Type: "member", Status: "active",
	}
	if err := config.DB.Create(cust).Error; err != nil {
		t.Fatal(err)
	}
	cleanupCustomer(t, cust.ID)
	old := &models.CustomerPasswordReset{
		CustomerID: cust.ID, Token: utils.HashToken("old-" + uuid.NewString()), IPAddress: "test",
		ExpiresAt: time.Now().Add(time.Hour),
	}
	config.DB.Create(old)

	if err := ResendPassword(cust.ID); err != nil {
		t.Fatal(err)
	}
	msg := waitWelcome(t)

	var saved models.Customer
	config.DB.First(&saved, "id = ?", cust.ID)
	if saved.PasswordHash == nil || *saved.PasswordHash != hash || saved.TokenVersion != cust.TokenVersion {
		t.Error("resend must not change the password or end sessions")
	}
	var oldNow models.CustomerPasswordReset
	config.DB.First(&oldNow, old.ID)
	if oldNow.UsedAt == nil {
		t.Error("previous unused link must be revoked")
	}
	var fresh models.CustomerPasswordReset
	if err := config.DB.Where("customer_id = ? AND used_at IS NULL", cust.ID).First(&fresh).Error; err != nil {
		t.Fatalf("no new link: %v", err)
	}
	if fresh.Token != utils.HashToken(tokenFromLink(t, msg.Link)) {
		t.Error("new link token does not match DB")
	}
}

func TestWhatsappPlaceholderLog_HidesBodyAndNumber(t *testing.T) {
	line := whatsappPlaceholderLog("081234567890")
	if strings.Contains(line, "081234567890") {
		t.Errorf("full number must be masked: %q", line)
	}
	if !strings.HasSuffix(strings.TrimSpace(line), "7890") {
		t.Errorf("last 4 digits help support find the customer: %q", line)
	}
}

func ptrStr(s string) *string { return &s }

func TestUseBuiltInWelcome(t *testing.T) {
	link := "http://app/reset-password/abc"
	cases := []struct {
		name string
		tmpl *ntService.RenderedTemplate
		err  error
		want bool
	}{
		{"no template", nil, errors.New("not found"), true},
		{"old body without link", &ntService.RenderedTemplate{IsEmailActive: true, EmailBody: "Password: {{password}}"}, nil, true},
		{"new body with link", &ntService.RenderedTemplate{IsEmailActive: true, EmailBody: "Atur password: " + link}, nil, false},
		{"email turned off by admin", &ntService.RenderedTemplate{IsEmailActive: false, EmailBody: "old"}, nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := useBuiltInWelcome(tc.tmpl, tc.err, link); got != tc.want {
				t.Errorf("want %v, got %v", tc.want, got)
			}
		})
	}
}
