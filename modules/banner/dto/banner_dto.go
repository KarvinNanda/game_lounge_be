package dto

import "game_lounge_be/models"

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

// PublicBanner = banner untuk endpoint public. Whitelist field: kolom audit
// (created_by/updated_by berisi username staff) tidak boleh keluar.
type PublicBanner struct {
	ID             uint    `json:"id"`
	Title          string  `json:"title"`
	Subtitle       *string `json:"subtitle"`
	Description    *string `json:"description"`
	ImageURL       string  `json:"image_url"`
	DetailImageURL *string `json:"detail_image_url"`
	SortOrder      uint    `json:"sort_order"`
}

func ToPublicBanner(b models.Banner) PublicBanner {
	return PublicBanner{
		ID: b.ID, Title: b.Title, Subtitle: b.Subtitle, Description: b.Description,
		ImageURL: b.ImageURL, DetailImageURL: b.DetailImageURL, SortOrder: b.SortOrder,
	}
}
