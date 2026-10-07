package controller

import (
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"game_lounge_be/config"
	"game_lounge_be/models"
	"game_lounge_be/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

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
            AND expires_at > ?
            AND expires_at <= ?`,
			// Waktu dari Go, bukan NOW() MySQL: zona waktu server DB bisa berbeda
			// dengan zona waktu yang dipakai app saat menulis expires_at.
			customerID, time.Now(), time.Now().AddDate(0, 0, 7)).
		Find(&credits)

	utils.ResponseSuccess(c, http.StatusOK, "OK", gin.H{
		"count":   len(credits),
		"credits": credits,
	})
}

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
		ExpiresAt:  time.Now().Add(creditIntentDuration),
	}
	if err := config.DB.Create(intent).Error; err != nil {
		utils.ResponseError(c, http.StatusInternalServerError, "Gagal membuat transaksi")
		return
	}

	// Buat Xendit invoice — prefix "PC-" untuk bedakan dari booking di webhook
	appURL := os.Getenv("APP_URL")
	externalID := fmt.Sprintf("PC-%s", intent.ID[:16])
	payerEmail := ""
	if customer.Email != nil {
		payerEmail = *customer.Email
	}

	invoice, err := utils.CreateXenditInvoice(utils.XenditInvoiceRequest{
		ExternalID:      externalID,
		Amount:          pkg.Price,
		InvoiceDuration: int(creditIntentDuration.Seconds()),
		PayerEmail:      payerEmail,
		Description:     fmt.Sprintf("Top Up Play Credits — %s (%.0f Jam)", pkg.Name, pkg.TotalHours),
		SuccessURL:      fmt.Sprintf("%s/credits/success?intent_id=%s", appURL, intent.ID),
		FailureURL:      fmt.Sprintf("%s/credits/failed?intent_id=%s", appURL, intent.ID),
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
		"expires_at":  intent.ExpiresAt.Format(time.RFC3339),
		"package": gin.H{
			"name":          pkg.Name,
			"total_hours":   pkg.TotalHours,
			"validity_days": pkg.ValidityDays,
		},
	})
}

// MockConfirmPlayCredits untuk simulasi pembayaran credits berhasil (mock mode saja).
func MockConfirmPlayCredits(c *gin.Context) {
	if !utils.XenditMockMode() {
		utils.ResponseError(c, http.StatusForbidden, "Endpoint ini hanya tersedia di mode testing")
		return
	}

	intentID := c.Param("intent_id")
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

	if err := confirmPlayCreditsPayment(intent.ID); err != nil {
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
// creditIntentDuration = masa berlaku intent pembelian credits; invoice Xendit
// diberi durasi yang sama supaya tidak bisa dibayar setelah intent habis.
const creditIntentDuration = 30 * time.Minute

func ConfirmPlayCreditsPaymentFromWebhook(intentID string) {
	if err := confirmPlayCreditsPayment(intentID); err != nil {
		log.Printf("[WEBHOOK] play credits intent %s: %v", intentID, err)
	}
}

// confirmPlayCreditsPayment membuat CustomerPlayCredit setelah pembayaran berhasil.
// Dipanggil dari webhook Xendit atau mock confirm.
// confirmPlayCreditsPayment selalu mencatat payment_method "xendit" (enum: manual|xendit).
func confirmPlayCreditsPayment(intentID string) error {
	var intent models.PlayCreditsPurchaseIntent
	if err := config.DB.Preload("Package").Preload("Customer").Preload("Store").
		Where("id = ? AND status = 'pending'", intentID).
		First(&intent).Error; err != nil {
		return errors.New("intent tidak ditemukan atau sudah diproses")
	}

	now := time.Now()
	amount := intent.Amount
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

	// Idempotent: intent "diklaim" (pending → paid) di transaction yang sama dengan
	// pembuatan credit. Webhook ganda/bersamaan mendapat 0 baris → tidak ada credit kedua.
	errAlreadyProcessed := errors.New("intent sudah diproses")
	err := config.DB.Transaction(func(tx *gorm.DB) error {
		claim := tx.Model(&models.PlayCreditsPurchaseIntent{}).
			Where("id = ? AND status = 'pending'", intent.ID).
			Updates(map[string]interface{}{"status": "paid", "paid_at": now})
		if claim.Error != nil {
			return claim.Error
		}
		if claim.RowsAffected != 1 {
			return errAlreadyProcessed
		}
		return tx.Create(credit).Error
	})
	if errors.Is(err, errAlreadyProcessed) {
		return err
	}
	if err != nil {
		return errors.New("gagal membuat record play credits")
	}

	// Kirim notifikasi email async
	utils.SafeGo(func() { sendPlayCreditsConfirmationEmail(intent.Customer, intent.Package, intent.Store, credit) })

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

// GetMyCredits mengambil semua play credits aktif milik customer, diurutkan FIFO.
func GetMyCredits(c *gin.Context) {
	customerID := c.GetString("customer_id")

	var credits []models.CustomerPlayCredit
	config.DB.Preload("Package").
		Where("customer_id = ? AND is_active = true AND expires_at > ? AND deleted_at IS NULL",
			customerID, time.Now()).
		Order("expires_at ASC").
		Find(&credits)

	type CreditWithUsage struct {
		models.CustomerPlayCredit
		UsedHours   float64 `json:"used_hours"`
		UsedPercent float64 `json:"used_percent"`
	}

	var result []CreditWithUsage
	for _, cr := range credits {
		used := cr.TotalHours - cr.RemainingHours
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
