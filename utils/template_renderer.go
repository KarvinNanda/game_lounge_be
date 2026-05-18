package utils

import "strings"

// RenderTemplate mengganti semua variabel {{key}} dalam template
// dengan nilai yang tersedia di map vars.
// Contoh: "Halo {{nama_customer}}" + {"nama_customer":"Viking"} → "Halo Viking"
func RenderTemplate(template string, vars map[string]string) string {
	result := template
	for key, value := range vars {
		result = strings.ReplaceAll(result, "{{"+key+"}}", value)
	}
	return result
}

// GetTemplateOrFallback mengembalikan isi template dari DB.
// Jika template nil atau kosong, kembalikan fallback string.
func GetTemplateOrFallback(template *string, fallback string) string {
	if template == nil || *template == "" {
		return fallback
	}
	return *template
}
