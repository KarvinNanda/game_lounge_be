package controller

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"game_lounge_be/config"
	"game_lounge_be/models"

	"github.com/google/uuid"
)

func TestGetMyBookingByHold(t *testing.T) {
	skipIfNoDB(t)
	h := &models.BookingHold{
		ID: uuid.NewString(), CustomerID: uuid.NewString(), StoreID: uuid.NewString(), RoomID: uuid.NewString(),
		RoomTemplateID: 1, BookingDate: time.Now().AddDate(0, 0, 3), StartTime: "12:00", EndTime: "13:00",
		PriceBreakdown: "{}", ExpiresAt: time.Now().Add(10 * time.Minute),
	}
	if err := config.DB.Create(h).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { config.DB.Delete(&models.BookingHold{}, "id = ?", h.ID) })

	own := serve(http.MethodGet, "/x", h.CustomerID, "", withParam("hold_id", h.ID, GetMyBookingByHold))
	var resp struct {
		Data struct {
			Status string `json:"status"`
		} `json:"data"`
	}
	json.Unmarshal(own.Body.Bytes(), &resp)
	if own.Code != http.StatusOK || resp.Data.Status != "pending" {
		t.Fatalf("pemilik: want 200 pending, got %d %s", own.Code, own.Body.String())
	}

	// Hold milik orang lain dan ID asal harus identik (tidak membocorkan keberadaan ID).
	other := serve(http.MethodGet, "/x", uuid.NewString(), "", withParam("hold_id", h.ID, GetMyBookingByHold))
	unknown := serve(http.MethodGet, "/x", h.CustomerID, "", withParam("hold_id", uuid.NewString(), GetMyBookingByHold))
	if other.Code != http.StatusNotFound || unknown.Code != http.StatusNotFound {
		t.Fatalf("want 404/404, got %d/%d", other.Code, unknown.Code)
	}
	if other.Body.String() != unknown.Body.String() {
		t.Errorf("body berbeda: %s vs %s", other.Body.String(), unknown.Body.String())
	}
}
