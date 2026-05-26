package dto

type CreateBannerRequest struct {
	Title          string `json:"title" binding:"required,min=2,max=150"`
	Subtitle       string `json:"subtitle"`
	Description    string `json:"description"`
	ImageURL       string `json:"image_url" binding:"required"`
	DetailImageURL string `json:"detail_image_url"`
	SortOrder      uint   `json:"sort_order"`
	IsActive       *bool  `json:"is_active"` // pointer agar bisa bedakan null vs false
}

type UpdateBannerRequest struct {
	Title          string `json:"title" binding:"required,min=2,max=150"`
	Subtitle       string `json:"subtitle"`
	Description    string `json:"description"`
	ImageURL       string `json:"image_url" binding:"required"`
	DetailImageURL string `json:"detail_image_url"`
	SortOrder      uint   `json:"sort_order"`
	IsActive       *bool  `json:"is_active"`
}

// ReorderRequest untuk update urutan banyak banner sekaligus.
type ReorderRequest struct {
	Orders []BannerOrder `json:"orders" binding:"required"`
}

type BannerOrder struct {
	ID        uint `json:"id" binding:"required"`
	SortOrder uint `json:"sort_order"`
}
