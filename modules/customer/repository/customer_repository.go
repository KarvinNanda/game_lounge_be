package repository

import (
	"game_lounge_be/config"
	"game_lounge_be/models"
)

// ── Find All ──────────────────────────────────────────────────────────────────

func FindAllCustomers(search, customerType, status string, page, perPage int) ([]models.Customer, int64, error) {
	var customers []models.Customer
	var total int64

	query := config.DB.Model(&models.Customer{}).Where("deleted_at IS NULL")

	if search != "" {
		like := "%" + search + "%"
		query = query.Where("name LIKE ? OR whatsapp LIKE ? OR email LIKE ?", like, like, like)
	}
	if customerType != "" {
		query = query.Where("type = ?", customerType)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	query.Count(&total)

	offset := (page - 1) * perPage
	err := query.
		Preload("FavoriteRoomTypes").
		Preload("FavoriteRoomTypes.RoomTemplate").
		Order("created_at DESC").
		Limit(perPage).Offset(offset).
		Find(&customers).Error

	return customers, total, err
}

// ── Find One ──────────────────────────────────────────────────────────────────

func FindCustomerByID(id string) (*models.Customer, error) {
	var customer models.Customer
	err := config.DB.
		Preload("FavoriteRoomTypes").
		Preload("FavoriteRoomTypes.RoomTemplate").
		Preload("BookingHistory").
		Preload("BookingHistory.Room").
		Where("id = ? AND deleted_at IS NULL", id).
		First(&customer).Error
	if err != nil {
		return nil, err
	}
	return &customer, nil
}

func FindCustomerByEmail(email string) (*models.Customer, error) {
	var customer models.Customer
	err := config.DB.Where("email = ? AND deleted_at IS NULL", email).First(&customer).Error
	if err != nil {
		return nil, err
	}
	return &customer, nil
}

func FindCustomerByWhatsapp(whatsapp string) (*models.Customer, error) {
	var customer models.Customer
	err := config.DB.Where("whatsapp = ? AND deleted_at IS NULL", whatsapp).First(&customer).Error
	if err != nil {
		return nil, err
	}
	return &customer, nil
}

// ── Write ─────────────────────────────────────────────────────────────────────

func CreateCustomer(customer *models.Customer) error {
	return config.DB.Create(customer).Error
}

func UpdateCustomer(customer *models.Customer) error {
	return config.DB.Save(customer).Error
}

func SoftDeleteCustomer(customer *models.Customer, deletedBy string) error {
	customer.DeletedBy = &deletedBy
	return config.DB.Save(customer).Error
}

// ── Favorite Room Types ───────────────────────────────────────────────────────

// SyncFavoriteRoomTypes menghapus semua favorit lama lalu insert yang baru.
func SyncFavoriteRoomTypes(customerID string, roomTemplateIDs []uint) error {
	// Hapus lama
	if err := config.DB.Where("customer_id = ?", customerID).
		Delete(&models.CustomerFavoriteRoomType{}).Error; err != nil {
		return err
	}
	if len(roomTemplateIDs) == 0 {
		return nil
	}

	var records []models.CustomerFavoriteRoomType
	for _, rtID := range roomTemplateIDs {
		records = append(records, models.CustomerFavoriteRoomType{
			CustomerID:     customerID,
			RoomTemplateID: rtID,
		})
	}
	return config.DB.Create(&records).Error
}
