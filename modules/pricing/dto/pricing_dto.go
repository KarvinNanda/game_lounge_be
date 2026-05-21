package dto

// ── Pricing Config ────────────────────────────────────────────

type CreatePricingRequest struct {
	StoreID            string `json:"store_id" binding:"required"`
	IsHappyHourEnabled bool   `json:"is_happy_hour_enabled"`
	IsMixedTimeEnabled bool   `json:"is_mixed_time_enabled"`
	EdgeCase2h         string `json:"edge_case_2h"`
	EdgeCase4h         string `json:"edge_case_4h"`
}

type UpdatePricingRequest struct {
	IsHappyHourEnabled bool   `json:"is_happy_hour_enabled"`
	IsMixedTimeEnabled bool   `json:"is_mixed_time_enabled"`
	EdgeCase2h         string `json:"edge_case_2h"`
	EdgeCase4h         string `json:"edge_case_4h"`
}

type PricingListFilter struct {
	Search  string `form:"search"`
	Page    int    `form:"page,default=1"`
	PerPage int    `form:"per_page,default=20"`
}

// ── Happy Hour Schedule ───────────────────────────────────────

type AddScheduleRequest struct {
	StartTime string `json:"start_time" binding:"required"`
	EndTime   string `json:"end_time" binding:"required"`
}

// ── Happy Hour Prices ─────────────────────────────────────────

type HappyHourPriceInput struct {
	RoomTemplateID uint    `json:"room_template_id" binding:"required"`
	PricePerHour   float64 `json:"price_per_hour" binding:"required,min=0"`
}

type BulkUpsertHappyHourPricesRequest struct {
	Prices []HappyHourPriceInput `json:"prices" binding:"required"`
}

// ── Package Prices ────────────────────────────────────────────

type PackagePriceInput struct {
	RoomTemplateID uint    `json:"room_template_id" binding:"required"`
	DurationHours  uint    `json:"duration_hours" binding:"required,min=1"`
	Price          float64 `json:"price" binding:"required,min=0"`
	IsCustom       bool    `json:"is_custom"`
}

type BulkUpsertPackagePricesRequest struct {
	Prices []PackagePriceInput `json:"prices" binding:"required"`
}

// ── Flash Sale ────────────────────────────────────────────────

type CreateFlashSaleRequest struct {
	RoomTemplateID uint    `json:"room_template_id" binding:"required"`
	Name           string  `json:"name" binding:"required,min=2,max=150"`
	Description    string  `json:"description"`
	PricePerHour   float64 `json:"price_per_hour" binding:"required,gt=0"`
	DateFrom       string  `json:"date_from" binding:"required"`
	DateTo         string  `json:"date_to" binding:"required"`
	TimeFrom       string  `json:"time_from" binding:"required"`
	TimeTo         string  `json:"time_to" binding:"required"`
	IsActive       *bool   `json:"is_active"`
}

type UpdateFlashSaleRequest struct {
	RoomTemplateID uint    `json:"room_template_id" binding:"required"`
	Name         string  `json:"name" binding:"required,min=2,max=150"`
	Description  string  `json:"description"`
	PricePerHour float64 `json:"price_per_hour" binding:"required,gt=0"`
	DateFrom     string  `json:"date_from" binding:"required"`
	DateTo       string  `json:"date_to" binding:"required"`
	TimeFrom     string  `json:"time_from" binding:"required"`
	TimeTo       string  `json:"time_to" binding:"required"`
	IsActive     *bool   `json:"is_active"`
}

// ── Price Calculator ──────────────────────────────────────────

type CalculatePriceRequest struct {
	StoreID        string `json:"store_id" binding:"required"`
	RoomTemplateID uint   `json:"room_template_id" binding:"required"`
	BookingDate    string `json:"booking_date" binding:"required"`
	StartTime      string `json:"start_time" binding:"required"`
	EndTime        string `json:"end_time" binding:"required"`
}

type PriceBreakdownItem struct {
	TimeRange   string  `json:"time_range"`
	Type        string  `json:"type"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
}

type CalculatePriceResponse struct {
	TotalHours    float64              `json:"total_hours"`
	BasePrice     float64              `json:"base_price"`
	FlashDiscount float64              `json:"flash_discount"`
	FinalPrice    float64              `json:"final_price"`
	Breakdown     []PriceBreakdownItem `json:"breakdown"`
	HasFlashSale  bool                 `json:"has_flash_sale"`
	FlashSaleName string               `json:"flash_sale_name,omitempty"`
}
