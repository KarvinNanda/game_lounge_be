package repository

import (
	"testing"
	"time"

	"game_lounge_be/config"
	"game_lounge_be/models"

	"github.com/google/uuid"
)

// Fixture tahun 2099 + store_id acak supaya tidak menyentuh data dev.
func newTestEvent(t *testing.T, storeID string, paymentStatus *string, createdAt time.Time) *models.EventBooking {
	t.Helper()
	e := &models.EventBooking{
		ID: uuid.NewString(), StoreID: storeID, EventName: "TEST event", CustomerName: "TEST",
		BookingDate: time.Date(2099, 5, 5, 0, 0, 0, 0, time.Local), StartTime: "12:00", EndTime: "16:00",
		DurationHours: 4, PricePerDay: 1, TotalPrice: 1, Status: "upcoming",
		PaymentStatus: paymentStatus, CreatedAt: createdAt,
	}
	if err := config.DB.Create(e).Error; err != nil {
		t.Fatalf("create fixture: %v", err)
	}
	t.Cleanup(func() { config.DB.Delete(&models.EventBooking{}, "id = ?", e.ID) })
	return e
}

func strp(s string) *string { return &s }

func TestCheckOverlapWithEvent_JendelaPembayaran(t *testing.T) {
	skipIfNoDB(t)
	cases := []struct {
		name    string
		status  *string
		created time.Time
		want    bool
	}{
		{"event admin (tanpa payment_status) memblokir", nil, time.Now(), true},
		{"customer sudah bayar memblokir", strp("paid"), time.Now().Add(-2 * time.Hour), true},
		{"customer belum bayar, masih dalam jendela → memblokir", strp("pending_payment"), time.Now(), true},
		{"customer belum bayar, jendela habis → TIDAK memblokir", strp("pending_payment"), time.Now().Add(-EventPaymentWindow - time.Minute), false},
		{"pembayaran gagal → TIDAK memblokir", strp("failed"), time.Now(), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			storeID := uuid.NewString()
			newTestEvent(t, storeID, tc.status, tc.created)
			got, err := CheckOverlapWithEvent(storeID, "2099-05-05", "13:00", "14:00", "")
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Errorf("want %v, got %v", tc.want, got)
			}
		})
	}
}

func TestCancelExpiredUnpaid(t *testing.T) {
	skipIfNoDB(t)
	storeID := uuid.NewString()
	expired := newTestEvent(t, storeID, strp("pending_payment"), time.Now().Add(-EventPaymentWindow-time.Minute))
	fresh := newTestEvent(t, storeID, strp("pending_payment"), time.Now())
	paid := newTestEvent(t, storeID, strp("paid"), time.Now().Add(-2*time.Hour))

	if _, err := CancelExpiredUnpaid(); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		e    *models.EventBooking
		want string
	}{{expired, "cancelled"}, {fresh, "upcoming"}, {paid, "upcoming"}} {
		var got models.EventBooking
		config.DB.First(&got, "id = ?", c.e.ID)
		if got.Status != c.want {
			t.Errorf("event %s: want %s, got %s", *c.e.PaymentStatus, c.want, got.Status)
		}
	}
}
