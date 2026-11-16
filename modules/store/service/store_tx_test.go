package service

import (
	"testing"

	"game_lounge_be/config"
	"game_lounge_be/models"
	"game_lounge_be/modules/store/dto"
)

// A failing write in the middle (an invalid day_type rejected by the enum
// column) must roll back everything written before it.
func TestCreateStore_RollsBackWhenALaterWriteFails(t *testing.T) {
	skipIfNoDB(t)
	tmplID := newTemplate(t, true)
	req := createReq(t, dto.RoomSetupInput{RoomTemplateID: tmplID, UnitCount: 2})
	req.OperatingHours = []dto.OperatingHourInput{{DayType: "not-a-day", OpenTime: "10:00", CloseTime: "22:00", IsActive: true}}

	if _, err := CreateStore(req, "test"); err == nil {
		t.Fatal("invalid day_type must fail")
	}
	if n := storeCount(req.Name); n != 0 {
		t.Errorf("store must be rolled back, found %d", n)
	}
}

func TestUpdateStore_RollsBackWhenALaterWriteFails(t *testing.T) {
	skipIfNoDB(t)
	storeID, tmplID := newRoomsFixture(t)
	bad := updateReq("TEST Renamed", dto.RoomSetupInput{RoomTemplateID: tmplID, UnitCount: 2})
	bad.OperatingHours = []dto.OperatingHourInput{{DayType: "not-a-day", OpenTime: "10:00", CloseTime: "22:00", IsActive: true}}

	if _, err := UpdateStore(storeID, bad, "test"); err == nil {
		t.Fatal("invalid day_type must fail")
	}
	var s models.Store
	config.DB.First(&s, "id = ?", storeID)
	if s.Name != "TEST Rooms" {
		t.Errorf("store name must be rolled back, got %q", s.Name)
	}
	if n := len(unitsOf(t, storeID)); n != 0 {
		t.Errorf("new units must be rolled back, got %d", n)
	}
}
