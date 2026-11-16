package service

import (
	"sort"
	"testing"
	"time"

	"game_lounge_be/config"
	"game_lounge_be/models"
	"game_lounge_be/modules/store/dto"

	"github.com/google/uuid"
)

// newRoomsFixture: an existing store without units, plus a room template.
func newRoomsFixture(t *testing.T) (storeID string, tmplID uint) {
	t.Helper()
	storeID = uuid.NewString()
	if err := config.DB.Create(&models.Store{ID: storeID, Name: "TEST Rooms", Address: "x", Status: "active"}).Error; err != nil {
		t.Fatal(err)
	}
	tmpl := models.RoomTemplate{Name: "TEST Template " + uuid.NewString()[:6], CapacityMax: 4, IsActive: true}
	if err := config.DB.Create(&tmpl).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		config.DB.Unscoped().Where("store_id = ?", storeID).Delete(&models.StoreRoom{})
		config.DB.Unscoped().Delete(&models.Store{}, "id = ?", storeID)
		config.DB.Unscoped().Delete(&models.RoomTemplate{}, "id = ?", tmpl.ID)
	})
	return storeID, tmpl.ID
}

func updateReq(name string, rooms ...dto.RoomSetupInput) dto.UpdateStoreRequest {
	return dto.UpdateStoreRequest{Name: name, Address: "x", Status: "active", Rooms: rooms}
}

func unitsOf(t *testing.T, storeID string) []models.StoreRoom {
	t.Helper()
	var rooms []models.StoreRoom
	config.DB.Where("store_id = ? AND deleted_at IS NULL", storeID).Order("unit_number").Find(&rooms)
	return rooms
}

func TestUpdateStore_AddsRoomsToExistingStore(t *testing.T) {
	skipIfNoDB(t)
	storeID, tmplID := newRoomsFixture(t)

	if _, err := UpdateStore(storeID, updateReq("TEST Rooms", dto.RoomSetupInput{RoomTemplateID: tmplID, UnitCount: 2}), "test"); err != nil {
		t.Fatal(err)
	}
	first := unitsOf(t, storeID)
	if len(first) != 2 || first[0].UnitNumber != 1 || first[1].UnitNumber != 2 || !first[0].IsActive {
		t.Fatalf("want active units 1 and 2, got %+v", first)
	}

	// Raise to 3: only unit 3 is new, units 1 and 2 keep their IDs (bookings point at them).
	if _, err := UpdateStore(storeID, updateReq("TEST Rooms", dto.RoomSetupInput{RoomTemplateID: tmplID, UnitCount: 3}), "test"); err != nil {
		t.Fatal(err)
	}
	second := unitsOf(t, storeID)
	if len(second) != 3 || second[2].UnitNumber != 3 {
		t.Fatalf("want 3 units, got %+v", second)
	}
	ids := []string{second[0].ID, second[1].ID}
	want := []string{first[0].ID, first[1].ID}
	sort.Strings(ids)
	sort.Strings(want)
	if ids[0] != want[0] || ids[1] != want[1] {
		t.Error("existing units must not be recreated")
	}

	// Same count again (e.g. the admin saves the form twice): nothing changes.
	if _, err := UpdateStore(storeID, updateReq("TEST Rooms", dto.RoomSetupInput{RoomTemplateID: tmplID, UnitCount: 3}), "test"); err != nil {
		t.Fatal(err)
	}
	if n := len(unitsOf(t, storeID)); n != 3 {
		t.Errorf("same count must be a no-op, got %d units", n)
	}
}

func TestUpdateStore_RejectsFewerUnitsWithoutSideEffects(t *testing.T) {
	skipIfNoDB(t)
	storeID, tmplID := newRoomsFixture(t)
	if _, err := UpdateStore(storeID, updateReq("TEST Rooms", dto.RoomSetupInput{RoomTemplateID: tmplID, UnitCount: 2}), "test"); err != nil {
		t.Fatal(err)
	}

	_, err := UpdateStore(storeID, updateReq("TEST Renamed", dto.RoomSetupInput{RoomTemplateID: tmplID, UnitCount: 1}), "test")
	if err == nil {
		t.Fatal("reducing units must be rejected (deactivate units with the toggle instead)")
	}
	if n := len(unitsOf(t, storeID)); n != 2 {
		t.Errorf("no unit may be removed, got %d", n)
	}
	var s models.Store
	config.DB.First(&s, "id = ?", storeID)
	if s.Name != "TEST Rooms" {
		t.Error("a rejected request must not change the store")
	}
}

func TestUpdateStore_RejectsUnknownTemplate(t *testing.T) {
	skipIfNoDB(t)
	storeID, _ := newRoomsFixture(t)
	if _, err := UpdateStore(storeID, updateReq("TEST Rooms", dto.RoomSetupInput{RoomTemplateID: 999999999, UnitCount: 1}), "test"); err == nil {
		t.Error("unknown room template must be rejected, not skipped silently")
	}
}

func TestUpdateStore_InactiveOrDeletedTemplate_NoNewUnits(t *testing.T) {
	skipIfNoDB(t)
	storeID, tmplID := newRoomsFixture(t)
	if _, err := UpdateStore(storeID, updateReq("TEST Rooms", dto.RoomSetupInput{RoomTemplateID: tmplID, UnitCount: 2}), "test"); err != nil {
		t.Fatal(err)
	}

	config.DB.Model(&models.RoomTemplate{}).Where("id = ?", tmplID).Update("is_active", false)

	// The admin form sends every type with its current count: that must still save.
	if _, err := UpdateStore(storeID, updateReq("TEST Rooms", dto.RoomSetupInput{RoomTemplateID: tmplID, UnitCount: 2}), "test"); err != nil {
		t.Fatalf("same count for an inactive type must be accepted: %v", err)
	}
	if _, err := UpdateStore(storeID, updateReq("TEST Rooms", dto.RoomSetupInput{RoomTemplateID: tmplID, UnitCount: 3}), "test"); err == nil {
		t.Error("adding units to an inactive room type must be rejected")
	}

	config.DB.Model(&models.RoomTemplate{}).Where("id = ?", tmplID).Updates(map[string]any{"is_active": true, "deleted_at": time.Now()})
	if _, err := UpdateStore(storeID, updateReq("TEST Rooms", dto.RoomSetupInput{RoomTemplateID: tmplID, UnitCount: 3}), "test"); err == nil {
		t.Error("adding units to a deleted room type must be rejected")
	}
	if n := len(unitsOf(t, storeID)); n != 2 {
		t.Errorf("want 2 units, got %d", n)
	}
}
