package service

import (
	"testing"

	"game_lounge_be/config"
	"game_lounge_be/models"
	"game_lounge_be/modules/pricing/dto"

	"github.com/google/uuid"
)

// store_pricings.store_id UNIQUE + soft delete: setelah pricing dihapus, membuat
// pricing baru untuk store yang sama dulu gagal dengan error 1062.
func TestCreatePricing_SetelahDihapus_BisaDibuatLagi(t *testing.T) {
	skipIfNoDB(t)
	storeID := uuid.NewString()
	t.Cleanup(func() { config.DB.Unscoped().Delete(&models.StorePricing{}, "store_id = ?", storeID) })

	if _, err := CreatePricing(dto.CreatePricingRequest{StoreID: storeID, IsHappyHourEnabled: true}, "test"); err != nil {
		t.Fatalf("create pertama: %v", err)
	}
	if err := DeletePricing(storeID, "test"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	p, err := CreatePricing(dto.CreatePricingRequest{StoreID: storeID, IsHappyHourEnabled: false, EdgeCase2h: "force_3h"}, "test2")
	if err != nil {
		t.Fatalf("create ulang setelah delete harus berhasil, got %v", err)
	}
	if p.IsHappyHourEnabled || p.EdgeCase2h != "force_3h" || p.DeletedAt != nil {
		t.Errorf("pricing harus aktif dengan nilai baru, got hh=%v edge=%s deleted=%v", p.IsHappyHourEnabled, p.EdgeCase2h, p.DeletedAt)
	}
	if _, err := CreatePricing(dto.CreatePricingRequest{StoreID: storeID}, "test"); err == nil {
		t.Error("pricing aktif yang sudah ada tetap tidak boleh dibuat dobel")
	}
}
