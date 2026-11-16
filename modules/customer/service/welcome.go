package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"game_lounge_be/config"
	"game_lounge_be/models"
	ntService "game_lounge_be/modules/notification_template/service"
	"game_lounge_be/utils"

	"gorm.io/gorm"
)

// welcomeLinkTTL: how long the set-password link in the welcome message works.
// Longer than forgot-password (15 minutes) because the customer did not ask for it
// and may open the email much later.
const welcomeLinkTTL = 72 * time.Hour

// welcomeIPMarker fills customer_password_resets.ip_address for links created by
// an admin. It never equals a real IP, so it does not count toward the
// forgot-password limit of 3 requests per hour per IP.
const welcomeIPMarker = "admin-invite"

type welcomeMessage struct {
	Email, Name, Whatsapp, Link string
}

// sendWelcome is a variable so tests can replace it: the real one sends email.
var sendWelcome = sendWelcomeMessage

// issueSetPasswordLink revokes the customer's unused links and creates a new
// one-time link. Only the hash of the token is stored; the raw token exists only in the link.
func issueSetPasswordLink(customerID string) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	token := hex.EncodeToString(raw)

	err := config.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.CustomerPasswordReset{}).
			Where("customer_id = ? AND used_at IS NULL", customerID).
			Update("used_at", time.Now()).Error; err != nil {
			return err
		}
		return tx.Create(&models.CustomerPasswordReset{
			CustomerID: customerID,
			Token:      utils.HashToken(token),
			IPAddress:  welcomeIPMarker,
			ExpiresAt:  time.Now().Add(welcomeLinkTTL),
		}).Error
	})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/reset-password/%s", os.Getenv("APP_URL"), token), nil
}

// sendWelcomeMessage renders customer_welcome. If the template is missing, or is an
// old one without {{link_set_password}}, it sends the built-in email instead, so the
// customer always receives the link.
func sendWelcomeMessage(msg welcomeMessage) error {
	tmpl, err := ntService.GetRendered("customer_welcome", map[string]string{
		"nama_customer":     msg.Name,
		"email":             msg.Email,
		"link_set_password": msg.Link,
	})
	if useBuiltInWelcome(tmpl, err, msg.Link) {
		return utils.SendCustomerSetPasswordEmail(msg.Email, msg.Name, msg.Link)
	}
	if tmpl.IsWhatsappActive && msg.Whatsapp != "" {
		log.Print(whatsappPlaceholderLog(msg.Whatsapp))
	}
	if tmpl.IsEmailActive {
		return utils.SendEmail(msg.Email, msg.Name, tmpl.EmailSubject, tmpl.EmailBody)
	}
	return nil
}

// useBuiltInWelcome: no template, or an active email template that would not
// carry the link (an old body that still uses {{password}}).
func useBuiltInWelcome(tmpl *ntService.RenderedTemplate, err error, link string) bool {
	return err != nil || (tmpl.IsEmailActive && !strings.Contains(tmpl.EmailBody, link))
}

// whatsappPlaceholderLog never includes the message body (it holds the
// set-password link) and masks the number down to its last 4 digits.
func whatsappPlaceholderLog(wa string) string {
	masked := wa
	if len(wa) > 4 {
		masked = strings.Repeat("*", len(wa)-4) + wa[len(wa)-4:]
	}
	return "[Customer][WhatsApp placeholder] welcome message not sent (no WhatsApp provider) to " + masked
}
