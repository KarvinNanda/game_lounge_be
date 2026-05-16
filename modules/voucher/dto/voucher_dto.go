package dto

type CreateVoucherRequest struct {
	Name          string   `json:"name" binding:"required,min=2,max=150"`
	Code          string   `json:"code" binding:"required,min=2,max=50"`
	Description   string   `json:"description"`
	Type          string   `json:"type" binding:"required,oneof=booking play_credits both"`
	DiscountType  string   `json:"discount_type" binding:"required,oneof=percentage nominal"`
	DiscountValue float64  `json:"discount_value" binding:"required,min=0"`
	MaxDiscount   float64  `json:"max_discount"`
	MinPurchase   float64  `json:"min_purchase"`
	StartDate     string   `json:"start_date" binding:"required"`
	EndDate       string   `json:"end_date"`
	SendChannel   string   `json:"send_channel"`
	IsAllStores   bool     `json:"is_all_stores"`
	StoreIDs      []string `json:"store_ids"`
}

type UpdateVoucherRequest struct {
	Name          string   `json:"name" binding:"required,min=2,max=150"`
	Description   string   `json:"description"`
	Type          string   `json:"type" binding:"required,oneof=booking play_credits both"`
	DiscountType  string   `json:"discount_type" binding:"required,oneof=percentage nominal"`
	DiscountValue float64  `json:"discount_value" binding:"required,min=0"`
	MaxDiscount   float64  `json:"max_discount"`
	MinPurchase   float64  `json:"min_purchase"`
	StartDate     string   `json:"start_date" binding:"required"`
	EndDate       string   `json:"end_date"`
	IsAllStores   bool     `json:"is_all_stores"`
	StoreIDs      []string `json:"store_ids"`
	IsActive      *bool    `json:"is_active"`
}

type VoucherFilter struct {
	Search  string `form:"search"`
	Status  string `form:"status"`
	Type    string `form:"type"`
	StoreID string `form:"store_id"`
	Page    int    `form:"page,default=1"`
	PerPage int    `form:"per_page,default=10"`
}

// ValidateVoucherRequest dipakai modul Booking nanti untuk cek voucher.
type ValidateVoucherRequest struct {
	Code       string  `json:"code" binding:"required"`
	CustomerID string  `json:"customer_id" binding:"required"`
	StoreID    string  `json:"store_id" binding:"required"`
	Amount     float64 `json:"amount" binding:"required,min=0"`
	UseType    string  `json:"use_type" binding:"required,oneof=booking play_credits"`
}

type ValidateVoucherResponse struct {
	IsValid        bool    `json:"is_valid"`
	VoucherID      string  `json:"voucher_id"`
	Code           string  `json:"code"`
	DiscountAmount float64 `json:"discount_amount"`
	Message        string  `json:"message"`
}
