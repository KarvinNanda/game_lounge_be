package controller

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"game_lounge_be/config"
	"game_lounge_be/models"
	eventBookingDto "game_lounge_be/modules/event_booking/dto"
	eventBookingRepo "game_lounge_be/modules/event_booking/repository"
	eventBookingService "game_lounge_be/modules/event_booking/service"
	"game_lounge_be/utils"

	"github.com/gin-gonic/gin"
)

// CheckEventAvailability mengecek ketersediaan dan preview harga event.
// Reuse: eventBookingService.PreviewPrice + eventBookingRepo untuk blocked ranges.
func CheckEventAvailability(c *gin.Context) {
	storeID := c.Query("store_id")
	date := c.Query("date")
	startTime := c.Query("start_time")
	endTime := c.Query("end_time")

	if storeID == "" || date == "" {
		utils.ResponseError(c, http.StatusBadRequest, "store_id dan date wajib diisi")
		return
	}

	type BlockedRange struct {
		StartTime string `json:"start_time"`
		EndTime   string `json:"end_time"`
		Type      string `json:"type"`
	}
	var blockedRanges []BlockedRange

	// Event bookings yang sudah ada pada tanggal & store ini
	if blockedEvents, err := eventBookingRepo.FindForDashboard(storeID, date); err == nil {
		for _, e := range blockedEvents {
			blockedRanges = append(blockedRanges, BlockedRange{e.StartTime, e.EndTime, "event"})
		}
	}

	// Regular bookings yang blocked pada store + tanggal yang sama
	var regularBookings []models.Booking
	config.DB.Table("bookings b").
		Joins("JOIN store_rooms sr ON sr.id = b.room_id").
		Select("b.start_time, b.end_time").
		Where("sr.store_id = ? AND b.booking_date = ? AND b.status NOT IN ('cancelled')",
			storeID, date).
		Find(&regularBookings)
	for _, b := range regularBookings {
		blockedRanges = append(blockedRanges, BlockedRange{b.StartTime, b.EndTime, "booking"})
	}

	// Harga event — reuse service yang sudah ada
	priceInfo := map[string]interface{}{"price_per_day": 0.0}
	if startTime != "" && endTime != "" {
		if preview, err := eventBookingService.PreviewPrice(storeID, startTime, endTime); err == nil {
			priceInfo = preview
		}
	} else {
		if ep, err := eventBookingRepo.GetEventPrice(storeID); err == nil {
			priceInfo = map[string]interface{}{"price_per_day": ep.PricePerDay}
		}
	}

	utils.ResponseSuccess(c, http.StatusOK, "OK", gin.H{
		"blocked_ranges": blockedRanges,
		"event_price":    priceInfo,
	})
}

// InitiateEventBooking membuat event booking customer menggunakan existing service.
// Reuse: eventBookingService.Create() yang sudah handle overlap check + price calculation.
func InitiateEventBooking(c *gin.Context) {
	var req struct {
		StoreID       string `json:"store_id" binding:"required"`
		EventName     string `json:"event_name" binding:"required,min=2"`
		BookingDate   string `json:"booking_date" binding:"required"`
		StartTime     string `json:"start_time" binding:"required"`
		EndTime       string `json:"end_time" binding:"required"`
		PaymentMethod string `json:"payment_method" binding:"required"`
		Notes         string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseValidationError(c, http.StatusBadRequest, "Validasi gagal", err.Error())
		return
	}

	customerID := c.GetString("customer_id")

	loc, _ := time.LoadLocation("Asia/Jakarta")
	if req.BookingDate < time.Now().In(loc).Format("2006-01-02") {
		utils.ResponseError(c, http.StatusBadRequest, "Tanggal event sudah lewat")
		return
	}

	var customer models.Customer
	if err := config.DB.Where("id = ?", customerID).First(&customer).Error; err != nil {
		utils.ResponseError(c, http.StatusNotFound, "Data customer tidak ditemukan")
		return
	}

	customerEmail := ""
	if customer.Email != nil {
		customerEmail = *customer.Email
	}

	// Reuse DTO & service yang sudah ada — overlap check + price calculation sudah di dalamnya
	createReq := eventBookingDto.CreateEventBookingRequest{
		StoreID:          req.StoreID,
		EventName:        req.EventName,
		CustomerName:     customer.Name,
		CustomerWhatsapp: customer.Whatsapp,
		CustomerEmail:    customerEmail,
		BookingDate:      req.BookingDate,
		StartTime:        req.StartTime,
		EndTime:          req.EndTime,
		Notes:            req.Notes,
		DurationType:     "hourly",
		BookingScope:     "full_venue",
	}

	eventBooking, err := eventBookingService.Create(createReq, customerID)
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}

	// Buat Xendit invoice (prefix "EB-" untuk event booking)
	appURL := os.Getenv("APP_URL")
	externalID := fmt.Sprintf("EB-%s", eventBooking.ID)

	invoice, err := utils.CreateXenditInvoice(utils.XenditInvoiceRequest{
		ExternalID:      externalID,
		Amount:          eventBooking.TotalPrice,
		InvoiceDuration: int(eventBookingRepo.EventPaymentWindow.Seconds()),
		PayerEmail:      customerEmail,
		Description: fmt.Sprintf("Event Booking — %s, %s %s-%s",
			req.EventName, req.BookingDate, req.StartTime, req.EndTime),
		SuccessURL: fmt.Sprintf("%s/payment/success?type=event&id=%s", appURL, eventBooking.ID),
		FailureURL: fmt.Sprintf("%s/payment/failed?type=event", appURL),
	})
	if err != nil {
		// Rollback: batalkan event booking yang sudah dibuat
		cancelReq := eventBookingDto.CancelEventBookingRequest{Reason: "Gagal membuat invoice pembayaran"}
		eventBookingService.Cancel(eventBooking.ID, cancelReq, "system")
		utils.ResponseError(c, http.StatusInternalServerError, "Gagal membuat invoice pembayaran")
		return
	}

	invoiceURL := invoice.InvoiceURL
	if strings.Contains(invoiceURL, "/payment/mock") {
		invoiceURL = fmt.Sprintf("%s&intent_id=%s&type=event", invoiceURL, eventBooking.ID)
	}

	// Update event booking dengan info customer + xendit
	// Dilakukan SETELAH record ada di DB (Create sudah selesai di atas)
	paymentPending := "pending_payment"
	config.DB.Model(&models.EventBooking{}).
		Where("id = ?", eventBooking.ID).
		Updates(map[string]interface{}{
			"customer_id":         customerID,
			"is_customer_booking": true,
			"payment_status":      paymentPending,
			"payment_method":      req.PaymentMethod,
			"xendit_invoice_id":   invoice.ID,
			"xendit_invoice_url":  invoiceURL,
		})

	utils.ResponseSuccess(c, http.StatusCreated, "Event booking berhasil dibuat", gin.H{
		"event_booking_id": eventBooking.ID,
		"invoice_url":      invoiceURL,
		"total_price":      eventBooking.TotalPrice,
		"duration_hours":   eventBooking.DurationHours,
		"event_name":       req.EventName,
		"payment_status":   "pending_payment",
		// Batas bayar; setelah ini event dibatalkan otomatis (lihat EventPaymentWindow).
		"expires_at": eventBooking.CreatedAt.Add(eventBookingRepo.EventPaymentWindow).Format(time.RFC3339),
	})
}

// MockConfirmEventBooking untuk simulasi konfirmasi pembayaran event (testing saja).
func MockConfirmEventBooking(c *gin.Context) {
	if !utils.XenditMockMode() {
		utils.ResponseError(c, http.StatusForbidden, "Hanya tersedia di mode testing")
		return
	}

	eventID := c.Param("event_id")
	customerID := c.GetString("customer_id")

	pendingStatus := "pending_payment"
	var eb models.EventBooking
	if err := config.DB.
		Where("id = ? AND customer_id = ? AND payment_status = ?",
			eventID, customerID, pendingStatus).
		First(&eb).Error; err != nil {
		utils.ResponseError(c, http.StatusNotFound, "Event booking tidak ditemukan atau sudah diproses")
		return
	}

	if eb.XenditInvoiceID == nil {
		utils.ResponseError(c, http.StatusInternalServerError, "Invoice tidak ditemukan pada event ini")
		return
	}
	// Jalur yang sama dengan webhook Xendit.
	if err := eventBookingService.ConfirmCustomerPayment(*eb.XenditInvoiceID); err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.ResponseSuccess(c, http.StatusOK, "Event booking dikonfirmasi", gin.H{
		"event_name":   eb.EventName,
		"booking_date": eb.BookingDate,
		"start_time":   eb.StartTime,
		"end_time":     eb.EndTime,
		"total_price":  eb.TotalPrice,
	})
}
