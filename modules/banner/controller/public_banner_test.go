package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"game_lounge_be/config"
	"game_lounge_be/models"
)

// internalFields = kolom audit yang tidak boleh keluar di endpoint public
// (created_by/updated_by berisi username staff).
var internalFields = []string{"created_by", "updated_by", "deleted_by", "deleted_at", "is_active"}

func newTestBanner(t *testing.T, active bool) *models.Banner {
	t.Helper()
	staff := "TEST-staff"
	b := &models.Banner{Title: "TEST banner", ImageURL: "/x.jpg", IsActive: true, CreatedBy: &staff, UpdatedBy: &staff}
	if err := config.DB.Create(b).Error; err != nil {
		t.Fatal(err)
	}
	if !active {
		config.DB.Model(b).Update("is_active", false)
	}
	t.Cleanup(func() { config.DB.Unscoped().Delete(&models.Banner{}, b.ID) })
	return b
}

func decodeData(t *testing.T, body []byte) json.RawMessage {
	t.Helper()
	var resp struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatal(err)
	}
	return resp.Data
}

func assertNoInternal(t *testing.T, obj map[string]any) {
	t.Helper()
	for _, f := range internalFields {
		if _, ok := obj[f]; ok {
			t.Errorf("field internal %q ikut terkirim: %v", f, obj)
		}
	}
}

func TestPublicBannerByID_TanpaFieldInternal(t *testing.T) {
	skipIfNoDB(t)
	b := newTestBanner(t, true)
	w := serve(http.MethodGet, fmt.Sprintf("/banners/%d", b.ID), "", "", withParam("id", b.ID, GetPublicByID))
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	var obj map[string]any
	json.Unmarshal(decodeData(t, w.Body.Bytes()), &obj)
	assertNoInternal(t, obj)
	if obj["title"] != "TEST banner" {
		t.Errorf("title hilang: %v", obj)
	}
}

func TestPublicBannerByID_NonaktifTidakDitemukan(t *testing.T) {
	skipIfNoDB(t)
	b := newTestBanner(t, false)
	w := serve(http.MethodGet, fmt.Sprintf("/banners/%d", b.ID), "", "", withParam("id", b.ID, GetPublicByID))
	if w.Code != http.StatusNotFound {
		t.Fatalf("banner nonaktif: want 404, got %d", w.Code)
	}
}

func TestPublicBannerList_TanpaFieldInternal(t *testing.T) {
	skipIfNoDB(t)
	newTestBanner(t, true)
	w := serve(http.MethodGet, "/banners", "", "", GetAllPublic)
	var list []map[string]any
	json.Unmarshal(decodeData(t, w.Body.Bytes()), &list)
	if len(list) == 0 {
		t.Fatal("list kosong")
	}
	for _, obj := range list {
		assertNoInternal(t, obj)
	}
}
