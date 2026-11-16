package main

import (
	"encoding/json"
	"log"
	"strings"

	"game_lounge_be/models"

	"gorm.io/gorm"
)

// The customer_welcome template used to send {{password}} in plain text. The app now
// sends a one-time set-password link as {{link_set_password}}. Templates are editable
// by admins, so existing rows are rewritten here instead of only changing the seed.
var welcomeTextReplacer = strings.NewReplacer(
	"Password: {{password}}", "Atur password: {{link_set_password}}",
	"{{password}}", "{{link_set_password}}",
	"Segera ganti password setelah login.", "Link berlaku 72 jam dan hanya bisa dipakai 1 kali.",
)

func upgradeWelcomeText(s string) string {
	return welcomeTextReplacer.Replace(s)
}

func upgradeWelcomeVars(jsonVars string) (string, error) {
	var vars []map[string]string
	if err := json.Unmarshal([]byte(jsonVars), &vars); err != nil {
		return "", err
	}
	changed := false
	for _, v := range vars {
		if v["key"] == "password" {
			v["key"], v["label"] = "link_set_password", "Link Atur Password"
			changed = true
		}
	}
	if !changed {
		return jsonVars, nil
	}
	b, err := json.Marshal(vars)
	return string(b), err
}

// upgradeWelcomeTemplate is idempotent: a template that is already upgraded is left alone.
func upgradeWelcomeTemplate(db *gorm.DB) {
	var t models.NotificationTemplate
	if err := db.Where("notification_key = ?", "customer_welcome").First(&t).Error; err != nil {
		return // not seeded yet; cmd/seed creates the new version
	}
	updates := map[string]interface{}{}
	for col, body := range map[string]*string{"email_body": t.EmailBody, "whatsapp_body": t.WhatsappBody} {
		if body != nil {
			if up := upgradeWelcomeText(*body); up != *body {
				updates[col] = up
			}
		}
	}
	vars, err := upgradeWelcomeVars(t.AvailableVariables)
	if err != nil {
		log.Fatalf("available_variables customer_welcome tidak valid: %v", err)
	}
	if vars != t.AvailableVariables {
		updates["available_variables"] = vars
	}
	if len(updates) == 0 {
		return
	}
	if err := db.Model(&models.NotificationTemplate{}).Where("id = ?", t.ID).Updates(updates).Error; err != nil {
		log.Fatalf("Gagal upgrade template customer_welcome: %v", err)
	}
	log.Println("Template customer_welcome: {{password}} diganti {{link_set_password}}")
}
