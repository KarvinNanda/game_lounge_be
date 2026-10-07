package repository

import (
	"testing"
	"time"

	"game_lounge_be/config"
	"game_lounge_be/models"

	"github.com/google/uuid"
)

// Store acak tanpa jam operasional → fallback 10:00–02:00 (OperatingWindow).
func newTestBooking(t *testing.T, storeID string, date time.Time, start, end string) {
	t.Helper()
	b := &models.Booking{
		ID: uuid.NewString(), BookingCode: "T" + uuid.NewString()[:12], StoreID: storeID,
		RoomID: uuid.NewString(), CustomerName: "TEST", BookingDate: date,
		StartTime: start, EndTime: end, DurationHours: 1, BasePrice: 1, TotalPrice: 1, Status: "upcoming",
	}
	if err := config.DB.Create(b).Error; err != nil {
		t.Fatalf("create booking fixture: %v", err)
	}
	t.Cleanup(func() { config.DB.Delete(&models.Booking{}, "id = ?", b.ID) })
}

func newTestEventAt(t *testing.T, storeID string, date time.Time, start, end string) {
	t.Helper()
	e := newTestEvent(t, storeID, nil, time.Now())
	config.DB.Model(e).Updates(map[string]interface{}{"booking_date": date, "start_time": start, "end_time": end})
}

var (
	day0 = time.Date(2099, 6, 10, 0, 0, 0, 0, time.Local)
	day1 = day0.AddDate(0, 0, 1)
)

func TestCheckOverlapWithRegular_LintasTengahMalam(t *testing.T) {
	skipIfNoDB(t)
	cases := []struct {
		name           string
		bDate          time.Time
		bStart, bEnd   string
		evStart, evEnd string
		want           bool
	}{
		{"booking setelah tengah malam di hari operasional yang sama", day0, "00:00", "01:00", "20:00", "02:00", true},
		{"booking lintas tengah malam vs event setelah tengah malam", day0, "23:00", "01:00", "00:00", "02:00", true},
		{"booking siang esok hari, event berakhir sebelum buka", day1, "10:00", "12:00", "20:00", "02:00", false},
		{"event melewati jam buka esok hari", day1, "10:00", "12:00", "20:00", "11:00", true},
		{"booking pagi tidak bentrok dengan event malam", day0, "10:00", "12:00", "20:00", "02:00", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			storeID := uuid.NewString()
			newTestBooking(t, storeID, tc.bDate, tc.bStart, tc.bEnd)
			got, err := CheckOverlapWithRegular(storeID, "2099-06-10", tc.evStart, tc.evEnd)
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Errorf("want %v, got %v", tc.want, got)
			}
		})
	}
}

func TestCheckOverlapWithEvent_LintasTengahMalam(t *testing.T) {
	skipIfNoDB(t)
	cases := []struct {
		name           string
		evDate         time.Time
		evStart, evEnd string
		qStart, qEnd   string
		want           bool
	}{
		{"booking setelah tengah malam vs event malam", day0, "20:00", "02:00", "00:00", "01:00", true},
		{"event setelah tengah malam vs booking lintas tengah malam", day0, "00:00", "02:00", "23:00", "01:00", true},
		{"event kemarin melewati jam buka hari ini", day0.AddDate(0, 0, -1), "20:00", "11:00", "10:00", "12:00", true},
		{"event malam tidak bentrok dengan booking siang", day0, "20:00", "02:00", "10:00", "12:00", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			storeID := uuid.NewString()
			newTestEventAt(t, storeID, tc.evDate, tc.evStart, tc.evEnd)
			got, err := CheckOverlapWithEvent(storeID, "2099-06-10", tc.qStart, tc.qEnd, "")
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Errorf("want %v, got %v", tc.want, got)
			}
		})
	}
}
