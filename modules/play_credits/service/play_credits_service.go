package service

import (
	"errors"
	"math"
	"time"

	"game_lounge_be/models"
	"game_lounge_be/modules/play_credits/dto"
	"game_lounge_be/modules/play_credits/repository"
	"game_lounge_be/utils"

	"github.com/google/uuid"
)

// ── Package ───────────────────────────────────────────────────────────────────

// GetAllPackages mengambil list paket dengan filter.
func GetAllPackages(filter dto.PackageFilter) ([]models.PlayCreditsPackage, int64, error) {
	return repository.FindAllPackages(filter.Search, filter.StoreID, filter.Status, filter.Page, filter.PerPage)
}

// GetActivePackages mengambil semua paket aktif (untuk dropdown).
func GetActivePackages() ([]models.PlayCreditsPackage, error) {
	return repository.FindAllActivePackages()
}

// GetPackageByID mengambil detail paket.
func GetPackageByID(id string) (*models.PlayCreditsPackage, error) {
	pkg, err := repository.FindPackageByID(id)
	if err != nil {
		return nil, errors.New("paket tidak ditemukan")
	}
	return pkg, nil
}

// CreatePackage membuat paket baru.
func CreatePackage(req dto.CreatePackageRequest, iconURL string, createdBy string) (*models.PlayCreditsPackage, error) {
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	pkg := &models.PlayCreditsPackage{
		ID:               uuid.NewString(),
		Name:             req.Name,
		IconURL:          nilString(iconURL),
		TotalHours:       req.TotalHours,
		Price:            req.Price,
		ValidityDays:     req.ValidityDays,
		Description:      nilString(req.Description),
		IsActive:         isActive,
		ApplyToAllStores: req.ApplyToAllStores,
		CreatedBy:        &createdBy,
	}

	if err := repository.CreatePackage(pkg); err != nil {
		if iconURL != "" {
			utils.DeleteFile(iconURL)
		}
		return nil, errors.New("gagal membuat paket")
	}

	if !req.ApplyToAllStores && len(req.StoreIDs) > 0 {
		_ = repository.SyncPackageStores(pkg.ID, req.StoreIDs)
	}

	return repository.FindPackageByID(pkg.ID)
}

// UpdatePackage mengupdate paket.
func UpdatePackage(id string, req dto.UpdatePackageRequest, iconURL string, updatedBy string) (*models.PlayCreditsPackage, error) {
	pkg, err := repository.FindPackageByID(id)
	if err != nil {
		return nil, errors.New("paket tidak ditemukan")
	}

	oldIcon := ""
	if pkg.IconURL != nil {
		oldIcon = *pkg.IconURL
	}

	isActive := pkg.IsActive
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	pkg.Name = req.Name
	pkg.TotalHours = req.TotalHours
	pkg.Price = req.Price
	pkg.ValidityDays = req.ValidityDays
	pkg.Description = nilString(req.Description)
	pkg.IsActive = isActive
	pkg.ApplyToAllStores = req.ApplyToAllStores
	pkg.UpdatedBy = &updatedBy

	if iconURL != "" {
		pkg.IconURL = &iconURL
	}

	if err := repository.UpdatePackage(pkg); err != nil {
		return nil, errors.New("gagal update paket")
	}

	if !req.ApplyToAllStores {
		_ = repository.SyncPackageStores(pkg.ID, req.StoreIDs)
	} else {
		_ = repository.SyncPackageStores(pkg.ID, []string{})
	}

	// Hapus icon lama setelah update berhasil
	if iconURL != "" && oldIcon != "" {
		utils.DeleteFile(oldIcon)
	}

	return repository.FindPackageByID(pkg.ID)
}

// DeletePackage soft-delete paket.
func DeletePackage(id string, deletedBy string) error {
	pkg, err := repository.FindPackageByID(id)
	if err != nil {
		return errors.New("paket tidak ditemukan")
	}
	return repository.SoftDeletePackage(pkg, deletedBy)
}

// ── Member Credits ────────────────────────────────────────────────────────────

// MemberCreditResponse adalah response dengan computed fields tambahan.
type MemberCreditResponse struct {
	models.CustomerPlayCredit
	UsedHours   float64 `json:"used_hours"`
	PercentUsed float64 `json:"percent_used"`
	DaysLeft    int     `json:"days_left"`
	IsExpired   bool    `json:"is_expired"`
}

// enrichCredit menambahkan computed fields ke credit.
func enrichCredit(c models.CustomerPlayCredit) MemberCreditResponse {
	usedHours := c.TotalHours - c.RemainingHours
	percentUsed := 0.0
	if c.TotalHours > 0 {
		percentUsed = math.Round((usedHours/c.TotalHours)*100*10) / 10
	}
	daysLeft := int(time.Until(c.ExpiresAt).Hours() / 24)
	isExpired := time.Now().After(c.ExpiresAt)

	return MemberCreditResponse{
		CustomerPlayCredit: c,
		UsedHours:          usedHours,
		PercentUsed:        percentUsed,
		DaysLeft:           daysLeft,
		IsExpired:          isExpired,
	}
}

// GetAllMemberCredits mengambil list credits member.
func GetAllMemberCredits(filter dto.MemberCreditFilter) ([]MemberCreditResponse, int64, error) {
	credits, total, err := repository.FindAllMemberCredits(
		filter.Search, filter.PackageID, filter.StoreID,
		filter.Status, filter.Whatsapp, filter.Page, filter.PerPage,
	)
	if err != nil {
		return nil, 0, err
	}

	var result []MemberCreditResponse
	for _, c := range credits {
		result = append(result, enrichCredit(c))
	}
	if result == nil {
		result = []MemberCreditResponse{}
	}
	return result, total, nil
}

// GetCreditByID mengambil detail 1 credit.
func GetCreditByID(id string) (*MemberCreditResponse, error) {
	credit, err := repository.FindCreditByID(id)
	if err != nil {
		return nil, errors.New("credits tidak ditemukan")
	}
	r := enrichCredit(*credit)
	return &r, nil
}

// AssignCredit admin assign credits ke customer secara manual.
func AssignCredit(req dto.AssignCreditRequest, createdBy string) (*MemberCreditResponse, error) {
	pkg, err := repository.FindPackageByID(req.PackageID)
	if err != nil {
		return nil, errors.New("paket tidak ditemukan")
	}

	purchasedAt := time.Now()
	expiresAt := purchasedAt.AddDate(0, 0, int(pkg.ValidityDays))

	payAmount := req.PaymentAmount
	if payAmount == 0 {
		payAmount = pkg.Price
	}

	credit := &models.CustomerPlayCredit{
		ID:             uuid.NewString(),
		CustomerID:     req.CustomerID,
		PackageID:      req.PackageID,
		TotalHours:     pkg.TotalHours,
		RemainingHours: pkg.TotalHours,
		PurchasedAt:    purchasedAt,
		ExpiresAt:      expiresAt,
		PaymentMethod:  "manual",
		PaymentAmount:  &payAmount,
		Notes:          nilString(req.Notes),
		IsActive:       true,
		CreatedBy:      &createdBy,
	}

	if err := repository.CreateCredit(credit); err != nil {
		return nil, errors.New("gagal assign credits")
	}

	return GetCreditByID(credit.ID)
}

// AdjustCredit admin adjust jam dan/atau perpanjang masa berlaku.
// adjust_hours: positif = tambah jam, negatif = kurangi jam
// extend_days: hari yang ditambahkan ke expires_at
func AdjustCredit(id string, req dto.AdjustCreditRequest, updatedBy string) (*MemberCreditResponse, error) {
	credit, err := repository.FindCreditByID(id)
	if err != nil {
		return nil, errors.New("credits tidak ditemukan")
	}

	// Adjust jam
	if req.AdjustHours != 0 {
		newRemaining := credit.RemainingHours + req.AdjustHours
		if newRemaining < 0 {
			return nil, errors.New("sisa jam tidak boleh negatif")
		}
		credit.RemainingHours = newRemaining
	}

	// Perpanjang masa berlaku
	if req.ExtendDays > 0 {
		credit.ExpiresAt = credit.ExpiresAt.AddDate(0, 0, req.ExtendDays)
	}

	// Append notes dengan timestamp & actor
	if req.Notes != "" {
		entry := "[" + time.Now().Format("02/01/2006 15:04") + " - " + updatedBy + "] " + req.Notes
		if credit.Notes != nil && *credit.Notes != "" {
			combined := *credit.Notes + "\n" + entry
			credit.Notes = &combined
		} else {
			credit.Notes = &entry
		}
	}

	credit.UpdatedBy = &updatedBy

	if err := repository.UpdateCredit(credit); err != nil {
		return nil, errors.New("gagal update credits")
	}

	return GetCreditByID(credit.ID)
}

// DeleteCredit soft-delete credit.
func DeleteCredit(id string, deletedBy string) error {
	credit, err := repository.FindCreditByID(id)
	if err != nil {
		return errors.New("credits tidak ditemukan")
	}
	return repository.SoftDeleteCredit(credit, deletedBy)
}

// nilString helper: kembalikan nil jika string kosong.
func nilString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
