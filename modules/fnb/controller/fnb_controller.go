package controller

import (
	"net/http"
	"strconv"

	"game_lounge_be/modules/fnb/dto"
	"game_lounge_be/modules/fnb/repository"
	"game_lounge_be/modules/fnb/service"
	"game_lounge_be/utils"

	"github.com/gin-gonic/gin"
)

// ── Public (tanpa auth) ───────────────────────────────────────

// GetPublicMenu mengambil semua kategori + item aktif untuk customer web.
func GetPublicMenu(c *gin.Context) {
	cats, _ := service.GetAllCategories()

	type publicItem struct {
		ID          uint    `json:"id"`
		Name        string  `json:"name"`
		Description *string `json:"description"`
		ImageURL    *string `json:"image_url"`
		Price       float64 `json:"price"`
		SortOrder   int     `json:"sort_order"`
	}
	type publicCategory struct {
		ID          uint         `json:"id"`
		Name        string       `json:"name"`
		Description *string      `json:"description"`
		ImageURL    *string      `json:"image_url"`
		Items       []publicItem `json:"items"`
	}

	result := make([]publicCategory, 0)
	for _, cat := range cats {
		if !cat.IsActive {
			continue
		}
		activeItems := make([]publicItem, 0)
		for _, item := range cat.Items {
			if item.IsActive && item.IsAvailable {
				activeItems = append(activeItems, publicItem{
					ID:          item.ID,
					Name:        item.Name,
					Description: item.Description,
					ImageURL:    item.ImageURL,
					Price:       item.Price,
					SortOrder:   item.SortOrder,
				})
			}
		}
		result = append(result, publicCategory{
			ID:          cat.ID,
			Name:        cat.Name,
			Description: cat.Description,
			ImageURL:    cat.ImageURL,
			Items:       activeItems,
		})
	}
	utils.ResponseSuccess(c, http.StatusOK, "OK", result)
}

// ── Admin: Categories ─────────────────────────────────────────

func AdminGetCategories(c *gin.Context) {
	cats, _ := service.GetAllCategories()
	utils.ResponseSuccess(c, http.StatusOK, "OK", cats)
}

func AdminCreateCategory(c *gin.Context) {
	var req dto.CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseValidationError(c, http.StatusBadRequest, "Validasi gagal", err.Error())
		return
	}
	cat, err := service.CreateCategory(req, c.GetString("staff_username"))
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusCreated, "Kategori berhasil dibuat", cat)
}

func AdminUpdateCategory(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req dto.UpdateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseValidationError(c, http.StatusBadRequest, "Validasi gagal", err.Error())
		return
	}
	cat, err := service.UpdateCategory(uint(id), req, c.GetString("staff_username"))
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "Kategori berhasil diupdate", cat)
}

func AdminDeleteCategory(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := service.DeleteCategory(uint(id), c.GetString("staff_username")); err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "Kategori berhasil dihapus", nil)
}

// ── Admin: Items ──────────────────────────────────────────────

func AdminGetItems(c *gin.Context) {
	catID, _ := strconv.Atoi(c.Query("category_id"))
	items, _  := service.GetAllItems(uint(catID), false)
	utils.ResponseSuccess(c, http.StatusOK, "OK", items)
}

func AdminCreateItem(c *gin.Context) {
	var req dto.CreateItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseValidationError(c, http.StatusBadRequest, "Validasi gagal", err.Error())
		return
	}
	item, err := service.CreateItem(req, c.GetString("staff_username"))
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusCreated, "Item berhasil dibuat", item)
}

func AdminUpdateItem(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req dto.UpdateItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseValidationError(c, http.StatusBadRequest, "Validasi gagal", err.Error())
		return
	}
	item, err := service.UpdateItem(uint(id), req, c.GetString("staff_username"))
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "Item berhasil diupdate", item)
}

// ── Admin: Orders ─────────────────────────────────────────────

func AdminGetOrders(c *gin.Context) {
	storeID    := c.Query("store_id")
	status     := c.Query("status")
	page, _    := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	orders, total, _ := repository.FindAllOrders(storeID, status, page, perPage)

	totalPage := int(total) / perPage
	if int(total)%perPage != 0 {
		totalPage++
	}
	utils.ResponseSuccessPaginate(c, http.StatusOK, "OK", orders, utils.Meta{
		Page: page, PerPage: perPage, Total: total, TotalPage: totalPage,
	})
}

func AdminUpdateOrderStatus(c *gin.Context) {
	var req dto.UpdateOrderStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseValidationError(c, http.StatusBadRequest, "Validasi gagal", err.Error())
		return
	}
	order, err := service.UpdateOrderStatus(c.Param("id"), req.Status)
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "Status order diupdate", order)
}

// ── Admin: Moka Sync ──────────────────────────────────────────

func AdminSyncMoka(c *gin.Context) {
	catCount, itemCount, err := service.SyncFromMoka()
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "Sync dari Moka berhasil", gin.H{
		"categories_synced": catCount,
		"items_synced":      itemCount,
	})
}

// ── Customer ──────────────────────────────────────────────────

// CustomerCreateOrder membuat pesanan FnB selama sesi bermain aktif.
func CustomerCreateOrder(c *gin.Context) {
	var req dto.CreateFnbOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseValidationError(c, http.StatusBadRequest, "Validasi gagal", err.Error())
		return
	}
	order, err := service.CreateFnbOrder(req, c.GetString("customer_id"))
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusCreated, "Pesanan FnB berhasil dikirim!", order)
}

// CustomerGetMyOrders mengambil semua order FnB milik customer yang login.
func CustomerGetMyOrders(c *gin.Context) {
	orders, _ := repository.FindOrdersByCustomer(c.GetString("customer_id"))
	utils.ResponseSuccess(c, http.StatusOK, "OK", orders)
}
