package dto

// UpdateTemplateRequest payload untuk update template.
type UpdateTemplateRequest struct {
	EmailSubject     string `json:"email_subject"`
	EmailBody        string `json:"email_body"`
	WhatsappBody     string `json:"whatsapp_body"`
	IsEmailActive    bool   `json:"is_email_active"`
	IsWhatsappActive bool   `json:"is_whatsapp_active"`
}

// PreviewRequest payload untuk preview template dengan data contoh.
type PreviewRequest struct {
	NotificationKey string            `json:"notification_key" binding:"required"`
	Channel         string            `json:"channel" binding:"required,oneof=email whatsapp"`
	SampleVars      map[string]string `json:"sample_vars"` // opsional — override sample data default
}
