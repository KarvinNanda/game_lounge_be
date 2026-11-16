package service

import (
	"errors"
	"testing"

	"game_lounge_be/config"
	"game_lounge_be/models"
	"game_lounge_be/modules/store/dto"

	"github.com/google/uuid"
)

func newTemplate(t *testing.T, active bool) uint {
	t.Helper()
	tmpl := models.RoomTemplate{Name: "TEST Template " + uuid.NewString()[:6], CapacityMax: 4, IsActive: active}
	if err := config.DB.Create(&tmpl).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { config.DB.Unscoped().Delete(&models.RoomTemplate{}, "id = ?", tmpl.ID) })
	return tmpl.ID
}

// createReq uses a unique store name so the test can check whether a store was inserted.
func createReq(t *testing.T, rooms ...dto.RoomSetupInput) dto.CreateStoreRequest {
	t.Helper()
	name := "TEST Create " + uuid.NewString()[:8]
	t.Cleanup(func() {
		var ids []string
		config.DB.Model(&models.Store{}).Where("name = ?", name).Pluck("id", &ids)
		for _, id := range ids {
			config.DB.Unscoped().Where("store_id = ?", id).Delete(&models.StoreRoom{})
			config.DB.Unscoped().Delete(&models.Store{}, "id = ?", id)
		}
	})
	return dto.CreateStoreRequest{Name: name, Address: "x", Status: "inactive", Rooms: rooms}
}

func storeCount(name string) int64 {
	var n int64
	config.DB.Model(&models.Store{}).Where("name = ?", name).Count(&n)
	return n
}

func TestCreateStore_CreatesUnitsForActiveType(t *testing.T) {
	skipIfNoDB(t)
	tmplID := newTemplate(t, true)
	s, err := CreateStore(createReq(t, dto.RoomSetupInput{RoomTemplateID: tmplID, UnitCount: 2}), "test")
	if err != nil {
		t.Fatal(err)
	}
	units := unitsOf(t, s.ID)
	if len(units) != 2 || units[0].UnitNumber != 1 || units[1].UnitNumber != 2 {
		t.Errorf("want units 1 and 2, got %+v", units)
	}
}

func TestCreateStore_RejectsInvalidRoomsBeforeCreatingStore(t *testing.T) {
	skipIfNoDB(t)
	inactive := newTemplate(t, false)
	cases := map[string]dto.RoomSetupInput{
		"inactive type":  {RoomTemplateID: inactive, UnitCount: 1},
		"unknown type":   {RoomTemplateID: 999999999, UnitCount: 1},
		"negative count": {RoomTemplateID: newTemplate(t, true), UnitCount: -1},
	}
	for name, room := range cases {
		t.Run(name, func(t *testing.T) {
			req := createReq(t, room)
			_, err := CreateStore(req, "test")
			if !errors.Is(err, ErrRoomSetup) {
				t.Fatalf("want ErrRoomSetup (400), got %v", err)
			}
			if n := storeCount(req.Name); n != 0 {
				t.Errorf("rejected request must not create the store, found %d", n)
			}
		})
	}
}
