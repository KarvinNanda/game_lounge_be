package controller

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"game_lounge_be/config"
	"game_lounge_be/models"
	eventBookingDto     "game_lounge_be/modules/event_booking/dto"
	eventBookingRepo    "game_lounge_be/modules/event_booking/repository"
	eventBookingService "game_lounge_be/modules/event_booking/service"
	"game_lounge_be/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Login memproses login customer (cek tabel customers, bukan staffs).
func Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseValidationError(c, http.StatusBadRequest, "Validasi gagal", err.Error())
		return
	}

	var customer models.Customer
	if err := config.DB.Where(
		"email = ? AND deleted_at IS NULL AND status = 'active'",
		req.Email,
	).First(&customer).Error; err != nil {
		utils.ResponseError(c, http.StatusUnauthorized, "Email atau password salah")
		return
	}

	// PasswordHash adalah *string — cek nil sebelum dereference
	if customer.PasswordHash == nil || !utils.CheckPassword(req.Password, *customer.PasswordHash) {
		utils.ResponseError(c, http.StatusUnauthorized, "Email atau password salah")
		return
	}

	token, err := utils.GenerateCustomerJWT(customer.ID, customer.Type)
	if err != nil {
		utils.ResponseError(c, http.StatusInternalServerError, "Gagal generate token")
		return
	}

	utils.ResponseSuccess(c, http.StatusOK, "Login berhasil", gin.H{
		"token":    token,
		"customer": customer,
	})
}

// Me mengambil profil customer yang sedang login.
func Me(c *gin.Context) {
	customerID := c.GetString("customer_id")
	var customer models.Customer
	if err := config.DB.
		Preload("FavoriteRoomTypes.RoomTemplate").
		Where("id = ? AND deleted_at IS NULL", customerID).
		First(&customer).Error; err != nil {
		utils.ResponseError(c, http.StatusNotFound, "Customer tidak ditemukan")
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "OK", customer)
}

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

// ── Public endpoints (tanpa auth) ─────────────────────────────────────────────

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

// ── Logout ───────────────────────────────────────────────────────────────────

// Logout cukup return 200 — token di-clear dari sisi frontend.
// JWT bersifat stateless sehingga tidak perlu invalidasi di server.
func Logout(c *gin.Context) {
	utils.ResponseSuccess(c, http.StatusOK, "Logout berhasil", nil)
}

// ── Update Profile ────────────────────────────────────────────────────────────

// UpdateProfile memperbarui nama dan nomor WhatsApp customer.
// Email tidak bisa diubah karena dipakai sebagai identifier login.
func UpdateProfile(c *gin.Context) {
	customerID := c.GetString("customer_id")

	var req struct {
		Name     string `json:"name" binding:"required,min=2,max=150"`
		Whatsapp string `json:"whatsapp" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseValidationError(c, http.StatusBadRequest, "Validasi gagal", err.Error())
		return
	}

	if err := config.DB.Model(&models.Customer{}).
		Where("id = ?", customerID).
		Updates(map[string]interface{}{
			"name":     req.Name,
			"whatsapp": req.Whatsapp,
		}).Error; err != nil {
		utils.ResponseError(c, http.StatusInternalServerError, "Gagal menyimpan profil")
		return
	}

	var customer models.Customer
	config.DB.Where("id = ?", customerID).First(&customer)
	utils.ResponseSuccess(c, http.StatusOK, "Profil berhasil diperbarui", customer)
}

// ── Change Password ───────────────────────────────────────────────────────────

// ChangePassword memperbarui password customer dengan validasi password lama.
func ChangePassword(c *gin.Context) {
	customerID := c.GetString("customer_id")

	var req struct {
		OldPassword     string `json:"old_password" binding:"required"`
		NewPassword     string `json:"new_password" binding:"required,min=8"`
		ConfirmPassword string `json:"confirm_password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseValidationError(c, http.StatusBadRequest, "Validasi gagal", err.Error())
		return
	}
	if req.NewPassword != req.ConfirmPassword {
		utils.ResponseError(c, http.StatusBadRequest, "Konfirmasi password tidak cocok")
		return
	}

	var customer models.Customer
	if err := config.DB.Where("id = ?", customerID).First(&customer).Error; err != nil {
		utils.ResponseError(c, http.StatusNotFound, "Customer tidak ditemukan")
		return
	}

	if customer.PasswordHash == nil || !utils.CheckPassword(req.OldPassword, *customer.PasswordHash) {
		utils.ResponseError(c, http.StatusUnauthorized, "Password lama tidak sesuai")
		return
	}

	newHash, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		utils.ResponseError(c, http.StatusInternalServerError, "Gagal memproses password")
		return
	}

	if err := config.DB.Model(&models.Customer{}).
		Where("id = ?", customerID).
		Update("password_hash", newHash).Error; err != nil {
		utils.ResponseError(c, http.StatusInternalServerError, "Gagal menyimpan password baru")
		return
	}

	utils.ResponseSuccess(c, http.StatusOK, "Password berhasil diubah", nil)
}

// ── Credits Expiring ──────────────────────────────────────────────────────────

// CreditsExpiring mengambil play credits customer yang akan expired dalam 7 hari.
// Dipakai FE untuk menampilkan badge/count di notification bell.
func CreditsExpiring(c *gin.Context) {
	customerID := c.GetString("customer_id")

	var credits []models.CustomerPlayCredit
	config.DB.Preload("Package").
		Where(`customer_id = ?
            AND is_active = true
            AND deleted_at IS NULL
            AND remaining_hours > 0
            AND expires_at > NOW()
            AND expires_at <= DATE_ADD(NOW(), INTERVAL 7 DAY)`,
			customerID).
		Find(&credits)

	utils.ResponseSuccess(c, http.StatusOK, "OK", gin.H{
		"count":   len(credits),
		"credits": credits,
	})
}

// ── Public Room Templates ─────────────────────────────────────────────────────

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

// ── Forgot / Reset Password ───────────────────────────────────────────────────

// ForgotPassword menerima email customer dan memproses reset password secara async.
// SELALU return response yang sama — penyerang tidak tahu apakah email valid.
func ForgotPassword(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Jika email terdaftar, link reset password akan dikirim.",
		})
		return
	}

	ip := c.ClientIP()
	if forwarded := c.GetHeader("X-Forwarded-For"); forwarded != "" {
		ip = strings.Split(forwarded, ",")[0]
	}

	go processCustomerPasswordReset(req.Email, ip)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Jika email terdaftar, link reset password akan dikirim.",
	})
}

// processCustomerPasswordReset adalah logic internal — dijalankan secara async.
func processCustomerPasswordReset(email, ip string) {
	// 1. Rate limiting: max 3 request per jam per IP
	var requestCount int64
	config.DB.Model(&models.CustomerPasswordReset{}).
		Where("ip_address = ? AND created_at > ?", ip, time.Now().Add(-1*time.Hour)).
		Count(&requestCount)
	if requestCount >= 3 {
		return
	}

	// 2. Cari customer dengan email ini (aktif)
	var customer models.Customer
	if err := config.DB.Where(
		"email = ? AND status = 'active' AND deleted_at IS NULL", email,
	).First(&customer).Error; err != nil {
		return
	}

	// 3. Generate token 32-byte random hex
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return
	}
	token := hex.EncodeToString(tokenBytes)

	// 4. Simpan token ke DB
	reset := &models.CustomerPasswordReset{
		CustomerID: customer.ID,
		Token:      token,
		IPAddress:  ip,
		ExpiresAt:  time.Now().Add(15 * time.Minute),
	}
	if err := config.DB.Create(reset).Error; err != nil {
		return
	}

	// 5. Kirim email reset password
	if customer.Email == nil {
		return
	}
	appURL    := os.Getenv("APP_URL")
	resetLink := fmt.Sprintf("%s/reset-password/%s", appURL, token)
	subject   := "Reset Password — Quantum Game Station"
	body := fmt.Sprintf(`Halo %s,

Kami menerima permintaan reset password untuk akun kamu di Quantum Game Station.

Klik link berikut untuk membuat password baru:
%s

Link ini hanya berlaku selama 15 menit dan hanya bisa digunakan 1 kali.

Jika kamu tidak meminta reset password, abaikan email ini.
Password kamu tidak akan berubah.

Salam,
Tim Quantum Game Station`, customer.Name, resetLink)

	utils.SendEmail(*customer.Email, customer.Name, subject, body)
}

// ValidateResetToken memvalidasi token sebelum FE menampilkan form reset password.
func ValidateResetToken(c *gin.Context) {
	token := c.Param("token")
	var reset models.CustomerPasswordReset
	if err := config.DB.
		Where("token = ? AND expires_at > ? AND used_at IS NULL", token, time.Now()).
		First(&reset).Error; err != nil {
		utils.ResponseError(c, http.StatusBadRequest, "Link reset password tidak valid atau sudah kadaluwarsa")
		return
	}
	utils.ResponseSuccess(c, http.StatusOK, "Token valid", nil)
}

// ResetPassword memperbarui password customer menggunakan token yang valid.
func ResetPassword(c *gin.Context) {
	token := c.Param("token")
	var req struct {
		NewPassword     string `json:"new_password" binding:"required,min=8"`
		ConfirmPassword string `json:"confirm_password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseValidationError(c, http.StatusBadRequest, "Validasi gagal", err.Error())
		return
	}
	if req.NewPassword != req.ConfirmPassword {
		utils.ResponseError(c, http.StatusBadRequest, "Konfirmasi password tidak cocok")
		return
	}

	// Cari token valid
	var reset models.CustomerPasswordReset
	if err := config.DB.Preload("Customer").
		Where("token = ? AND expires_at > ? AND used_at IS NULL", token, time.Now()).
		First(&reset).Error; err != nil {
		utils.ResponseError(c, http.StatusBadRequest, "Link reset password tidak valid atau sudah kadaluwarsa")
		return
	}

	// Hash password baru
	hashedPassword, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		utils.ResponseError(c, http.StatusInternalServerError, "Gagal memproses password")
		return
	}

	// Update password di DB
	if err := config.DB.Model(&models.Customer{}).
		Where("id = ?", reset.CustomerID).
		Update("password_hash", hashedPassword).Error; err != nil {
		utils.ResponseError(c, http.StatusInternalServerError, "Gagal menyimpan password baru")
		return
	}

	// Tandai token sudah dipakai (one-time use)
	now := time.Now()
	config.DB.Model(&reset).Update("used_at", now)

	utils.ResponseSuccess(c, http.StatusOK, "Password berhasil diperbarui. Silakan login.", nil)
}

// ── Play Credits — Public & Purchase ─────────────────────────────────────────

// PublicGetPlayCreditsPackages mengambil paket credits aktif yang tersedia di store tertentu.
// Package tersedia di store jika: apply_to_all_stores = true ATAU ada di play_credits_package_stores.
func PublicGetPlayCreditsPackages(c *gin.Context) {
	storeID := c.Query("store_id")
	if storeID == "" {
		utils.ResponseError(c, http.StatusBadRequest, "store_id wajib diisi")
		return
	}

	var packages []models.PlayCreditsPackage
	config.DB.
		Where(`is_active = true AND deleted_at IS NULL AND (
            apply_to_all_stores = true
            OR id IN (
                SELECT package_id FROM play_credits_package_stores
                WHERE store_id = ?
            )
        )`, storeID).
		Order("price ASC").
		Find(&packages)

	utils.ResponseSuccess(c, http.StatusOK, "OK", packages)
}

// InitiatePlayCreditsPurchase membuat purchase intent dan Xendit invoice.
func InitiatePlayCreditsPurchase(c *gin.Context) {
	var req struct {
		PackageID     string `json:"package_id" binding:"required"`
		StoreID       string `json:"store_id" binding:"required"`
		PaymentMethod string `json:"payment_method" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseValidationError(c, http.StatusBadRequest, "Validasi gagal", err.Error())
		return
	}

	customerID := c.GetString("customer_id")

	// Ambil data package
	var pkg models.PlayCreditsPackage
	if err := config.DB.Where("id = ? AND is_active = true AND deleted_at IS NULL", req.PackageID).
		First(&pkg).Error; err != nil {
		utils.ResponseError(c, http.StatusNotFound, "Paket tidak ditemukan")
		return
	}

	// Validasi package tersedia di store ini
	available := false
	if pkg.ApplyToAllStores {
		available = true
	} else {
		var count int64
		config.DB.Model(&models.PlayCreditsPackageStore{}).
			Where("package_id = ? AND store_id = ?", req.PackageID, req.StoreID).
			Count(&count)
		available = count > 0
	}
	if !available {
		utils.ResponseError(c, http.StatusBadRequest, "Paket ini tidak tersedia di cabang yang dipilih")
		return
	}

	// Ambil data customer
	var customer models.Customer
	config.DB.Where("id = ?", customerID).First(&customer)

	// Buat purchase intent
	intent := &models.PlayCreditsPurchaseIntent{
		ID:         uuid.NewString(),
		CustomerID: customerID,
		StoreID:    req.StoreID,
		PackageID:  req.PackageID,
		Amount:     pkg.Price,
		ExpiresAt:  time.Now().Add(30 * time.Minute),
	}
	if err := config.DB.Create(intent).Error; err != nil {
		utils.ResponseError(c, http.StatusInternalServerError, "Gagal membuat transaksi")
		return
	}

	// Buat Xendit invoice — prefix "PC-" untuk bedakan dari booking di webhook
	appURL     := os.Getenv("APP_URL")
	externalID := fmt.Sprintf("PC-%s", intent.ID[:16])
	payerEmail := ""
	if customer.Email != nil {
		payerEmail = *customer.Email
	}

	invoice, err := utils.CreateXenditInvoice(utils.XenditInvoiceRequest{
		ExternalID:  externalID,
		Amount:      pkg.Price,
		PayerEmail:  payerEmail,
		Description: fmt.Sprintf("Top Up Play Credits — %s (%.0f Jam)", pkg.Name, pkg.TotalHours),
		SuccessURL:  fmt.Sprintf("%s/credits/success?intent_id=%s", appURL, intent.ID),
		FailureURL:  fmt.Sprintf("%s/credits/failed?intent_id=%s", appURL, intent.ID),
	})
	if err != nil {
		config.DB.Delete(&intent)
		utils.ResponseError(c, http.StatusInternalServerError, "Gagal membuat invoice pembayaran")
		return
	}

	// Tambahkan intent_id ke mock URL jika mock mode
	invoiceURL := invoice.InvoiceURL
	if strings.Contains(invoiceURL, "/payment/mock") {
		invoiceURL = fmt.Sprintf("%s&intent_id=%s&type=credits", invoiceURL, intent.ID)
	}

	// Update intent dengan invoice info
	config.DB.Model(intent).Updates(map[string]interface{}{
		"xendit_invoice_id":  invoice.ID,
		"xendit_invoice_url": invoiceURL,
	})

	utils.ResponseSuccess(c, http.StatusCreated, "Invoice berhasil dibuat", gin.H{
		"intent_id":   intent.ID,
		"invoice_url": invoiceURL,
		"amount":      pkg.Price,
		"package": gin.H{
			"name":          pkg.Name,
			"total_hours":   pkg.TotalHours,
			"validity_days": pkg.ValidityDays,
		},
	})
}

// MockConfirmPlayCredits untuk simulasi pembayaran credits berhasil (mock mode saja).
func MockConfirmPlayCredits(c *gin.Context) {
	if os.Getenv("XENDIT_SECRET_KEY") != "" {
		utils.ResponseError(c, http.StatusForbidden, "Endpoint ini hanya tersedia di mode testing")
		return
	}

	intentID   := c.Param("intent_id")
	customerID := c.GetString("customer_id")

	var intent models.PlayCreditsPurchaseIntent
	if err := config.DB.Preload("Package").Preload("Store").
		Where("id = ? AND customer_id = ? AND status = 'pending'", intentID, customerID).
		First(&intent).Error; err != nil {
		utils.ResponseError(c, http.StatusNotFound, "Transaksi tidak ditemukan")
		return
	}

	if time.Now().After(intent.ExpiresAt) {
		utils.ResponseError(c, http.StatusBadRequest, "Waktu pembayaran sudah habis")
		return
	}

	if err := confirmPlayCreditsPayment(intent.ID, "mock_payment"); err != nil {
		utils.ResponseError(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.ResponseSuccess(c, http.StatusOK, "Pembelian credits berhasil", gin.H{
		"package_name":  intent.Package.Name,
		"total_hours":   intent.Package.TotalHours,
		"validity_days": intent.Package.ValidityDays,
		"store_name":    intent.Store.Name,
	})
}

// ConfirmPlayCreditsPaymentFromWebhook dipanggil dari webhook handler Xendit.
func ConfirmPlayCreditsPaymentFromWebhook(intentID, paymentMethod string) {
	confirmPlayCreditsPayment(intentID, paymentMethod) //nolint:errcheck
}

// confirmPlayCreditsPayment membuat CustomerPlayCredit setelah pembayaran berhasil.
// Dipanggil dari webhook Xendit atau mock confirm.
func confirmPlayCreditsPayment(intentID, _ string) error {
	var intent models.PlayCreditsPurchaseIntent
	if err := config.DB.Preload("Package").Preload("Customer").Preload("Store").
		Where("id = ? AND status = 'pending'", intentID).
		First(&intent).Error; err != nil {
		return errors.New("intent tidak ditemukan atau sudah diproses")
	}

	now    := time.Now()
	amount := intent.Amount

	// Buat customer play credits
	credit := &models.CustomerPlayCredit{
		ID:             uuid.NewString(),
		CustomerID:     intent.CustomerID,
		PackageID:      intent.PackageID,
		TotalHours:     intent.Package.TotalHours,
		RemainingHours: intent.Package.TotalHours,
		ExpiresAt:      now.AddDate(0, 0, int(intent.Package.ValidityDays)),
		IsActive:       true,
		PaymentMethod:  "xendit",
		PaymentAmount:  &amount,
	}
	if err := config.DB.Create(credit).Error; err != nil {
		return errors.New("gagal membuat record play credits")
	}

	// Update intent status
	paidAt := now
	config.DB.Model(&intent).Updates(map[string]interface{}{
		"status":  "paid",
		"paid_at": paidAt,
	})

	// Kirim notifikasi email async
	go sendPlayCreditsConfirmationEmail(intent.Customer, intent.Package, intent.Store, credit)

	return nil
}

// sendPlayCreditsConfirmationEmail mengirim email konfirmasi setelah credits aktif.
func sendPlayCreditsConfirmationEmail(customer models.Customer, pkg models.PlayCreditsPackage, store models.Store, credit *models.CustomerPlayCredit) {
	if customer.Email == nil {
		return
	}
	subject := "Play Credits Kamu Sudah Aktif! — Quantum Game Station"
	body := fmt.Sprintf(`Halo %s,

Pembayaran Play Credits kamu telah berhasil dan paket sudah aktif!

Detail Pembelian:
━━━━━━━━━━━━━━━━━━━━━━━━━━
Paket         : %s
Total Jam     : %.0f Jam
Cabang        : %s
Berlaku Sampai: %s
━━━━━━━━━━━━━━━━━━━━━━━━━━

Cara menggunakan credits:
Pilih "Gunakan Play Credits" saat melakukan booking.

Selamat bermain!
Tim Quantum Game Station`,
		customer.Name,
		pkg.Name,
		pkg.TotalHours,
		store.Name,
		credit.ExpiresAt.Format("02 January 2006"),
	)
	utils.SendEmail(*customer.Email, customer.Name, subject, body)
}

// ── My Credits ────────────────────────────────────────────────────────────────

// GetMyCredits mengambil semua play credits aktif milik customer, diurutkan FIFO.
func GetMyCredits(c *gin.Context) {
	customerID := c.GetString("customer_id")

	var credits []models.CustomerPlayCredit
	config.DB.Preload("Package").
		Where("customer_id = ? AND is_active = true AND expires_at > NOW() AND deleted_at IS NULL",
			customerID).
		Order("expires_at ASC").
		Find(&credits)

	type CreditWithUsage struct {
		models.CustomerPlayCredit
		UsedHours   float64 `json:"used_hours"`
		UsedPercent float64 `json:"used_percent"`
	}

	var result []CreditWithUsage
	for _, cr := range credits {
		used    := cr.TotalHours - cr.RemainingHours
		percent := 0.0
		if cr.TotalHours > 0 {
			percent = (used / cr.TotalHours) * 100
		}
		result = append(result, CreditWithUsage{
			CustomerPlayCredit: cr,
			UsedHours:          used,
			UsedPercent:        math.Round(percent*10) / 10,
		})
	}

	utils.ResponseSuccess(c, http.StatusOK, "OK", result)
}

// ── Available Vouchers ────────────────────────────────────────────────────────

// GetMyVouchers mengambil voucher yang tersedia untuk customer.
// Logika: ambil semua voucher aktif & valid, lalu exclude yang sudah dipakai customer ini.
// Sama persis dengan query di admin booking page.
func GetMyVouchers(c *gin.Context) {
	customerID := c.GetString("customer_id")
	storeID    := c.Query("store_id") // opsional

	type VoucherResult struct {
		ID            string   `json:"voucher_id"`
		Code          string   `json:"code"`
		Name          string   `json:"name"`
		Description   *string  `json:"description"`
		DiscountType  string   `json:"discount_type"`
		DiscountValue float64  `json:"discount_value"`
		MaxDiscount   *float64 `json:"max_discount"`
		MinPurchase   float64  `json:"min_purchase"`
		EndDate       *string  `json:"end_date"`
		IsAllStores   bool     `json:"is_all_stores"`
	}

	query := config.DB.Table("vouchers v").
		Select(`v.id, v.code, v.name, v.description,
                v.discount_type, v.discount_value,
                v.max_discount, v.min_purchase,
                v.end_date, v.is_all_stores`).
		Where(`v.deleted_at IS NULL
            AND v.is_active = true
            AND v.start_date <= CURDATE()
            AND (v.end_date IS NULL OR v.end_date >= CURDATE())
            AND (v.type = 'booking' OR v.type = 'both')
            AND v.id NOT IN (
                SELECT vu.voucher_id FROM voucher_usages vu
                WHERE vu.customer_id = ?
            )`, customerID)

	if storeID != "" {
		query = query.Where(`(
            v.is_all_stores = true
            OR v.id IN (
                SELECT vs.voucher_id FROM voucher_stores vs
                WHERE vs.store_id = ?
            )
        )`, storeID)
	}

	query = query.Order("v.created_at DESC")

	var vouchers []VoucherResult
	query.Find(&vouchers)

	utils.ResponseSuccess(c, http.StatusOK, "OK", vouchers)
}

// ── Event Booking (Customer) ──────────────────────────────────────────────────

// CheckEventAvailability mengecek ketersediaan dan preview harga event.
// Reuse: eventBookingService.PreviewPrice + eventBookingRepo untuk blocked ranges.
func CheckEventAvailability(c *gin.Context) {
	storeID   := c.Query("store_id")
	date      := c.Query("date")
	startTime := c.Query("start_time")
	endTime   := c.Query("end_time")

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
	}

	eventBooking, err := eventBookingService.Create(createReq, customerID)
	if err != nil {
		utils.ResponseError(c, http.StatusBadRequest, err.Error())
		return
	}

	// Buat Xendit invoice (prefix "EB-" untuk event booking)
	appURL     := os.Getenv("APP_URL")
	externalID := fmt.Sprintf("EB-%s", eventBooking.ID[:16])

	invoice, err := utils.CreateXenditInvoice(utils.XenditInvoiceRequest{
		ExternalID:  externalID,
		Amount:      eventBooking.TotalPrice,
		PayerEmail:  customerEmail,
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
			"customer_id":          customerID,
			"is_customer_booking":  true,
			"payment_status":       paymentPending,
			"payment_method":       req.PaymentMethod,
			"xendit_invoice_id":    invoice.ID,
			"xendit_invoice_url":   invoiceURL,
		})

	utils.ResponseSuccess(c, http.StatusCreated, "Event booking berhasil dibuat", gin.H{
		"event_booking_id": eventBooking.ID,
		"invoice_url":      invoiceURL,
		"total_price":      eventBooking.TotalPrice,
		"duration_hours":   eventBooking.DurationHours,
		"event_name":       req.EventName,
	})
}

// MockConfirmEventBooking untuk simulasi konfirmasi pembayaran event (testing saja).
func MockConfirmEventBooking(c *gin.Context) {
	if os.Getenv("XENDIT_SECRET_KEY") != "" {
		utils.ResponseError(c, http.StatusForbidden, "Hanya tersedia di mode testing")
		return
	}

	eventID    := c.Param("event_id")
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

	paidStatus := "paid"
	config.DB.Model(&models.EventBooking{}).
		Where("id = ?", eventID).
		Updates(map[string]interface{}{
			"payment_status": paidStatus,
		})

	utils.ResponseSuccess(c, http.StatusOK, "Event booking dikonfirmasi", gin.H{
		"event_name":   eb.EventName,
		"booking_date": eb.BookingDate,
		"start_time":   eb.StartTime,
		"end_time":     eb.EndTime,
		"total_price":  eb.TotalPrice,
	})
}

// checkEventConflict mengecek apakah ada booking/event yang conflict dengan slot yang diminta.
func checkEventConflict(storeID, date, startTime, endTime, excludeID string) (bool, error) {
	var bCount int64
	config.DB.Table("bookings b").
		Joins("JOIN store_rooms sr ON sr.id = b.room_id").
		Where("sr.store_id = ? AND b.booking_date = ? AND b.status NOT IN ('cancelled') AND b.start_time < ? AND b.end_time > ?",
			storeID, date, endTime, startTime).
		Count(&bCount)
	if bCount > 0 {
		return true, nil
	}

	q := config.DB.Model(&models.EventBooking{}).
		Where("store_id = ? AND booking_date = ? AND status NOT IN ('cancelled') AND start_time < ? AND end_time > ?",
			storeID, date, endTime, startTime)
	if excludeID != "" {
		q = q.Where("id != ?", excludeID)
	}
	var eCount int64
	q.Count(&eCount)
	return eCount > 0, nil
}

// parseTimeMins mengkonversi "HH:MM" ke jumlah menit dari tengah malam.
func parseTimeMins(t string) int {
	if len(t) < 5 {
		return 0
	}
	return int(t[0]-'0')*10*60 + int(t[1]-'0')*60 + int(t[3]-'0')*10 + int(t[4]-'0')
}
