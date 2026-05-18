package service

import (
	"encoding/json"
	"errors"

	"game_lounge_be/models"
	"game_lounge_be/modules/notification_template/dto"
	"game_lounge_be/modules/notification_template/repository"
	"game_lounge_be/utils"
)

// ── CRUD ──────────────────────────────────────────────────────────────────────

// GetAll mengambil semua template.
func GetAll() ([]models.NotificationTemplate, error) {
	return repository.FindAll()
}

// GetByKey mengambil satu template berdasarkan key.
func GetByKey(key string) (*models.NotificationTemplate, error) {
	t, err := repository.FindByKey(key)
	if err != nil {
		return nil, errors.New("template tidak ditemukan")
	}
	return t, nil
}

// Update menyimpan perubahan template.
func Update(key string, req dto.UpdateTemplateRequest, updatedBy string) (*models.NotificationTemplate, error) {
	updates := map[string]interface{}{
		"email_subject":      req.EmailSubject,
		"email_body":         req.EmailBody,
		"whatsapp_body":      req.WhatsappBody,
		"is_email_active":    req.IsEmailActive,
		"is_whatsapp_active": req.IsWhatsappActive,
		"updated_by":         updatedBy,
	}

	result, err := repository.UpdateByKey(key, updates)
	if err != nil {
		return nil, errors.New("gagal menyimpan template")
	}
	return result, nil
}

// ── Preview ───────────────────────────────────────────────────────────────────

// defaultSampleVars mengembalikan data contoh default per notification_key.
func defaultSampleVars(key string) map[string]string {
	samples := map[string]map[string]string{
		"customer_welcome": {
			"nama_customer": "Viking Pratama",
			"email":         "viking@gmail.com",
			"password":      "v1k1ng$#Gl",
		},
		"voucher_notification": {
			"nama_customer":     "Viking Pratama",
			"nama_voucher":      "Diskon VIP 10%",
			"kode_voucher":      "VIP10",
			"berlaku_sampai":    "30 Juni 2026",
			"deskripsi_voucher": "Diskon 10% untuk booking VIP Room.",
		},
		"booking_confirmation": {
			"nama_customer": "Viking Pratama",
			"kode_booking":  "BK-260516-0001",
			"nama_ruangan":  "VIP Room 1 (Jelambar)",
			"tanggal":       "Sabtu, 16 Mei 2026",
			"jam_mulai":     "15:00",
			"jam_selesai":   "18:00",
			"durasi":        "3",
			"total_harga":   "135.000",
		},
	}
	if s, ok := samples[key]; ok {
		return s
	}
	return map[string]string{}
}

// Preview merender template dengan data contoh untuk ditampilkan ke admin.
func Preview(req dto.PreviewRequest) (string, error) {
	t, err := repository.FindByKey(req.NotificationKey)
	if err != nil {
		return "", errors.New("template tidak ditemukan")
	}

	// Merge sample vars: default + override dari request
	vars := defaultSampleVars(req.NotificationKey)
	for k, v := range req.SampleVars {
		vars[k] = v
	}

	var body string
	switch req.Channel {
	case "email":
		body = utils.GetTemplateOrFallback(t.EmailBody, "")
	case "whatsapp":
		body = utils.GetTemplateOrFallback(t.WhatsappBody, "")
	default:
		return "", errors.New("channel tidak valid")
	}

	return utils.RenderTemplate(body, vars), nil
}

// ── GetRendered — dipakai service lain ───────────────────────────────────────

// RenderedTemplate hasil render template siap kirim.
type RenderedTemplate struct {
	EmailSubject     string
	EmailBody        string
	WhatsappBody     string
	IsEmailActive    bool
	IsWhatsappActive bool
}

// GetRendered mengambil template dari DB dan merender semua variabel.
// Dipanggil dari: customer service, voucher service, booking service.
func GetRendered(key string, vars map[string]string) (*RenderedTemplate, error) {
	t, err := repository.FindByKey(key)
	if err != nil {
		return nil, errors.New("template notifikasi tidak ditemukan: " + key)
	}
	return &RenderedTemplate{
		EmailSubject:     utils.RenderTemplate(utils.GetTemplateOrFallback(t.EmailSubject, ""), vars),
		EmailBody:        utils.RenderTemplate(utils.GetTemplateOrFallback(t.EmailBody, ""), vars),
		WhatsappBody:     utils.RenderTemplate(utils.GetTemplateOrFallback(t.WhatsappBody, ""), vars),
		IsEmailActive:    t.IsEmailActive,
		IsWhatsappActive: t.IsWhatsappActive,
	}, nil
}

// ParseVariables mengubah JSON string available_variables menjadi slice.
func ParseVariables(jsonStr string) []map[string]string {
	var result []map[string]string
	_ = json.Unmarshal([]byte(jsonStr), &result)
	return result
}
