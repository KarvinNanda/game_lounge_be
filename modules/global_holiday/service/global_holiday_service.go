package service

import (
	"errors"
	"log"
	"time"

	"game_lounge_be/config"
	"game_lounge_be/models"
	"game_lounge_be/modules/global_holiday/dto"
	"game_lounge_be/modules/global_holiday/repository"
)

func GetAll(filter dto.GlobalHolidayFilter) ([]models.GlobalHolidaySchedule, int64, error) {
	return repository.FindAll(filter.Search, filter.Year, filter.Page, filter.PerPage)
}

// Create membuat global holiday BARU dan otomatis generate store_holiday_schedules
// untuk SEMUA store yang aktif.
func Create(req dto.CreateGlobalHolidayRequest, createdBy string) (*models.GlobalHolidaySchedule, error) {
	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		return nil, errors.New("format tanggal tidak valid (YYYY-MM-DD)")
	}

	h := &models.GlobalHolidaySchedule{
		Date:      date,
		Name:      req.Name,
		OpenTime:  req.OpenTime,
		CloseTime: req.CloseTime,
		CreatedBy: &createdBy,
	}
	if err := repository.Create(h); err != nil {
		return nil, errors.New("tanggal ini mungkin sudah ada atau gagal disimpan")
	}

	// Auto-populate ke semua store aktif (async, tidak blocking)
	go propagateToAllStores(h)

	return h, nil
}

// Update mengubah global holiday dan sinkronisasi ke store holidays yang belum di-override.
func Update(id uint, req dto.UpdateGlobalHolidayRequest, updatedBy string) (*models.GlobalHolidaySchedule, error) {
	h, err := repository.FindByID(id)
	if err != nil {
		return nil, errors.New("data tidak ditemukan")
	}
	h.Name      = req.Name
	h.OpenTime  = req.OpenTime
	h.CloseTime = req.CloseTime
	h.UpdatedBy = &updatedBy
	if err := repository.Update(h); err != nil {
		return nil, errors.New("gagal menyimpan perubahan")
	}

	// Sinkronisasi ke store holidays yang masih menggunakan jam dari global
	// (yang global_holiday_id = id ini dan belum di-override manual)
	go syncStoreHolidays(h)

	return h, nil
}

func Delete(id uint, deletedBy string) error {
	h, err := repository.FindByID(id)
	if err != nil {
		return errors.New("data tidak ditemukan")
	}
	return repository.SoftDelete(h, deletedBy)
}

// propagateToAllStores membuat store_holiday_schedule untuk semua store aktif.
// Jika store sudah punya holiday di tanggal itu (manual), SKIP — tidak overwrite.
func propagateToAllStores(h *models.GlobalHolidaySchedule) {
	var stores []models.Store
	config.DB.Where("status = 'active' AND deleted_at IS NULL").Find(&stores)

	source := "system"
	for _, store := range stores {
		// Cek apakah store sudah punya holiday di tanggal ini
		var existing models.StoreHolidaySchedule
		err := config.DB.Where(
			"store_id = ? AND DATE(date) = ? AND deleted_at IS NULL",
			store.ID, h.Date.Format("2006-01-02"),
		).First(&existing).Error

		if err != nil {
			// Belum ada → buat baru dari global holiday
			newHoliday := models.StoreHolidaySchedule{
				StoreID:         store.ID,
				Date:            h.Date,
				OpenTime:        h.OpenTime,
				CloseTime:       h.CloseTime,
				GlobalHolidayID: &h.ID,
				CreatedBy:       &source,
			}
			if err := config.DB.Create(&newHoliday).Error; err != nil {
				log.Printf("Gagal buat store holiday untuk store %s: %v", store.ID, err)
			}
		}
		// Jika sudah ada (manual/existing) → SKIP, tidak overwrite
	}
}

// syncStoreHolidays mengupdate jam store holidays yang masih terhubung ke global holiday ini.
// Hanya update yang `global_holiday_id = h.ID` — yang sudah diubah manual TIDAK disentuh.
func syncStoreHolidays(h *models.GlobalHolidaySchedule) {
	config.DB.Model(&models.StoreHolidaySchedule{}).
		Where("global_holiday_id = ? AND deleted_at IS NULL", h.ID).
		Updates(map[string]interface{}{
			"open_time":  h.OpenTime,
			"close_time": h.CloseTime,
			"updated_by": "system",
		})
}
