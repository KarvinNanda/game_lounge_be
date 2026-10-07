package controller

import (
	"net/http"
	"strconv"

	"game_lounge_be/config"
	"game_lounge_be/models"
	customerBookingRepo "game_lounge_be/modules/customer_booking/repository"
	pricingDto "game_lounge_be/modules/pricing/dto"
	pricingService "game_lounge_be/modules/pricing/service"
	"game_lounge_be/utils"

	"github.com/gin-gonic/gin"
)

// RoomRecommendations mengambil rekomendasi ruangan untuk customer yang login.
// Prioritas: favorit customer → fallback 4 room template aktif pertama.
func RoomRecommendations(c *gin.Context) {
	customerID := c.GetString("customer_id")

	// Ambil room template ID favorit customer
	var favoriteIDs []uint
	config.DB.Model(&models.CustomerFavoriteRoomType{}).
		Where("customer_id = ?", customerID).
		Pluck("room_template_id", &favoriteIDs)

	var rooms []models.RoomTemplate
	if len(favoriteIDs) > 0 {
		config.DB.Where("id IN ? AND is_active = true AND deleted_at IS NULL", favoriteIDs).
			Find(&rooms)
	}
	// Fallback: jika tidak ada favorit atau tidak ada room ditemukan
	if len(rooms) == 0 {
		config.DB.Where("is_active = true AND deleted_at IS NULL").
			Order("id ASC").Limit(4).Find(&rooms)
	}

	// Tambahkan harga terendah paket 1 jam per room template
	type RoomWithPrice struct {
		models.RoomTemplate
		MinPrice float64 `json:"min_price"`
	}
	result := make([]RoomWithPrice, 0, len(rooms))
	for _, r := range rooms {
		var minPrice float64
		config.DB.Table("store_package_prices").
			Where("room_template_id = ? AND duration_hours = 1 AND deleted_at IS NULL", r.ID).
			Select("MIN(price)").Scan(&minPrice)
		result = append(result, RoomWithPrice{RoomTemplate: r, MinPrice: minPrice})
	}

	utils.ResponseSuccess(c, http.StatusOK, "OK", result)
}

// PublicGetStores mengambil list store aktif.
func PublicGetStores(c *gin.Context) {
	var stores []models.Store
	config.DB.Where("status = 'active' AND deleted_at IS NULL").Preload("OperatingHours").
		Order("name ASC").Find(&stores)
	utils.ResponseSuccess(c, http.StatusOK, "OK", stores)
}

// PublicGetStoreByID mengambil detail satu store aktif.
func PublicGetStoreByID(c *gin.Context) {
	var store models.Store
	if err := config.DB.
		Preload("OperatingHours").
		Preload("Rooms.RoomTemplate").
		Where("id = ? AND status = 'active' AND deleted_at IS NULL", c.Param("id")).
		First(&store).Error; err != nil {
		utils.ResponseError(c, http.StatusNotFound, "Store tidak ditemukan")
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "OK", store)
}

// PublicGetRoomTemplates mengambil room template yang tersedia di store tertentu.
// Jika store_id dikirim → filter hanya room template yang punya unit aktif di store itu.
// Jika store_id kosong → return semua room template aktif.
// Harga min_price diambil dari store yang dipilih (bukan global minimum).
func PublicGetRoomTemplates(c *gin.Context) {
	storeID := c.Query("store_id")

	var templates []models.RoomTemplate

	if storeID != "" {
		// Ambil room template yang punya minimal 1 unit aktif di store ini
		config.DB.Where(`
            is_active = true
            AND deleted_at IS NULL
            AND id IN (
                SELECT DISTINCT room_template_id
                FROM store_rooms
                WHERE store_id = ?
                AND is_active = true
                AND deleted_at IS NULL
            )`, storeID).
			Order("id ASC").
			Find(&templates)
	} else {
		// Tanpa filter store: return semua template aktif
		config.DB.Where("is_active = true AND deleted_at IS NULL").
			Order("id ASC").Find(&templates)
	}

	// Tambahkan min_price dari store yang dipilih (paket 1 jam)
	type TemplateWithPrice struct {
		models.RoomTemplate
		MinPrice float64 `json:"min_price"`
	}

	var result []TemplateWithPrice
	for _, t := range templates {
		var minPrice float64

		if storeID != "" {
			// Harga minimum dari store spesifik yang dipilih
			config.DB.Table("store_package_prices").
				Where("store_id = ? AND room_template_id = ? AND duration_hours = 1 AND deleted_at IS NULL",
					storeID, t.ID).
				Select("MIN(price)").Scan(&minPrice)
		} else {
			// Tanpa store: ambil harga minimum global
			config.DB.Table("store_package_prices").
				Where("room_template_id = ? AND duration_hours = 1 AND deleted_at IS NULL", t.ID).
				Select("MIN(price)").Scan(&minPrice)
		}

		result = append(result, TemplateWithPrice{RoomTemplate: t, MinPrice: minPrice})
	}

	utils.ResponseSuccess(c, http.StatusOK, "OK", result)
}

// PublicGetRoomTemplateByID mengambil detail 1 room template (tanpa auth).
// Digunakan di halaman detail ruangan customer web.
func PublicGetRoomTemplateByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, "ID tidak valid")
		return
	}

	var template models.RoomTemplate
	if err := config.DB.
		Preload("Facilities").
		Where("id = ? AND is_active = true AND deleted_at IS NULL", id).
		First(&template).Error; err != nil {
		utils.ResponseError(c, http.StatusNotFound, "Ruangan tidak ditemukan")
		return
	}

	// Hitung min_price global (paket 1 jam di semua store)
	var minPrice float64
	config.DB.Table("store_package_prices").
		Where("room_template_id = ? AND duration_hours = 1 AND deleted_at IS NULL", id).
		Select("MIN(price)").Scan(&minPrice)

	utils.ResponseSuccess(c, http.StatusOK, "OK", gin.H{
		"id":           template.ID,
		"name":         template.Name,
		"description":  template.Description,
		"image_url":    template.ImageURL,
		"capacity_min": template.CapacityMin,
		"capacity_max": template.CapacityMax,
		"facilities":   template.Facilities,
		"min_price":    minPrice,
	})
}

// GetBookingSlots mengambil semua slot per jam dalam jam operasional store
// beserta ketersediaan (berapa unit tersisa) dan harga per slot.
// Digunakan di customer booking page untuk tampilan multi-slot selection.
func GetBookingSlots(c *gin.Context) {
	storeID := c.Query("store_id")
	roomTemplateID, _ := strconv.Atoi(c.Query("room_template_id"))
	date := c.Query("date")

	if storeID == "" || roomTemplateID == 0 || date == "" {
		utils.ResponseError(c, http.StatusBadRequest,
			"store_id, room_template_id, dan date wajib diisi")
		return
	}

	// Satu sumber availability (booking, hold, event, lintas tengah malam) —
	// sama dengan yang dipakai saat hold dibuat, supaya slot yang tampil
	// tersedia memang bisa dibooking.
	daySlots, err := customerBookingRepo.GetAvailableSlots(storeID, uint(roomTemplateID), date, 1)
	if err != nil {
		utils.ResponseError(c, http.StatusInternalServerError, "Gagal memuat slot")
		return
	}
	openMins, closeMins := customerBookingRepo.OperatingWindow(storeID, date)

	var totalRooms int64
	config.DB.Model(&models.StoreRoom{}).
		Where("store_id = ? AND room_template_id = ? AND is_active = true AND deleted_at IS NULL", storeID, roomTemplateID).
		Count(&totalRooms)

	type SlotResult struct {
		StartTime string  `json:"start_time"`
		EndTime   string  `json:"end_time"`
		Available bool    `json:"available"`
		Price     float64 `json:"price"`
	}
	slots := make([]SlotResult, 0, len(daySlots))
	for _, sl := range daySlots {
		// Harga slot ini (1 jam) via pricing service
		slotPrice := 0.0
		if priceResult, err := pricingService.CalculatePrice(pricingDto.CalculatePriceRequest{
			StoreID:        storeID,
			RoomTemplateID: uint(roomTemplateID),
			BookingDate:    date,
			StartTime:      sl.StartTime,
			EndTime:        sl.EndTime,
		}); err == nil {
			slotPrice = priceResult.FinalPrice
		}
		slots = append(slots, SlotResult{StartTime: sl.StartTime, EndTime: sl.EndTime, Available: sl.Available, Price: slotPrice})
	}

	utils.ResponseSuccess(c, http.StatusOK, "OK", gin.H{
		"slots":           slots,
		"operating_hours": gin.H{"open": utils.MinsToClock(openMins), "close": utils.MinsToClock(closeMins)},
		"total_rooms":     totalRooms,
	})
}
