package controller

import (
	"net/http"
	"testing"

	"game_lounge_be/config"
	"game_lounge_be/models"

	"github.com/google/uuid"
)

func TestQuoteEventBooking(t *testing.T) {
	skipIfNoDB(t)
	storeID := uuid.NewString()
	config.DB.Create(&models.StoreEventPrice{StoreID: storeID, PricePerDay: 1_000_000})
	t.Cleanup(func() { config.DB.Where("store_id = ?", storeID).Delete(&models.StoreEventPrice{}) })

	ok := serve(http.MethodGet, "/q?store_id="+storeID+"&booking_date=2099-07-01&start_time=20:00&end_time=21:00", "", "", QuoteEventBooking)
	if ok.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", ok.Code, ok.Body.String())
	}
	var q struct {
		TotalPrice float64 `json:"total_price"`
		Available  bool    `json:"available"`
	}
	decodeData(t, ok.Body.Bytes(), &q)
	if q.TotalPrice != 42_000 || !q.Available {
		t.Errorf("got %+v", q)
	}

	missing := serve(http.MethodGet, "/q?store_id="+storeID+"&booking_date=2099-07-01", "", "", QuoteEventBooking)
	if missing.Code != http.StatusBadRequest {
		t.Errorf("param kurang: want 400, got %d", missing.Code)
	}
}
