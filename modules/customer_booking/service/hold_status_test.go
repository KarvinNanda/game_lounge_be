package service

import (
	"errors"
	"testing"
	"time"

	"game_lounge_be/config"
	"game_lounge_be/models"

	"github.com/google/uuid"
)

// newTestHold: customer tanpa email (konfirmasi tidak mengirim email) + hold miliknya.
func newTestHold(t *testing.T, expiresAt time.Time) *models.BookingHold {
	t.Helper()
	cust := &models.Customer{ID: uuid.NewString(), Name: "TEST", Whatsapp: "0"}
	if err := config.DB.Create(cust).Error; err != nil {
		t.Fatal(err)
	}
	invoice := "TEST-" + uuid.NewString()
	h := &models.BookingHold{
		ID: uuid.NewString(), CustomerID: cust.ID, StoreID: uuid.NewString(), RoomID: uuid.NewString(),
		RoomTemplateID: 1, BookingDate: time.Now().AddDate(0, 0, 3), StartTime: "12:00", EndTime: "13:00",
		DurationHours: 1, BasePrice: 1, TotalPrice: 1, PriceBreakdown: "{}", XenditInvoiceID: &invoice,
		ExpiresAt: expiresAt,
	}
	if err := config.DB.Create(h).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		config.DB.Delete(&models.BookingHold{}, "id = ?", h.ID)
		config.DB.Delete(&models.Booking{}, "customer_id = ?", cust.ID)
		config.DB.Unscoped().Delete(&models.Customer{}, "id = ?", cust.ID)
	})
	return h
}

func TestConfirmBookingFromWebhook_MenyimpanHoldID(t *testing.T) {
	skipIfNoDB(t)
	h := newTestHold(t, time.Now().Add(10*time.Minute))
	if err := ConfirmBookingFromWebhook(*h.XenditInvoiceID, "TEST"); err != nil {
		t.Fatal(err)
	}
	var b models.Booking
	if err := config.DB.Where("hold_id = ?", h.ID).First(&b).Error; err != nil {
		t.Fatalf("booking dengan hold_id tidak ditemukan: %v", err)
	}
}

func TestGetHoldStatus(t *testing.T) {
	skipIfNoDB(t)

	t.Run("pending: hold masih berlaku", func(t *testing.T) {
		h := newTestHold(t, time.Now().Add(10*time.Minute))
		got, err := GetHoldStatus(h.ID, h.CustomerID)
		if err != nil || got.Status != "pending" || got.BookingCode != nil {
			t.Fatalf("got %+v, err %v", got, err)
		}
	})

	t.Run("expired: hold lewat expires_at", func(t *testing.T) {
		h := newTestHold(t, time.Now().Add(-time.Minute))
		got, err := GetHoldStatus(h.ID, h.CustomerID)
		if err != nil || got.Status != "expired" {
			t.Fatalf("got %+v, err %v", got, err)
		}
	})

	t.Run("confirmed: webhook sudah membuat booking", func(t *testing.T) {
		h := newTestHold(t, time.Now().Add(10*time.Minute))
		if err := ConfirmBookingFromWebhook(*h.XenditInvoiceID, "TEST"); err != nil {
			t.Fatal(err)
		}
		got, err := GetHoldStatus(h.ID, h.CustomerID)
		if err != nil || got.Status != "confirmed" || got.BookingCode == nil || got.BookingID == nil {
			t.Fatalf("got %+v, err %v", got, err)
		}
		var b models.Booking
		config.DB.First(&b, "id = ?", *got.BookingID)
		if b.BookingCode != *got.BookingCode || b.CustomerID == nil || *b.CustomerID != h.CustomerID {
			t.Errorf("booking tidak cocok: %+v", b)
		}
	})

	t.Run("hold milik customer lain → not found", func(t *testing.T) {
		h := newTestHold(t, time.Now().Add(10*time.Minute))
		if _, err := GetHoldStatus(h.ID, uuid.NewString()); !errors.Is(err, ErrHoldNotFound) {
			t.Fatalf("want ErrHoldNotFound, got %v", err)
		}
	})

	t.Run("booking milik customer lain → not found", func(t *testing.T) {
		h := newTestHold(t, time.Now().Add(10*time.Minute))
		if err := ConfirmBookingFromWebhook(*h.XenditInvoiceID, "TEST"); err != nil {
			t.Fatal(err)
		}
		if _, err := GetHoldStatus(h.ID, uuid.NewString()); !errors.Is(err, ErrHoldNotFound) {
			t.Fatalf("want ErrHoldNotFound, got %v", err)
		}
	})

	t.Run("id tidak dikenal → not found", func(t *testing.T) {
		if _, err := GetHoldStatus(uuid.NewString(), uuid.NewString()); !errors.Is(err, ErrHoldNotFound) {
			t.Fatalf("want ErrHoldNotFound, got %v", err)
		}
	})
}
