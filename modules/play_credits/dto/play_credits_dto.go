package dto

// ── Package ───────────────────────────────────────────────────────────────────

type CreatePackageRequest struct {
	Name             string   `form:"name" binding:"required,min=2,max=100"`
	TotalHours       float64  `form:"total_hours" binding:"required,min=0.5"`
	Price            float64  `form:"price" binding:"required,min=0"`
	ValidityDays     uint     `form:"validity_days" binding:"required,min=1"`
	Description      string   `form:"description"`
	IsActive         *bool    `form:"is_active"`
	ApplyToAllStores bool     `form:"apply_to_all_stores"`
	StoreIDs         []string `form:"store_ids[]"`
}

type UpdatePackageRequest struct {
	Name             string   `form:"name" binding:"required,min=2,max=100"`
	TotalHours       float64  `form:"total_hours" binding:"required,min=0.5"`
	Price            float64  `form:"price" binding:"required,min=0"`
	ValidityDays     uint     `form:"validity_days" binding:"required,min=1"`
	Description      string   `form:"description"`
	IsActive         *bool    `form:"is_active"`
	ApplyToAllStores bool     `form:"apply_to_all_stores"`
	StoreIDs         []string `form:"store_ids[]"`
}

type PackageFilter struct {
	Search  string `form:"search"`
	StoreID string `form:"store_id"`
	Status  string `form:"status"`
	Page    int    `form:"page,default=1"`
	PerPage int    `form:"per_page,default=10"`
}

// ── Member Credits ────────────────────────────────────────────────────────────

// AssignCreditRequest dipakai admin untuk assign credits ke customer secara manual.
type AssignCreditRequest struct {
	CustomerID    string  `json:"customer_id" binding:"required"`
	PackageID     string  `json:"package_id" binding:"required"`
	PaymentAmount float64 `json:"payment_amount"`
	Notes         string  `json:"notes"`
}

// AdjustCreditRequest dipakai admin untuk Tambah/Kurangi jam dan Perpanjang masa berlaku.
type AdjustCreditRequest struct {
	AdjustHours float64 `json:"adjust_hours"` // positif = tambah, negatif = kurangi
	ExtendDays  int     `json:"extend_days"`  // hari yang ditambahkan ke expires_at
	Notes       string  `json:"notes"`
}

type MemberCreditFilter struct {
	Search    string `form:"search"`
	PackageID string `form:"package_id"`
	StoreID   string `form:"store_id"`
	Status    string `form:"status"` // "active" | "expired"
	Whatsapp  string `form:"whatsapp"`
	Page      int    `form:"page,default=1"`
	PerPage   int    `form:"per_page,default=10"`
}
