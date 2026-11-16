package service

import (
	"testing"

	"game_lounge_be/config"
	"game_lounge_be/models"
	"game_lounge_be/modules/store/dto"
	"game_lounge_be/modules/store/repository"
)

func TestDeleteStore_RemovesItsRoomsFromStats(t *testing.T) {
	skipIfNoDB(t)
	tmplID := newTemplate(t, true)
	s, err := CreateStore(createReq(t, dto.RoomSetupInput{RoomTemplateID: tmplID, UnitCount: 3}), "test")
	if err != nil {
		t.Fatal(err)
	}
	before := repository.GetStoreStats()["total_rooms"]

	if err := DeleteStore(s.ID, "test"); err != nil {
		t.Fatal(err)
	}

	var live int64
	config.DB.Model(&models.StoreRoom{}).Where("store_id = ? AND deleted_at IS NULL", s.ID).Count(&live)
	if live != 0 {
		t.Errorf("rooms of a deleted store must be soft-deleted, %d still live", live)
	}
	var active int64
	config.DB.Model(&models.StoreRoom{}).Where("store_id = ? AND is_active = true", s.ID).Count(&active)
	if active != 0 {
		t.Errorf("rooms of a deleted store must be inactive, %d still active", active)
	}
	if after := repository.GetStoreStats()["total_rooms"]; after != before-3 {
		t.Errorf("total_rooms: want %d, got %d", before-3, after)
	}
}
