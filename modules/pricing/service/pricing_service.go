package service

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"game_lounge_be/models"
	"game_lounge_be/modules/pricing/dto"
	"game_lounge_be/modules/pricing/repository"

	"github.com/google/uuid"
)

// ── Pricing Config ────────────────────────────────────────────

// GetAllPricingsWithStores mengambil semua store beserta status pricing-nya.
func GetAllPricingsWithStores(filter dto.PricingListFilter) (interface{}, int64, error) {
	stores, total, err := repository.FindAllStores(filter.Search, filter.Page, filter.PerPage)
	if err != nil {
		return nil, 0, err
	}

	type StoreWithPricing struct {
		models.Store
		HasPricing    bool   `json:"has_pricing"`
		LastUpdatedBy string `json:"last_updated_by"`
		LastUpdatedAt string `json:"last_updated_at"`
	}

	var result []StoreWithPricing
	for _, s := range stores {
		item := StoreWithPricing{Store: s}
		pricing, pErr := repository.FindPricingConfigByStoreID(s.ID)
		if pErr == nil && pricing != nil {
			item.HasPricing = true
			if pricing.UpdatedBy != nil {
				item.LastUpdatedBy = *pricing.UpdatedBy
			}
			item.LastUpdatedAt = pricing.UpdatedAt.Format("02 Jan 2006, 15:04")
		}
		result = append(result, item)
	}

	return result, total, nil
}

// GetPricingByStore mengambil pricing config untuk 1 store.
// Jika belum ada, otomatis buat default config (get-or-create).
func GetPricingByStore(storeID string) (*models.StorePricing, error) {
	pricing, err := repository.FindPricingByStoreID(storeID)

	// Jika belum ada → auto-create default config
	if err != nil {
		createdBy := "system"
		newPricing := &models.StorePricing{
			StoreID:            storeID,
			IsHappyHourEnabled: true,
			IsMixedTimeEnabled: true,
			EdgeCase2h:         "two_x_1h",
			EdgeCase4h:         "3h_plus_1h",
			CreatedBy:          &createdBy,
		}

		if createErr := repository.CreatePricing(newPricing); createErr != nil {
			return nil, errors.New("gagal inisialisasi pricing untuk store ini")
		}

		// Ambil ulang dengan semua relasi ter-preload
		return repository.FindPricingByStoreID(storeID)
	}

	return pricing, nil
}

// CreatePricing membuat pricing config baru untuk store.
func CreatePricing(req dto.CreatePricingRequest, createdBy string) (*models.StorePricing, error) {
	if _, err := repository.FindPricingConfigByStoreID(req.StoreID); err == nil {
		return nil, errors.New("pricing untuk store ini sudah ada")
	}

	edge2h := req.EdgeCase2h
	if edge2h == "" {
		edge2h = "two_x_1h"
	}
	edge4h := req.EdgeCase4h
	if edge4h == "" {
		edge4h = "3h_plus_1h"
	}

	pricing := &models.StorePricing{
		StoreID:            req.StoreID,
		IsHappyHourEnabled: req.IsHappyHourEnabled,
		IsMixedTimeEnabled: req.IsMixedTimeEnabled,
		EdgeCase2h:         edge2h,
		EdgeCase4h:         edge4h,
		CreatedBy:          &createdBy,
	}

	if err := repository.CreatePricing(pricing); err != nil {
		return nil, errors.New("gagal membuat pricing")
	}

	return repository.FindPricingByStoreID(pricing.StoreID)
}

// UpdatePricingConfig update config utama (edge case, toggle HH/mixed).
func UpdatePricingConfig(storeID string, req dto.UpdatePricingRequest, updatedBy string) (*models.StorePricing, error) {
	pricing, err := repository.FindPricingConfigByStoreID(storeID)
	if err != nil {
		return nil, errors.New("pricing tidak ditemukan")
	}

	pricing.IsHappyHourEnabled = req.IsHappyHourEnabled
	pricing.IsMixedTimeEnabled = req.IsMixedTimeEnabled
	if req.EdgeCase2h != "" {
		pricing.EdgeCase2h = req.EdgeCase2h
	}
	if req.EdgeCase4h != "" {
		pricing.EdgeCase4h = req.EdgeCase4h
	}
	pricing.UpdatedBy = &updatedBy

	if err := repository.UpdatePricing(pricing); err != nil {
		return nil, errors.New("gagal update pricing")
	}

	return repository.FindPricingByStoreID(storeID)
}

// DeletePricing soft-delete pricing config.
func DeletePricing(storeID string, deletedBy string) error {
	pricing, err := repository.FindPricingConfigByStoreID(storeID)
	if err != nil {
		return errors.New("pricing tidak ditemukan")
	}
	return repository.SoftDeletePricing(pricing, deletedBy)
}

// ── Happy Hour Schedules ──────────────────────────────────────

// AddSchedule menambahkan rentang waktu happy hour ke store.
func AddSchedule(storeID string, req dto.AddScheduleRequest, createdBy string) (*models.StoreHappyHourSchedule, error) {
	s := &models.StoreHappyHourSchedule{
		StoreID:   storeID,
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
		CreatedBy: &createdBy,
	}
	if err := repository.CreateSchedule(s); err != nil {
		return nil, errors.New("gagal menambahkan jadwal")
	}
	return s, nil
}

// DeleteSchedule menghapus jadwal happy hour.
func DeleteSchedule(id uint, storeID string, deletedBy string) error {
	return repository.DeleteSchedule(id, storeID, deletedBy)
}

// ── Happy Hour Prices ─────────────────────────────────────────

// GetHappyHourPrices mengambil semua harga HH untuk 1 store.
func GetHappyHourPrices(storeID string) ([]models.StoreHappyHourPrice, error) {
	return repository.FindHappyHourPricesByStore(storeID)
}

// BulkUpsertHappyHourPrices insert-or-update harga HH.
func BulkUpsertHappyHourPrices(storeID string, req dto.BulkUpsertHappyHourPricesRequest, actor string) ([]models.StoreHappyHourPrice, error) {
	var inputs []models.StoreHappyHourPrice
	for _, p := range req.Prices {
		by := actor
		inputs = append(inputs, models.StoreHappyHourPrice{
			StoreID:        storeID,
			RoomTemplateID: p.RoomTemplateID,
			PricePerHour:   p.PricePerHour,
			UpdatedBy:      &by,
			CreatedBy:      &by,
		})
	}
	if err := repository.UpsertHappyHourPrices(storeID, inputs); err != nil {
		return nil, errors.New("gagal update harga happy hour")
	}
	return repository.FindHappyHourPricesByStore(storeID)
}

// ── Package Prices ────────────────────────────────────────────

// GetPackagePrices mengambil semua harga paket untuk 1 store.
func GetPackagePrices(storeID string) ([]models.StorePackagePrice, error) {
	return repository.FindPackagePricesByStore(storeID)
}

// BulkUpsertPackagePrices insert-or-update harga paket.
func BulkUpsertPackagePrices(storeID string, req dto.BulkUpsertPackagePricesRequest, actor string) ([]models.StorePackagePrice, error) {
	var inputs []models.StorePackagePrice
	for _, p := range req.Prices {
		by := actor
		inputs = append(inputs, models.StorePackagePrice{
			StoreID:        storeID,
			RoomTemplateID: p.RoomTemplateID,
			DurationHours:  p.DurationHours,
			Price:          p.Price,
			IsCustom:       p.IsCustom,
			UpdatedBy:      &by,
			CreatedBy:      &by,
		})
	}
	if err := repository.UpsertPackagePrices(storeID, inputs); err != nil {
		return nil, errors.New("gagal update harga paket")
	}
	return repository.FindPackagePricesByStore(storeID)
}

// DeletePackagePrice menghapus 1 paket harga.
func DeletePackagePrice(id uint, storeID string, deletedBy string) error {
	return repository.DeletePackagePrice(id, storeID, deletedBy)
}

// ── Flash Sales ───────────────────────────────────────────────

// GetFlashSales mengambil semua flash sale untuk 1 store.
func GetFlashSales(storeID string) ([]models.StoreFlashSale, error) {
	return repository.FindFlashSalesByStore(storeID)
}

// CreateFlashSale membuat flash sale baru.
func CreateFlashSale(storeID string, req dto.CreateFlashSaleRequest, createdBy string) (*models.StoreFlashSale, error) {
	dateFrom, err := time.Parse("2006-01-02", req.DateFrom)
	if err != nil {
		return nil, errors.New("format date_from tidak valid (YYYY-MM-DD)")
	}
	dateTo, err := time.Parse("2006-01-02", req.DateTo)
	if err != nil {
		return nil, errors.New("format date_to tidak valid (YYYY-MM-DD)")
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	desc := req.Description

	sale := &models.StoreFlashSale{
		ID:             uuid.NewString(),
		StoreID:        storeID,
		RoomTemplateID: req.RoomTemplateID,
		Name:           req.Name,
		Description:    &desc,
		DiscountAmount: req.DiscountAmount,
		DateFrom:       dateFrom,
		DateTo:         dateTo,
		TimeFrom:       req.TimeFrom,
		TimeTo:         req.TimeTo,
		IsActive:       isActive,
		CreatedBy:      &createdBy,
	}

	if err := repository.CreateFlashSale(sale); err != nil {
		return nil, errors.New("gagal membuat flash sale")
	}

	return repository.FindFlashSaleByID(sale.ID)
}

// UpdateFlashSale mengupdate flash sale.
func UpdateFlashSale(id string, req dto.UpdateFlashSaleRequest, updatedBy string) (*models.StoreFlashSale, error) {
	sale, err := repository.FindFlashSaleByID(id)
	if err != nil {
		return nil, errors.New("flash sale tidak ditemukan")
	}

	dateFrom, err := time.Parse("2006-01-02", req.DateFrom)
	if err != nil {
		return nil, errors.New("format date_from tidak valid")
	}
	dateTo, err := time.Parse("2006-01-02", req.DateTo)
	if err != nil {
		return nil, errors.New("format date_to tidak valid")
	}

	isActive := sale.IsActive
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	desc := req.Description

	sale.Name = req.Name
	sale.Description = &desc
	sale.DiscountAmount = req.DiscountAmount
	sale.DateFrom = dateFrom
	sale.DateTo = dateTo
	sale.TimeFrom = req.TimeFrom
	sale.TimeTo = req.TimeTo
	sale.IsActive = isActive
	sale.UpdatedBy = &updatedBy

	if err := repository.UpdateFlashSale(sale); err != nil {
		return nil, errors.New("gagal update flash sale")
	}

	return repository.FindFlashSaleByID(sale.ID)
}

// DeleteFlashSale soft-delete flash sale.
func DeleteFlashSale(id string, deletedBy string) error {
	if _, err := repository.FindFlashSaleByID(id); err != nil {
		return errors.New("flash sale tidak ditemukan")
	}
	return repository.SoftDeleteFlashSale(id, deletedBy)
}

// ── Price Calculator ──────────────────────────────────────────

// CalculatePrice menghitung harga booking berdasarkan semua aturan pricing.
// Prioritas: Flash Sale discount > Happy Hour (Weekday) > Normal Hour / Weekend
func CalculatePrice(req dto.CalculatePriceRequest) (*dto.CalculatePriceResponse, error) {
	bookingDate, err := time.Parse("2006-01-02", req.BookingDate)
	if err != nil {
		return nil, errors.New("format booking_date tidak valid (YYYY-MM-DD)")
	}

	startMins := parseTimeToMinutes(req.StartTime)
	endMins := parseTimeToMinutes(req.EndTime)
	if endMins <= startMins {
		endMins += 24 * 60 // lintas tengah malam
	}
	totalMinutes := endMins - startMins
	totalHours := float64(totalMinutes) / 60.0

	pricing, err := repository.FindPricingConfigByStoreID(req.StoreID)
	if err != nil {
		return nil, errors.New("pricing tidak ditemukan untuk store ini")
	}

	packages, err := repository.FindPackagePricesForRoom(req.StoreID, req.RoomTemplateID)
	if err != nil || len(packages) == 0 {
		return nil, errors.New("harga paket belum dikonfigurasi untuk ruangan ini")
	}

	isWeekday := isWeekdayDate(bookingDate)

	var breakdown []dto.PriceBreakdownItem
	var basePrice float64

	if isWeekday && pricing.IsHappyHourEnabled {
		schedules, _ := repository.FindSchedulesByStoreID(req.StoreID)
		hhPrice, hhErr := repository.FindHappyHourPriceForRoom(req.StoreID, req.RoomTemplateID)

		hhMinutes := calculateHappyHourMinutes(startMins, endMins, schedules)
		normalMinutes := totalMinutes - hhMinutes
		
		if hhMinutes > 0 && hhErr == nil {
			hhHours := float64(hhMinutes) / 60.0
			hhTotal := hhHours * hhPrice.PricePerHour
			basePrice += hhTotal
			breakdown = append(breakdown, dto.PriceBreakdownItem{
				TimeRange:   fmt.Sprintf("%s - %s", minutesToTime(startMins), minutesToTime(startMins+hhMinutes)),
				Type:        "Happy Hour",
				Description: fmt.Sprintf("%s Jam × Rp %.0f", formatHours(hhHours), hhPrice.PricePerHour),
				Amount:      hhTotal,
			})
		}

		if normalMinutes > 0 {
			normalHours := int(math.Ceil(float64(normalMinutes) / 60.0))
			normalPrice, normalDesc := findCheapestPackage(normalHours, packages, pricing)
			basePrice += normalPrice
			breakdown = append(breakdown, dto.PriceBreakdownItem{
				TimeRange:   fmt.Sprintf("%s - %s", minutesToTime(startMins+hhMinutes), minutesToTime(endMins)),
				Type:        "Normal Hour",
				Description: normalDesc,
				Amount:      normalPrice,
			})
		}
	} else {
		normalHours := int(math.Ceil(totalHours))
		normalPrice, normalDesc := findCheapestPackage(normalHours, packages, pricing)
		basePrice = normalPrice
		breakdown = append(breakdown, dto.PriceBreakdownItem{
			TimeRange:   fmt.Sprintf("%s - %s", req.StartTime, req.EndTime),
			Type:        "Normal Hour",
			Description: normalDesc,
			Amount:      normalPrice,
		})
	}

	// Cek Flash Sale — prioritas tertinggi (discount dari base price)
	flashDiscount := 0.0
	hasFlash := false
	flashName := ""
	flashSale, flashErr := repository.FindActiveFlashSale(
		req.StoreID, req.RoomTemplateID, bookingDate, req.StartTime, req.EndTime,
	)
	if flashErr == nil && flashSale != nil {
		flashDiscount = flashSale.DiscountAmount
		hasFlash = true
		flashName = flashSale.Name
	}

	finalPrice := basePrice - flashDiscount
	if finalPrice < 0 {
		finalPrice = 0
	}

	return &dto.CalculatePriceResponse{
		TotalHours:    totalHours,
		BasePrice:     basePrice,
		FlashDiscount: flashDiscount,
		FinalPrice:    finalPrice,
		Breakdown:     breakdown,
		HasFlashSale:  hasFlash,
		FlashSaleName: flashName,
	}, nil
}

// ── Helper Functions ──────────────────────────────────────────

// formatHours menampilkan float jam tanpa desimal jika angka bulat (3 bukan 3.0),
// dan dengan 1 desimal jika ada pecahan (1.5).
func formatHours(h float64) string {
	if h == float64(int(h)) {
		return fmt.Sprintf("%d", int(h))
	}
	return fmt.Sprintf("%.1f", h)
}

// parseTimeToMinutes menerima format "HH:MM" maupun "HH:MM:SS" (format MySQL TIME).
func parseTimeToMinutes(t string) int {
	parts := strings.Split(t, ":")
	if len(parts) < 2 {
		return 0
	}
	h, _ := strconv.Atoi(parts[0])
	m, _ := strconv.Atoi(parts[1])
	return h*60 + m
}

func minutesToTime(mins int) string {
	mins = mins % (24 * 60)
	if mins < 0 {
		mins += 24 * 60
	}
	return fmt.Sprintf("%02d:%02d", mins/60, mins%60)
}

// isWeekdayDate mengembalikan true untuk Senin–Kamis.
func isWeekdayDate(date time.Time) bool {
	day := date.Weekday()
	return day >= time.Monday && day <= time.Thursday
}

// calculateHappyHourMinutes menghitung irisan menit booking dengan window happy hour.
func calculateHappyHourMinutes(startMins, endMins int, schedules []models.StoreHappyHourSchedule) int {
	total := 0
	for _, s := range schedules {
		hhStart := parseTimeToMinutes(s.StartTime)
		hhEnd := parseTimeToMinutes(s.EndTime)
		if hhEnd < hhStart {
			hhEnd += 24 * 60 // lintas tengah malam
		}
		overlapStart := intMax(startMins, hhStart)
		overlapEnd := intMin(endMins, hhEnd)
		if overlapEnd > overlapStart {
			total += overlapEnd - overlapStart
		}
	}
	return total
}

// findCheapestPackage mencari kombinasi paket termurah untuk n jam Normal Hour.
func findCheapestPackage(hours int, packages []models.StorePackagePrice, pricing *models.StorePricing) (float64, string) {
	// Edge case 2 jam
	if hours == 2 {
		switch pricing.EdgeCase2h {
		case "two_x_1h":
			if p := getPriceForDuration(1, packages); p > 0 {
				return p * 2, "2× Paket 1 Jam"
			}
		case "force_3h":
			if p := getPriceForDuration(3, packages); p > 0 {
				return p, "Paket 3 Jam"
			}
		}
	}

	// Edge case 4 jam
	if hours == 4 {
		switch pricing.EdgeCase4h {
		case "3h_plus_1h":
			p3h := getPriceForDuration(3, packages)
			p1h := getPriceForDuration(1, packages)
			if p3h > 0 && p1h > 0 {
				return p3h + p1h, "Paket 3 Jam + Paket 1 Jam"
			}
		case "force_5h":
			if p := getPriceForDuration(5, packages); p > 0 {
				return p, "Paket 5 Jam"
			}
		}
	}

	return dpCheapestCombination(hours, packages)
}

// dpCheapestCombination cari kombinasi paket termurah secara greedy (harga per jam terbaik).
func dpCheapestCombination(hours int, packages []models.StorePackagePrice) (float64, string) {
	// Urutkan descending duration supaya greedy memilih paket besar dulu
	pkgs := make([]models.StorePackagePrice, len(packages))
	copy(pkgs, packages)
	sort.Slice(pkgs, func(i, j int) bool {
		return pkgs[i].DurationHours > pkgs[j].DurationHours
	})

	totalPrice := 0.0
	remaining := hours
	var parts []string

	for remaining > 0 {
		best := findBestPackageForRemaining(remaining, pkgs)
		if best == nil {
			// Fallback ke paket 1 jam
			p1h := getPriceForDuration(1, pkgs)
			totalPrice += p1h * float64(remaining)
			if remaining > 1 {
				parts = append(parts, fmt.Sprintf("%d× Paket 1 Jam", remaining))
			} else {
				parts = append(parts, "Paket 1 Jam")
			}
			break
		}
		totalPrice += best.Price
		parts = append(parts, fmt.Sprintf("Paket %d Jam", best.DurationHours))
		remaining -= int(best.DurationHours)
	}

	return totalPrice, strings.Join(parts, " + ")
}

// findBestPackageForRemaining cari paket yang ≤ remaining jam dengan harga/jam termurah.
func findBestPackageForRemaining(remaining int, packages []models.StorePackagePrice) *models.StorePackagePrice {
	var best *models.StorePackagePrice
	bestPPH := math.MaxFloat64
	for i := range packages {
		pkg := &packages[i]
		if int(pkg.DurationHours) <= remaining && pkg.DurationHours > 0 {
			pph := pkg.Price / float64(pkg.DurationHours)
			if pph < bestPPH {
				bestPPH = pph
				best = pkg
			}
		}
	}
	return best
}

func getPriceForDuration(hours uint, packages []models.StorePackagePrice) float64 {
	for _, p := range packages {
		if p.DurationHours == hours {
			return p.Price
		}
	}
	return 0
}

func intMax(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func intMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}
