package repository

import (
	"game_lounge_be/config"
	"game_lounge_be/models"
)

// FindAll mengambil semua template notifikasi.
func FindAll() ([]models.NotificationTemplate, error) {
	var templates []models.NotificationTemplate
	err := config.DB.Order("id ASC").Find(&templates).Error
	return templates, err
}

// FindByKey mengambil satu template berdasarkan notification_key.
// Fungsi ini juga dipakai oleh service kirim email/WA.
func FindByKey(key string) (*models.NotificationTemplate, error) {
	var t models.NotificationTemplate
	err := config.DB.Where("notification_key = ?", key).First(&t).Error
	return &t, err
}

// UpdateByKey menyimpan perubahan template berdasarkan key.
func UpdateByKey(key string, updates map[string]interface{}) (*models.NotificationTemplate, error) {
	t, err := FindByKey(key)
	if err != nil {
		return nil, err
	}

	if err := config.DB.Model(t).Updates(updates).Error; err != nil {
		return nil, err
	}

	return FindByKey(key)
}
