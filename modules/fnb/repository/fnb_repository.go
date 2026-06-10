package repository

import (
	"time"

	"game_lounge_be/config"
	"game_lounge_be/models"
)

// ── Categories ────────────────────────────────────────────────

func FindAllCategories() ([]models.FnbCategory, error) {
	var cats []models.FnbCategory
	err := config.DB.Preload("Items").
		Where("deleted_at IS NULL").
		Order("sort_order ASC, id ASC").
		Find(&cats).Error
	return cats, err
}

func FindCategoryByID(id uint) (*models.FnbCategory, error) {
	var cat models.FnbCategory
	err := config.DB.Where("id = ? AND deleted_at IS NULL", id).First(&cat).Error
	return &cat, err
}

func CreateCategory(cat *models.FnbCategory) error { return config.DB.Create(cat).Error }
func UpdateCategory(cat *models.FnbCategory) error { return config.DB.Save(cat).Error }

func DeleteCategory(id uint, deletedBy string) error {
	now := time.Now()
	return config.DB.Model(&models.FnbCategory{}).Where("id = ?", id).
		Updates(map[string]interface{}{"deleted_at": now, "updated_by": deletedBy}).Error
}

// FindCategoryByMokaID digunakan untuk upsert saat sync Moka.
func FindCategoryByMokaID(mokaCatID string) (*models.FnbCategory, error) {
	var cat models.FnbCategory
	err := config.DB.Where("moka_category_id = ?", mokaCatID).First(&cat).Error
	return &cat, err
}

// ── Items ─────────────────────────────────────────────────────

func FindAllItems(categoryID uint, activeOnly bool) ([]models.FnbItem, error) {
	var items []models.FnbItem
	q := config.DB.Preload("Category").Where("deleted_at IS NULL")
	if categoryID > 0 {
		q = q.Where("category_id = ?", categoryID)
	}
	if activeOnly {
		q = q.Where("is_active = true AND is_available = true")
	}
	err := q.Order("sort_order ASC, id ASC").Find(&items).Error
	return items, err
}

func FindItemByID(id uint) (*models.FnbItem, error) {
	var item models.FnbItem
	err := config.DB.Preload("Category").
		Where("id = ? AND deleted_at IS NULL", id).First(&item).Error
	return &item, err
}

func CreateItem(item *models.FnbItem) error { return config.DB.Create(item).Error }
func UpdateItem(item *models.FnbItem) error { return config.DB.Save(item).Error }

// FindItemByMokaID digunakan untuk upsert saat sync Moka.
func FindItemByMokaID(mokaItemID string) (*models.FnbItem, error) {
	var item models.FnbItem
	err := config.DB.Where("moka_item_id = ?", mokaItemID).First(&item).Error
	return &item, err
}

// ── Orders ────────────────────────────────────────────────────

func FindAllOrders(storeID, status string, page, perPage int) ([]models.FnbOrder, int64, error) {
	var orders []models.FnbOrder
	var total int64

	q := config.DB.Model(&models.FnbOrder{}).
		Preload("Customer").
		Preload("Room.RoomTemplate").
		Preload("Items.Item")

	if storeID != "" {
		q = q.Where("store_id = ?", storeID)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}

	q.Count(&total)
	err := q.Order("created_at DESC").
		Offset((page - 1) * perPage).Limit(perPage).
		Find(&orders).Error

	return orders, total, err
}

func FindOrderByID(id string) (*models.FnbOrder, error) {
	var order models.FnbOrder
	err := config.DB.
		Preload("Customer").
		Preload("Room.RoomTemplate").
		Preload("Items.Item").
		Where("id = ?", id).First(&order).Error
	return &order, err
}

func FindOrdersByCustomer(customerID string) ([]models.FnbOrder, error) {
	var orders []models.FnbOrder
	err := config.DB.Preload("Items.Item").
		Where("customer_id = ?", customerID).
		Order("created_at DESC").Find(&orders).Error
	return orders, err
}

func CreateOrder(order *models.FnbOrder) error { return config.DB.Create(order).Error }

func UpdateOrderStatus(id, status string) error {
	return config.DB.Model(&models.FnbOrder{}).Where("id = ?", id).
		Update("status", status).Error
}
