package service

import (
	"testing"
	"time"

	"game_lounge_be/config"
	"game_lounge_be/models"

	"github.com/google/uuid"
)

// newCustBooking membuat booking milik customerID dengan status DB tertentu.
// Store acak → jam operasional fallback 10:00–02:00.
func newCustBooking(t *testing.T, customerID string, date time.Time, dbStatus string) *models.Booking {
	t.Helper()
	b := &models.Booking{
		ID: uuid.NewString(), BookingCode: "T" + uuid.NewString()[:12], StoreID: uuid.NewString(),
		RoomID: uuid.NewString(), CustomerID: &customerID, CustomerName: "TEST", BookingDate: date,
		StartTime: "12:00", EndTime: "13:00", DurationHours: 1, BasePrice: 1, TotalPrice: 1, Status: dbStatus,
	}
	if err := config.DB.Create(b).Error; err != nil {
		t.Fatalf("create fixture: %v", err)
	}
	t.Cleanup(func() { config.DB.Delete(&models.Booking{}, "id = ?", b.ID) })
	return b
}

func TestGetCustomerBookings_FilterStatusDanPagination(t *testing.T) {
	skipIfNoDB(t)
	cust := uuid.NewString()
	future := time.Now().AddDate(0, 0, 7)
	newCustBooking(t, cust, future, "upcoming")
	newCustBooking(t, cust, future.AddDate(0, 0, 1), "upcoming")
	newCustBooking(t, cust, future, "cancelled")

	got, total, err := GetCustomerBookings(cust, []string{"upcoming"}, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 || len(got) != 1 {
		t.Fatalf("want total=2 len=1, got total=%d len=%d", total, len(got))
	}
	if got[0].Status != "upcoming" {
		t.Errorf("want upcoming, got %s", got[0].Status)
	}
}

// Status di DB hanya disinkronkan saat admin membuka dashboard, jadi booking
// kemarin bisa masih tercatat "upcoming". Filter harus memakai status sebenarnya.
func TestGetCustomerBookings_StatusBasiDisinkronkan(t *testing.T) {
	skipIfNoDB(t)
	cust := uuid.NewString()
	past := newCustBooking(t, cust, time.Now().AddDate(0, 0, -2), "upcoming")

	upcoming, total, err := GetCustomerBookings(cust, []string{"upcoming"}, 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if total != 0 || len(upcoming) != 0 {
		t.Errorf("booking lampau tidak boleh muncul di upcoming, got %d", total)
	}

	done, _, err := GetCustomerBookings(cust, []string{"completed"}, 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(done) != 1 || done[0].ID != past.ID || done[0].Status != "completed" {
		t.Errorf("want booking lampau sebagai completed, got %+v", done)
	}
}

func TestGetCustomerBookings_TanpaFilterSemuaStatus(t *testing.T) {
	skipIfNoDB(t)
	cust := uuid.NewString()
	newCustBooking(t, cust, time.Now().AddDate(0, 0, 3), "upcoming")
	newCustBooking(t, cust, time.Now().AddDate(0, 0, 3), "cancelled")

	_, total, err := GetCustomerBookings(cust, nil, 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 {
		t.Errorf("want 2, got %d", total)
	}
}

func TestGetCustomerBookingByID_StatusDihitung(t *testing.T) {
	skipIfNoDB(t)
	cust := uuid.NewString()
	past := newCustBooking(t, cust, time.Now().AddDate(0, 0, -2), "upcoming")

	got, err := GetCustomerBookingByID(past.ID, cust)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "completed" {
		t.Errorf("want completed, got %s", got.Status)
	}
}
