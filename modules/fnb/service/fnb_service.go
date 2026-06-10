package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"game_lounge_be/config"
	"game_lounge_be/models"
	"game_lounge_be/modules/fnb/dto"
	"game_lounge_be/modules/fnb/repository"

	"github.com/google/uuid"
)

// ── Categories ────────────────────────────────────────────────

func GetAllCategories() ([]models.FnbCategory, error) {
	return repository.FindAllCategories()
}

func CreateCategory(req dto.CreateCategoryRequest, actor string) (*models.FnbCategory, error) {
	cat := &models.FnbCategory{
		Name:      req.Name,
		SortOrder: req.SortOrder,
		IsActive:  true,
		CreatedBy: &actor,
	}
	if req.Description != "" {
		desc := req.Description
		cat.Description = &desc
	}
	return cat, repository.CreateCategory(cat)
}

func UpdateCategory(id uint, req dto.UpdateCategoryRequest, actor string) (*models.FnbCategory, error) {
	cat, err := repository.FindCategoryByID(id)
	if err != nil {
		return nil, errors.New("kategori tidak ditemukan")
	}
	cat.Name      = req.Name
	cat.SortOrder = req.SortOrder
	cat.IsActive  = req.IsActive
	cat.UpdatedBy = &actor
	if req.Description != "" {
		cat.Description = &req.Description
	}
	return cat, repository.UpdateCategory(cat)
}

func DeleteCategory(id uint, actor string) error {
	if _, err := repository.FindCategoryByID(id); err != nil {
		return errors.New("kategori tidak ditemukan")
	}
	return repository.DeleteCategory(id, actor)
}

// ── Items ─────────────────────────────────────────────────────

func GetAllItems(categoryID uint, activeOnly bool) ([]models.FnbItem, error) {
	return repository.FindAllItems(categoryID, activeOnly)
}

func CreateItem(req dto.CreateItemRequest, actor string) (*models.FnbItem, error) {
	item := &models.FnbItem{
		CategoryID:  req.CategoryID,
		Name:        req.Name,
		Price:       req.Price,
		SortOrder:   req.SortOrder,
		IsAvailable: true,
		IsActive:    true,
		CreatedBy:   &actor,
	}
	if req.Description != "" {
		item.Description = &req.Description
	}
	return item, repository.CreateItem(item)
}

func UpdateItem(id uint, req dto.UpdateItemRequest, actor string) (*models.FnbItem, error) {
	item, err := repository.FindItemByID(id)
	if err != nil {
		return nil, errors.New("item tidak ditemukan")
	}
	item.CategoryID  = req.CategoryID
	item.Name        = req.Name
	item.Price       = req.Price
	item.SortOrder   = req.SortOrder
	item.IsAvailable = req.IsAvailable
	item.IsActive    = req.IsActive
	item.UpdatedBy   = &actor
	if req.Description != "" {
		item.Description = &req.Description
	}
	return item, repository.UpdateItem(item)
}

// ── Orders ────────────────────────────────────────────────────

// CreateFnbOrder membuat pesanan FnB berdasarkan booking aktif customer.
func CreateFnbOrder(req dto.CreateFnbOrderRequest, customerID string) (*models.FnbOrder, error) {
	// Validasi booking aktif milik customer ini (status = ongoing)
	var booking models.Booking
	if err := config.DB.Preload("Room").
		Where("id = ? AND customer_id = ? AND status = 'ongoing'",
			req.BookingID, customerID).
		First(&booking).Error; err != nil {
		return nil, errors.New("tidak ada sesi bermain aktif untuk booking ini. Pastikan sesi sedang berlangsung")
	}

	// Hitung total + buat order items
	var orderItems []models.FnbOrderItem
	totalAmount := 0.0

	for _, reqItem := range req.Items {
		item, err := repository.FindItemByID(reqItem.ItemID)
		if err != nil {
			return nil, fmt.Errorf("item ID %d tidak ditemukan", reqItem.ItemID)
		}
		if !item.IsActive || !item.IsAvailable {
			return nil, fmt.Errorf("item '%s' tidak tersedia saat ini", item.Name)
		}

		subtotal := item.Price * float64(reqItem.Quantity)
		totalAmount += subtotal

		oi := models.FnbOrderItem{
			ItemID:   item.ID,
			ItemName: item.Name,   // snapshot nama saat order
			Quantity: reqItem.Quantity,
			Price:    item.Price, // snapshot harga saat order
		}
		if reqItem.Notes != "" {
			n := reqItem.Notes
			oi.Notes = &n
		}
		orderItems = append(orderItems, oi)
	}

	order := &models.FnbOrder{
		ID:          uuid.NewString(),
		BookingID:   req.BookingID,
		CustomerID:  customerID,
		StoreID:     booking.StoreID,
		RoomID:      booking.RoomID,
		Status:      "pending",
		TotalAmount: totalAmount,
		Items:       orderItems,
	}
	if req.Notes != "" {
		n := req.Notes
		order.Notes = &n
	}

	if err := repository.CreateOrder(order); err != nil {
		return nil, errors.New("gagal membuat pesanan FnB")
	}

	return repository.FindOrderByID(order.ID)
}

func UpdateOrderStatus(id, status string) (*models.FnbOrder, error) {
	if _, err := repository.FindOrderByID(id); err != nil {
		return nil, errors.New("order tidak ditemukan")
	}
	if err := repository.UpdateOrderStatus(id, status); err != nil {
		return nil, errors.New("gagal update status order")
	}
	return repository.FindOrderByID(id)
}

// ── Moka Sync ─────────────────────────────────────────────────

// SyncFromMoka mengambil menu dari Moka POS API dan upsert ke DB kita.
// Mengembalikan jumlah kategori dan item yang berhasil di-sync.
func SyncFromMoka() (catSynced, itemSynced int, err error) {
	baseURL  := os.Getenv("MOKA_BASE_URL")
	token    := os.Getenv("MOKA_ACCESS_TOKEN")
	outletID := os.Getenv("MOKA_OUTLET_ID")

	if token == "" || outletID == "" {
		return 0, 0, errors.New("MOKA_ACCESS_TOKEN dan MOKA_OUTLET_ID belum dikonfigurasi di .env")
	}
	if baseURL == "" {
		baseURL = "https://api.mokapos.com"
	}

	// Helper untuk call Moka API
	callMoka := func(path string, target interface{}) error {
		req, _ := http.NewRequest("GET", baseURL+path+"?outlet_id="+outletID, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 30 * time.Second}
		resp, reqErr := client.Do(req)
		if reqErr != nil {
			return reqErr
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		return json.Unmarshal(body, target)
	}

	// 1. Sync categories
	var catResp struct {
		Data []dto.MokaCategory `json:"data"`
	}
	if callErr := callMoka("/v2/item_categories", &catResp); callErr == nil {
		actor := "moka_sync"
		for _, mokaCat := range catResp.Data {
			mokaID := mokaCat.ID
			existing, findErr := repository.FindCategoryByMokaID(mokaCat.ID)
			if findErr != nil {
				// Buat baru
				newCat := &models.FnbCategory{
					Name:           mokaCat.Name,
					MokaCategoryID: &mokaID,
					IsActive:       true,
					CreatedBy:      &actor,
				}
				repository.CreateCategory(newCat) //nolint:errcheck
			} else {
				// Update nama saja
				existing.Name      = mokaCat.Name
				existing.UpdatedBy = &actor
				repository.UpdateCategory(existing) //nolint:errcheck
			}
			catSynced++
		}
	}

	// 2. Sync items
	var itemResp struct {
		Data []dto.MokaItem `json:"data"`
	}
	if callErr := callMoka("/v2/items", &itemResp); callErr == nil {
		actor := "moka_sync"
		for _, mokaItem := range itemResp.Data {
			// Cari category_id lokal berdasarkan moka category id
			var localCatID uint
			if cat, catErr := repository.FindCategoryByMokaID(mokaItem.CategoryID); catErr == nil {
				localCatID = cat.ID
			}
			if localCatID == 0 {
				continue // skip jika kategori belum ada
			}

			mokaID := mokaItem.ID
			existing, findErr := repository.FindItemByMokaID(mokaItem.ID)
			if findErr != nil {
				// Buat baru
				newItem := &models.FnbItem{
					CategoryID:  localCatID,
					Name:        mokaItem.Name,
					Price:       mokaItem.Price,
					MokaItemID:  &mokaID,
					IsAvailable: true,
					IsActive:    true,
					CreatedBy:   &actor,
				}
				if mokaItem.Description != "" {
					newItem.Description = &mokaItem.Description
				}
				if mokaItem.ImageURL != "" {
					newItem.ImageURL = &mokaItem.ImageURL
				}
				repository.CreateItem(newItem) //nolint:errcheck
			} else {
				// Update data
				existing.Name       = mokaItem.Name
				existing.Price      = mokaItem.Price
				existing.CategoryID = localCatID
				existing.UpdatedBy  = &actor
				if mokaItem.Description != "" {
					existing.Description = &mokaItem.Description
				}
				if mokaItem.ImageURL != "" {
					existing.ImageURL = &mokaItem.ImageURL
				}
				repository.UpdateItem(existing) //nolint:errcheck
			}
			itemSynced++
		}
	}

	return catSynced, itemSynced, nil
}
