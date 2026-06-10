package dto

// ── Categories ────────────────────────────────────────────────

type CreateCategoryRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=100"`
	Description string `json:"description"`
	SortOrder   int    `json:"sort_order"`
}

type UpdateCategoryRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=100"`
	Description string `json:"description"`
	SortOrder   int    `json:"sort_order"`
	IsActive    bool   `json:"is_active"`
}

// ── Items ─────────────────────────────────────────────────────

type CreateItemRequest struct {
	CategoryID  uint    `json:"category_id" binding:"required"`
	Name        string  `json:"name" binding:"required,min=1,max=150"`
	Description string  `json:"description"`
	Price       float64 `json:"price" binding:"min=0"`
	SortOrder   int     `json:"sort_order"`
}

type UpdateItemRequest struct {
	CategoryID  uint    `json:"category_id" binding:"required"`
	Name        string  `json:"name" binding:"required,min=1,max=150"`
	Description string  `json:"description"`
	Price       float64 `json:"price" binding:"min=0"`
	SortOrder   int     `json:"sort_order"`
	IsAvailable bool    `json:"is_available"`
	IsActive    bool    `json:"is_active"`
}

// ── Orders ────────────────────────────────────────────────────

type CreateFnbOrderRequest struct {
	BookingID string           `json:"booking_id" binding:"required"`
	Notes     string           `json:"notes"`
	Items     []FnbOrderItemReq `json:"items" binding:"required,min=1"`
}

type FnbOrderItemReq struct {
	ItemID   uint   `json:"item_id" binding:"required"`
	Quantity int    `json:"quantity" binding:"required,min=1"`
	Notes    string `json:"notes"`
}

type UpdateOrderStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=pending preparing delivered cancelled"`
}

// ── Moka Sync ─────────────────────────────────────────────────

// MokaItem representasi item dari Moka API response.
type MokaItem struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"selling_price"`
	ImageURL    string  `json:"image_url"`
	CategoryID  string  `json:"item_category_id"`
}

type MokaCategory struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
