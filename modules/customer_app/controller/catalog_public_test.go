package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"game_lounge_be/config"
	"game_lounge_be/models"
)

// Kolom audit/internal yang tidak boleh keluar di endpoint public.
var internalFields = []string{"created_by", "updated_by", "deleted_by", "deleted_at", "is_active", "category", "category_id"}

func assertNoInternal(t *testing.T, where string, obj map[string]any) {
	t.Helper()
	for _, f := range internalFields {
		if _, ok := obj[f]; ok {
			t.Errorf("%s: field internal %q ikut terkirim", where, f)
		}
	}
}

func decodeData(t *testing.T, body []byte, v any) {
	t.Helper()
	var resp struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(resp.Data, v); err != nil {
		t.Fatal(err)
	}
}

// newTestTemplate: template aktif dengan 1 facility aktif dan 1 nonaktif.
func newTestTemplate(t *testing.T) *models.RoomTemplate {
	t.Helper()
	staff := "TEST-staff"
	tpl := &models.RoomTemplate{Name: "TEST tpl", CapacityMax: 4, IsActive: true, CreatedBy: &staff}
	if err := config.DB.Create(tpl).Error; err != nil {
		t.Fatal(err)
	}
	var facs []models.Facility
	for _, active := range []bool{true, false} {
		f := models.Facility{CategoryID: 999999, Name: fmt.Sprintf("TEST fac %v", active), IsActive: true, CreatedBy: &staff}
		if err := config.DB.Create(&f).Error; err != nil {
			t.Fatal(err)
		}
		if !active {
			config.DB.Model(&f).Update("is_active", false)
		}
		config.DB.Create(&models.RoomTemplateFacility{RoomTemplateID: tpl.ID, FacilityID: f.ID})
		facs = append(facs, f)
	}
	t.Cleanup(func() {
		config.DB.Where("room_template_id = ?", tpl.ID).Delete(&models.RoomTemplateFacility{})
		for _, f := range facs {
			config.DB.Unscoped().Delete(&models.Facility{}, f.ID)
		}
		config.DB.Unscoped().Delete(&models.RoomTemplate{}, tpl.ID)
	})
	return tpl
}

func TestPublicRoomTemplateByID_TanpaFieldInternal(t *testing.T) {
	skipIfNoDB(t)
	tpl := newTestTemplate(t)
	w := serve(http.MethodGet, fmt.Sprintf("/room-templates/%d", tpl.ID), "", "", withParam("id", tpl.ID, PublicGetRoomTemplateByID))
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	var obj struct {
		Facilities []map[string]any `json:"facilities"`
	}
	decodeData(t, w.Body.Bytes(), &obj)
	if len(obj.Facilities) != 1 {
		t.Fatalf("hanya facility aktif yang boleh tampil, dapat %d: %v", len(obj.Facilities), obj.Facilities)
	}
	assertNoInternal(t, "facility", obj.Facilities[0])
}

func TestPublicRoomTemplates_TanpaFieldInternal(t *testing.T) {
	skipIfNoDB(t)
	newTestTemplate(t)
	w := serve(http.MethodGet, "/room-templates", "", "", PublicGetRoomTemplates)
	var list []map[string]any
	decodeData(t, w.Body.Bytes(), &list)
	if len(list) == 0 {
		t.Fatal("list kosong")
	}
	for _, obj := range list {
		assertNoInternal(t, "template", obj)
		if _, ok := obj["min_price"]; !ok {
			t.Errorf("min_price hilang: %v", obj)
		}
	}
}

func TestPublicRoomTemplateByID_TanpaFacilityTetapKirimArray(t *testing.T) {
	skipIfNoDB(t)
	tpl := &models.RoomTemplate{Name: "TEST tpl kosong", CapacityMax: 2, IsActive: true}
	if err := config.DB.Create(tpl).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { config.DB.Unscoped().Delete(&models.RoomTemplate{}, tpl.ID) })

	w := serve(http.MethodGet, "/x", "", "", withParam("id", tpl.ID, PublicGetRoomTemplateByID))
	var obj map[string]json.RawMessage
	decodeData(t, w.Body.Bytes(), &obj)
	if string(obj["facilities"]) != "[]" {
		t.Errorf(`want "facilities": [], got %q`, obj["facilities"])
	}
}
