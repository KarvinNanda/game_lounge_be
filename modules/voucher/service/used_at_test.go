package service

import (
	"testing"
	"time"

	"game_lounge_be/config"
	"game_lounge_be/models"

	"github.com/google/uuid"
)

// used_at must come from the app clock like every other timestamp. The DB
// default CURRENT_TIMESTAMP uses the DB server timezone (UTC in the local
// container), which put reservations 7 hours away from the holds they belong to.
func TestVoucherUsage_UsedAtUsesAppClock(t *testing.T) {
	skipIfNoDB(t)
	u := &models.VoucherUsage{VoucherID: uuid.NewString(), CustomerID: uuid.NewString()}
	if err := config.DB.Create(u).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { config.DB.Delete(&models.VoucherUsage{}, u.ID) })

	var got models.VoucherUsage
	config.DB.First(&got, u.ID)
	if d := time.Since(got.UsedAt); d < -time.Minute || d > time.Minute {
		t.Errorf("used_at %v is %v away from now", got.UsedAt, d)
	}
}
