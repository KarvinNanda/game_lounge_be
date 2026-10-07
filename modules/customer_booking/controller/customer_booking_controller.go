package controller

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"game_lounge_be/config"
	"game_lounge_be/models"
	customerAppCtrl "game_lounge_be/modules/customer_app/controller"
	"game_lounge_be/modules/customer_booking/repository"
	"game_lounge_be/modules/customer_booking/service"
	eventService "game_lounge_be/modules/event_booking/service"
	"game_lounge_be/utils"

	"github.com/gin-gonic/gin"
)

// GetAvailability mengambil slot tersedia untuk room type pada tanggal tertentu.
// Public endpoint — tidak memerlukan autentikasi.
// Query params: store_id, room_template_id, date (YYYY-MM-DD), duration_hours
// GetQuote — GET /public/booking/quote?store_id&room_template_id&booking_date&selected_slots[]=10:00
// Harga final untuk slot yang dipilih, tanpa membuat hold. Logika sama dengan /bookings/initiate.
func GetQuote(c *gin.Context) {
	storeID := c.Query("store_id")
	roomTemplateID, _ := strconv.Atoi(c.Query("room_template_id"))
	date := c.Query("booking_date")
	slots := c.QueryArray("selected_slots[]")
	if len(slots) == 0 {
		slots = c.QueryArray("selected_slots")
	}
	if storeID == "" || roomTemplateID <= 0 || date == "" || len(slots) == 0 {
		utils.ResponseError(c, http.StatusBadRequest, "store_id, room_template_id, booking_date, dan selected_slots[] wajib diisi")
		return
	}
	if len(slots) > 24 {
		utils.ResponseError(c, http.StatusBadRequest, "maksimal 24 slot")
		return
	}

	result, err := service.QuoteBooking(storeID, uint(roomTemplateID), date, slots)
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "OK", result)
}

func GetAvailability(c *gin.Context) {
	storeID            := c.Query("store_id")
	roomTemplateID, _  := strconv.Atoi(c.Query("room_template_id"))
	date               := c.Query("date")
	duration, _        := strconv.ParseFloat(c.Query("duration_hours"), 64)

	if storeID == "" || roomTemplateID == 0 || date == "" || duration == 0 {
		utils.ResponseError(c, http.StatusBadRequest, "Parameter store_id, room_template_id, date, duration_hours wajib diisi")
		return
	}

	result, err := service.GetAvailability(storeID, uint(roomTemplateID), date, duration)
	if err != nil {
		utils.ResponseError(c, http.StatusInternalServerError, "Gagal mengambil ketersediaan")
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "OK", result)
}

// InitiateBooking membuat hold + Xendit invoice.
// Memerlukan customer JWT.
// Response berisi invoice_url untuk redirect ke halaman pembayaran.
func InitiateBooking(c *gin.Context) {
	var req service.InitiateBookingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseValidationError(c, http.StatusBadRequest, "Validasi gagal", err.Error())
		return
	}

	customerID := c.GetString("customer_id")
	result, err := service.InitiateBooking(req, customerID)
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusCreated, "Invoice berhasil dibuat", result)
}

// XenditWebhook menerima callback dari Xendit setelah pembayaran.
// Tanpa auth — diverifikasi via X-CALLBACK-TOKEN header.
// Routing berdasarkan prefix external_id:
//   - "PC-…" → play credits purchase
//   - lainnya → booking
func XenditWebhook(c *gin.Context) {
	callbackToken := c.GetHeader("x-callback-token")
	if !utils.VerifyXenditWebhook(callbackToken) {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
		return
	}

	var payload struct {
		ID            string `json:"id"`             // Xendit invoice ID
		ExternalID    string `json:"external_id"`    // prefix PC- = credits, lainnya = booking
		Status        string `json:"status"`         // PAID / EXPIRED / FAILED
		PaymentMethod string `json:"payment_method"` // QRIS, BCA, dll
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid payload"})
		return
	}

	// Hanya proses event PAID; abaikan yang lain
	if payload.Status != "PAID" {
		c.JSON(http.StatusOK, gin.H{"message": "ignored"})
		return
	}

	if strings.HasPrefix(payload.ExternalID, "PC-") {
		// ── Play credits purchase ───────────────────────────────
		var intent models.PlayCreditsPurchaseIntent
		config.DB.Where("xendit_invoice_id = ?", payload.ID).First(&intent)
		if intent.ID != "" {
			customerAppCtrl.ConfirmPlayCreditsPaymentFromWebhook(intent.ID)
		}
		c.JSON(http.StatusOK, gin.H{"message": "credits confirmed"})
	} else if strings.HasPrefix(payload.ExternalID, "EB-") {
		// ── Event booking ─────────────────────────────────────
		// Dicari lewat xendit_invoice_id (payload.ID), bukan external_id.
		err := eventService.ConfirmCustomerPayment(payload.ID)
		switch {
		case err == nil:
			c.JSON(http.StatusOK, gin.H{"message": "event booking confirmed"})
		case errors.Is(err, eventService.ErrEventPaymentNotFound),
			errors.Is(err, eventService.ErrEventPaymentProcessed),
			errors.Is(err, eventService.ErrEventPaidButSlotTaken):
			c.JSON(http.StatusOK, gin.H{"message": "ignored"})
		default:
			log.Printf("[WEBHOOK] gagal konfirmasi event invoice %s: %v", payload.ID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "retry"})
		}
	} else {
		// ── Room booking ───────────────────────────────────────
		err := service.ConfirmBookingFromWebhook(payload.ID, payload.PaymentMethod)
		switch {
		case err == nil:
			c.JSON(http.StatusOK, gin.H{"message": "booking confirmed"})
		case errors.Is(err, service.ErrHoldNotFound), errors.Is(err, service.ErrPaidButSlotTaken):
			// Sudah diproses / perlu tindakan manual — retry tidak akan membantu.
			c.JSON(http.StatusOK, gin.H{"message": "ignored"})
		default:
			// Error sementara (DB dll). Handler idempotent → aman minta Xendit retry.
			log.Printf("[WEBHOOK] gagal konfirmasi invoice %s: %v", payload.ID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "retry"})
		}
	}
}

// GetMyBookings mengambil list booking milik customer yang sedang login.
// Query: page, per_page (maks 50), status (opsional, dipisah koma: upcoming,ongoing).
func GetMyBookings(c *gin.Context) {
	page, _    := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "10"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 50 {
		perPage = 10
	}

	statuses, err := parseStatusFilter(c.Query("status"))
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}

	bookings, total, err := service.GetCustomerBookings(c.GetString("customer_id"), statuses, page, perPage)
	if err != nil {
		utils.ResponseError(c, http.StatusInternalServerError, "Gagal mengambil data booking")
		return
	}

	totalPage := int(total) / perPage
	if int(total)%perPage != 0 {
		totalPage++
	}

	utils.ResponseSuccessPaginate(c, http.StatusOK, "OK", bookings, utils.Meta{
		Page:      page,
		PerPage:   perPage,
		Total:     total,
		TotalPage: totalPage,
	})
}

// GetMyBookingByID mengambil detail satu booking milik customer yang sedang login.
func GetMyBookingByID(c *gin.Context) {
	b, err := service.GetCustomerBookingByID(c.Param("id"), c.GetString("customer_id"))
	if err != nil {
		utils.ResponseError(c, http.StatusNotFound, err.Error())
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "OK", b)
}

// MockConfirm digunakan untuk simulasi pembayaran berhasil tanpa Xendit.
// HANYA aktif jika XENDIT_SECRET_KEY kosong (mock mode).
// Akan langsung membuat booking dari hold yang ada.
func MockConfirm(c *gin.Context) {
	// Blokir di production (jika Xendit key sudah diisi)
	if !utils.XenditMockMode() {
		utils.ResponseError(c, http.StatusForbidden, "Endpoint ini hanya tersedia di mode testing")
		return
	}

	holdID     := c.Param("hold_id")
	customerID := c.GetString("customer_id")

	// Cari hold milik customer ini
	hold, err := repository.FindHoldByID(holdID)
	if err != nil {
		utils.ResponseError(c, http.StatusNotFound, "Hold tidak ditemukan")
		return
	}

	// Pastikan hold milik customer yang sedang login
	if hold.CustomerID != customerID {
		utils.ResponseError(c, http.StatusForbidden, "Akses ditolak")
		return
	}

	// Cek hold belum expire
	if hold.IsExpired() {
		utils.ResponseError(c, http.StatusBadRequest, "Waktu pembayaran sudah habis (hold expired). Silakan booking ulang.")
		return
	}

	// XenditInvoiceID wajib ada (di-set saat hold dibuat)
	if hold.XenditInvoiceID == nil {
		utils.ResponseError(c, http.StatusInternalServerError, "Invoice ID tidak ditemukan pada hold ini")
		return
	}

	// Konfirmasi booking langsung (sama seperti yang dilakukan webhook Xendit)
	if err := service.ConfirmBookingFromWebhook(*hold.XenditInvoiceID, "mock_payment"); err != nil {
		utils.ResponseError(c, http.StatusInternalServerError, err.Error())
		return
	}

	// Ambil booking yang baru dibuat untuk return booking code
	var booking models.Booking
	config.DB.Where("customer_id = ?", customerID).
		Order("created_at DESC").
		First(&booking)

	utils.ResponseSuccess(c, http.StatusOK, "Pembayaran berhasil dikonfirmasi", gin.H{
		"booking_code": booking.BookingCode,
		"booking_id":   booking.ID,
	})
}

// parseStatusFilter membaca ?status=upcoming,ongoing. Kosong = semua status.
func parseStatusFilter(raw string) ([]string, error) {
	var statuses []string
	for _, st := range strings.Split(raw, ",") {
		st = strings.TrimSpace(st)
		if st == "" {
			continue
		}
		switch st {
		case "upcoming", "ongoing", "completed", "cancelled":
			statuses = append(statuses, st)
		default:
			return nil, errors.New("status tidak valid (upcoming, ongoing, completed, cancelled)")
		}
	}
	return statuses, nil
}

// GetMyBookingByHold — GET /customer/bookings/by-hold/:hold_id
// Dipakai halaman payment success (redirect Xendit membawa hold_id) untuk polling
// status: pending → confirmed (+ booking_code) atau expired.
// Hold milik customer lain dijawab 404 yang sama dengan ID tidak dikenal,
// supaya endpoint ini tidak bisa dipakai mengecek keberadaan sebuah hold.
func GetMyBookingByHold(c *gin.Context) {
	status, err := service.GetHoldStatus(c.Param("hold_id"), c.GetString("customer_id"))
	if errors.Is(err, service.ErrHoldNotFound) {
		utils.ResponseError(c, http.StatusNotFound, "Pembayaran tidak ditemukan")
		return
	}
	if err != nil {
		log.Printf("[HOLD STATUS] hold=%s: %v", c.Param("hold_id"), err)
		utils.ResponseError(c, http.StatusInternalServerError, "Gagal mengambil status pembayaran")
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "OK", status)
}
