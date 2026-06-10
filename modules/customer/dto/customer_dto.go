package dto

// ── List Filter ───────────────────────────────────────────────────────────────

type CustomerListFilter struct {
	Search  string `form:"search"`
	Type    string `form:"type"`   // member | regular | "" (all)
	Status  string `form:"status"` // active | inactive | "" (all)
	Page    int    `form:"page"`
	PerPage int    `form:"per_page"`
}

// ── Create ────────────────────────────────────────────────────────────────────

type CreateCustomerRequest struct {
	Name             string `json:"name" binding:"required"`
	Whatsapp         string `json:"whatsapp" binding:"required"`
	Email            string `json:"email"`             // opsional; kalau ada → kirim password via email
	DateOfBirth      string `json:"date_of_birth"`     // format: YYYY-MM-DD
	Gender           string `json:"gender"`            // male | female
	Occupation       string `json:"occupation"`
	Type             string `json:"type"`              // member | regular (default: regular)
	Notes            string `json:"notes"`
	FavoriteRoomTypes []uint `json:"favorite_room_ids"` // []room_template_id
}

// ── Update ────────────────────────────────────────────────────────────────────

type UpdateCustomerRequest struct {
	Name             string `json:"name" binding:"required"`
	Whatsapp         string `json:"whatsapp" binding:"required"`
	Email            string `json:"email"`
	DateOfBirth      string `json:"date_of_birth"`
	Gender           string `json:"gender"`
	Occupation       string `json:"occupation"`
	Type             string `json:"type"`
	Status           string `json:"status"` // active | inactive
	Notes            string `json:"notes"`
	FavoriteRoomTypes []uint `json:"favorite_room_ids"`
}

// ── Update Notes ──────────────────────────────────────────────────────────────

type UpdateCustomerNotesRequest struct {
	Notes string `json:"notes"`
}
