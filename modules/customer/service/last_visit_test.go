package service

import (
	"encoding/json"
	"testing"
	"time"

	"game_lounge_be/config"
	"game_lounge_be/models"

	"github.com/google/uuid"
)

func visitBooking(t *testing.T, customerID string, day int, start, status string) {
	t.Helper()
	b := &models.Booking{
		ID: uuid.NewString(), BookingCode: "T" + uuid.NewString()[:12], StoreID: uuid.NewString(),
		RoomID: uuid.NewString(), CustomerID: &customerID, CustomerName: "TEST", BookingDate: time.Now().AddDate(0, 0, day),
		StartTime: start, EndTime: "23:00", DurationHours: 1, BasePrice: 1, TotalPrice: 1, Status: status, PaymentMethod: "cash",
	}
	if err := config.DB.Create(b).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { config.DB.Delete(&models.Booking{}, "id = ?", b.ID) })
}

func newStatsCustomer(t *testing.T) string {
	t.Helper()
	c := &models.Customer{ID: uuid.NewString(), Name: "TEST Visit", Whatsapp: "0895" + uuid.NewString()[:8], Type: "member", Status: "active"}
	if err := config.DB.Create(c).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { config.DB.Unscoped().Delete(&models.Customer{}, "id = ?", c.ID) })
	return c.ID
}

func lastVisit(t *testing.T, id string) (any, any) {
	t.Helper()
	c, err := GetCustomerByID(id)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(c)
	var m map[string]any
	json.Unmarshal(b, &m)
	return m["last_visit_date"], m["last_visit_time"]
}

// Same definition as total_visit: the latest non-cancelled booking that is
// completed or dated before today. Cancelled and future bookings do not count.
func TestCustomerLastVisit(t *testing.T) {
	skipIfNoDB(t)
	id := newStatsCustomer(t)
	visitBooking(t, id, -3, "10:00", "completed")
	visitBooking(t, id, -1, "18:00", "upcoming") // stale status, day has passed: a visit
	visitBooking(t, id, -1, "09:00", "completed")
	visitBooking(t, id, -1, "21:00", "cancelled") // later, but cancelled
	visitBooking(t, id, 1, "09:00", "upcoming")   // future

	wantDate := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	if d, tm := lastVisit(t, id); d != wantDate || tm != "18:00" {
		t.Errorf("last visit = %v %v, want %s 18:00", d, tm, wantDate)
	}
}

func TestCustomerLastVisit_NoneIsNull(t *testing.T) {
	skipIfNoDB(t)
	id := newStatsCustomer(t)
	visitBooking(t, id, 2, "09:00", "upcoming")
	if d, tm := lastVisit(t, id); d != nil || tm != nil {
		t.Errorf("no visit yet: want null, got %v %v", d, tm)
	}
}
