package service

import (
	"errors"

	"game_lounge_be/models"
	"game_lounge_be/modules/banner/dto"
	"game_lounge_be/modules/banner/repository"
)

// GetAllAdmin mengambil semua banner (aktif & nonaktif) untuk halaman admin.
func GetAllAdmin() ([]models.Banner, error) {
	return repository.FindAllAdmin()
}

// GetAllActive mengambil banner aktif saja — dipakai customer (public).
func GetAllActive() ([]models.Banner, error) {
	return repository.FindAllActive()
}

// GetByID mengambil satu banner berdasarkan ID.
func GetByID(id uint) (*models.Banner, error) {
	b, err := repository.FindByID(id)
	if err != nil {
		return nil, errors.New("banner tidak ditemukan")
	}
	return b, nil
}

// Create membuat banner baru.
func Create(req dto.CreateBannerRequest, createdBy string) (*models.Banner, error) {
	// Default aktif kecuali client eksplisit kirim false
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	var subtitle, desc, detailImg *string
	if req.Subtitle != "" {
		subtitle = &req.Subtitle
	}
	if req.Description != "" {
		desc = &req.Description
	}
	if req.DetailImageURL != "" {
		detailImg = &req.DetailImageURL
	}

	banner := &models.Banner{
		Title:          req.Title,
		Subtitle:       subtitle,
		Description:    desc,
		ImageURL:       req.ImageURL,
		DetailImageURL: detailImg,
		SortOrder:      req.SortOrder,
		IsActive:       isActive,
		CreatedBy:      &createdBy,
	}

	if err := repository.Create(banner); err != nil {
		return nil, errors.New("gagal membuat banner")
	}
	return banner, nil
}

// Update memperbarui banner yang sudah ada.
func Update(id uint, req dto.UpdateBannerRequest, updatedBy string) (*models.Banner, error) {
	banner, err := repository.FindByID(id)
	if err != nil {
		return nil, errors.New("banner tidak ditemukan")
	}

	// Pertahankan nilai is_active saat ini jika client tidak kirim field ini
	isActive := banner.IsActive
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	var subtitle, desc, detailImg *string
	if req.Subtitle != "" {
		subtitle = &req.Subtitle
	}
	if req.Description != "" {
		desc = &req.Description
	}
	if req.DetailImageURL != "" {
		detailImg = &req.DetailImageURL
	}

	banner.Title = req.Title
	banner.Subtitle = subtitle
	banner.Description = desc
	banner.ImageURL = req.ImageURL
	banner.DetailImageURL = detailImg
	banner.SortOrder = req.SortOrder
	banner.IsActive = isActive
	banner.UpdatedBy = &updatedBy

	if err := repository.Update(banner); err != nil {
		return nil, errors.New("gagal menyimpan banner")
	}
	return banner, nil
}

// Delete soft-delete sebuah banner.
func Delete(id uint, deletedBy string) error {
	banner, err := repository.FindByID(id)
	if err != nil {
		return errors.New("banner tidak ditemukan")
	}
	return repository.SoftDelete(banner, deletedBy)
}

// ToggleActive membalik status aktif/nonaktif banner.
func ToggleActive(id uint, updatedBy string) (*models.Banner, error) {
	banner, err := repository.FindByID(id)
	if err != nil {
		return nil, errors.New("banner tidak ditemukan")
	}
	banner.IsActive = !banner.IsActive
	banner.UpdatedBy = &updatedBy
	if err := repository.Update(banner); err != nil {
		return nil, errors.New("gagal mengubah status banner")
	}
	return banner, nil
}

// Reorder mengupdate urutan banyak banner sekaligus.
func Reorder(req dto.ReorderRequest) error {
	orders := make([]struct{ ID, SortOrder uint }, 0, len(req.Orders))
	for _, o := range req.Orders {
		orders = append(orders, struct{ ID, SortOrder uint }{o.ID, o.SortOrder})
	}
	return repository.Reorder(orders)
}
