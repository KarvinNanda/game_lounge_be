package repository

import (
	"testing"
	"time"

	"game_lounge_be/config"
	"game_lounge_be/models"
	"game_lounge_be/modules/global_holiday/dto"
)

// Tanggal far-future yang dipakai sebagai fixture agar tidak konflik dengan data produksi.
// Semua tanggal test menggunakan tahun 2099.
var (
	testDate1 = time.Date(2099, 1, 10, 0, 0, 0, 0, time.Local)
	testDate2 = time.Date(2099, 3, 25, 0, 0, 0, 0, time.Local)
	testDate3 = time.Date(2099, 12, 25, 0, 0, 0, 0, time.Local)
)

// cleanupDate hapus global holiday test pada tanggal tertentu (unscoped agar soft-deleted juga hilang).
func cleanupDate(date time.Time) {
	config.DB.Unscoped().Where("DATE(date) = ?", date.Format("2006-01-02")).
		Delete(&models.GlobalHolidaySchedule{})
}

// ── Create + FindByID ─────────────────────────────────────────

func TestCreate_FindByID(t *testing.T) {
	skipIfNoDB(t)
	cleanupDate(testDate1)

	actor := "test"
	h := &models.GlobalHolidaySchedule{
		Date:      testDate1,
		Name:      "TEST Hari Raya Idul Fitri",
		OpenTime:  "10:00",
		CloseTime: "22:00",
		CreatedBy: &actor,
	}

	if err := Create(h); err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if h.ID == 0 {
		t.Fatal("ID harus terisi setelah Create")
	}
	defer cleanupDate(testDate1)

	found, err := FindByID(h.ID)
	if err != nil {
		t.Fatalf("FindByID error: %v", err)
	}
	if found.Name != h.Name {
		t.Errorf("name: want %q, got %q", h.Name, found.Name)
	}
	if found.OpenTime != "10:00:00" && found.OpenTime != "10:00" {
		t.Errorf("open_time: got %q", found.OpenTime)
	}
}

func TestFindByID_TidakAda(t *testing.T) {
	skipIfNoDB(t)

	_, err := FindByID(999999999)
	if err == nil {
		t.Error("ID yang tidak ada harus return error")
	}
}

// ── FindByDate ────────────────────────────────────────────────

func TestFindByDate(t *testing.T) {
	skipIfNoDB(t)
	cleanupDate(testDate2)

	actor := "test"
	h := &models.GlobalHolidaySchedule{
		Date:      testDate2,
		Name:      "TEST Hari Kemerdekaan",
		OpenTime:  "09:00",
		CloseTime: "21:00",
		CreatedBy: &actor,
	}
	if err := Create(h); err != nil {
		t.Fatalf("Create error: %v", err)
	}
	defer cleanupDate(testDate2)

	found, err := FindByDate(testDate2)
	if err != nil {
		t.Fatalf("FindByDate error: %v", err)
	}
	if found.ID != h.ID {
		t.Errorf("FindByDate: want ID=%d, got ID=%d", h.ID, found.ID)
	}
}

func TestFindByDate_TidakAda(t *testing.T) {
	skipIfNoDB(t)

	_, err := FindByDate(time.Date(2099, 7, 7, 0, 0, 0, 0, time.Local))
	if err == nil {
		t.Error("tanggal yang tidak ada harus return error")
	}
}

// ── FindAll + Filter ──────────────────────────────────────────

func TestFindAll_WithYearAndSearchFilter(t *testing.T) {
	skipIfNoDB(t)
	cleanupDate(testDate3)

	actor := "test"
	h := &models.GlobalHolidaySchedule{
		Date:      testDate3,
		Name:      "TEST Natal Akhir Tahun",
		OpenTime:  "10:00",
		CloseTime: "23:00",
		CreatedBy: &actor,
	}
	if err := Create(h); err != nil {
		t.Fatalf("Create error: %v", err)
	}
	defer cleanupDate(testDate3)

	filter := dto.GlobalHolidayFilter{
		Search:  "Natal Akhir",
		Year:    2099,
		Page:    1,
		PerPage: 20,
	}
	holidays, total, err := FindAll(filter.Search, filter.Year, filter.Page, filter.PerPage)
	if err != nil {
		t.Fatalf("FindAll error: %v", err)
	}
	if total < 1 {
		t.Error("total harus >= 1 setelah insert")
	}

	found := false
	for _, hol := range holidays {
		if hol.ID == h.ID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("holiday test tidak ditemukan di FindAll (total=%d)", total)
	}
}

func TestFindAll_Pagination(t *testing.T) {
	skipIfNoDB(t)

	// Per_page=1 → hanya ambil 1 record
	holidays, _, err := FindAll("", 2099, 1, 1)
	if err != nil {
		t.Fatalf("FindAll pagination error: %v", err)
	}
	if len(holidays) > 1 {
		t.Errorf("per_page=1 tapi dapat %d records", len(holidays))
	}
}

// ── Update ────────────────────────────────────────────────────

func TestUpdate(t *testing.T) {
	skipIfNoDB(t)

	testDateUpdate := time.Date(2099, 2, 14, 0, 0, 0, 0, time.Local)
	cleanupDate(testDateUpdate)

	actor := "test"
	h := &models.GlobalHolidaySchedule{
		Date:      testDateUpdate,
		Name:      "TEST Before Update",
		OpenTime:  "10:00",
		CloseTime: "20:00",
		CreatedBy: &actor,
	}
	if err := Create(h); err != nil {
		t.Fatalf("Create error: %v", err)
	}
	defer cleanupDate(testDateUpdate)

	updatedBy := "test_updater"
	h.Name = "TEST After Update"
	h.OpenTime = "11:00"
	h.UpdatedBy = &updatedBy

	if err := Update(h); err != nil {
		t.Fatalf("Update error: %v", err)
	}

	found, _ := FindByID(h.ID)
	if found.Name != "TEST After Update" {
		t.Errorf("name setelah update: want 'TEST After Update', got %q", found.Name)
	}
	if found.UpdatedBy == nil || *found.UpdatedBy != "test_updater" {
		t.Errorf("updated_by: want 'test_updater', got %v", found.UpdatedBy)
	}
}

// ── SoftDelete ────────────────────────────────────────────────

func TestSoftDelete(t *testing.T) {
	skipIfNoDB(t)

	testDateDelete := time.Date(2099, 4, 1, 0, 0, 0, 0, time.Local)
	cleanupDate(testDateDelete)

	actor := "test"
	h := &models.GlobalHolidaySchedule{
		Date:      testDateDelete,
		Name:      "TEST Akan Dihapus",
		OpenTime:  "10:00",
		CloseTime: "22:00",
		CreatedBy: &actor,
	}
	if err := Create(h); err != nil {
		t.Fatalf("Create error: %v", err)
	}
	defer cleanupDate(testDateDelete) // unscoped cleanup di akhir

	if err := SoftDelete(h, "test_deleter"); err != nil {
		t.Fatalf("SoftDelete error: %v", err)
	}

	// Setelah soft delete → FindByID harus gagal (WHERE deleted_at IS NULL)
	_, err := FindByID(h.ID)
	if err == nil {
		t.Error("FindByID harus error setelah soft delete")
	}

	// Verifikasi deleted_by tersimpan
	var raw models.GlobalHolidaySchedule
	config.DB.Unscoped().First(&raw, h.ID)
	if raw.DeletedBy == nil || *raw.DeletedBy != "test_deleter" {
		t.Errorf("deleted_by: want 'test_deleter', got %v", raw.DeletedBy)
	}
	if raw.DeletedAt == nil {
		t.Error("deleted_at harus terisi setelah soft delete")
	}
}
