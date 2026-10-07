package service

import (
	"errors"
	"sync"
	"testing"
	"time"

	"game_lounge_be/config"
	"game_lounge_be/models"
	"game_lounge_be/modules/event_booking/repository"

	"github.com/google/uuid"
)

func newPendingEvent(t *testing.T, createdAt time.Time, status string) *models.EventBooking {
	t.Helper()
	pending, inv := "pending_payment", "inv-test-"+uuid.NewString()
	e := &models.EventBooking{
		ID: uuid.NewString(), StoreID: uuid.NewString(), EventName: "TEST event", CustomerName: "TEST",
		BookingDate: time.Date(2099, 6, 6, 0, 0, 0, 0, time.Local), StartTime: "12:00", EndTime: "16:00",
		DurationHours: 4, PricePerDay: 1, TotalPrice: 1, Status: status,
		IsCustomerBooking: true, PaymentStatus: &pending, XenditInvoiceID: &inv, CreatedAt: createdAt,
	}
	if err := config.DB.Create(e).Error; err != nil {
		t.Fatalf("fixture: %v", err)
	}
	t.Cleanup(func() { config.DB.Delete(&models.EventBooking{}, "store_id = ?", e.StoreID) })
	return e
}

func reload(id string) models.EventBooking {
	var e models.EventBooking
	config.DB.First(&e, "id = ?", id)
	return e
}

func TestConfirmCustomerPayment_WebhookBersamaan(t *testing.T) {
	skipIfNoDB(t)
	e := newPendingEvent(t, time.Now(), "upcoming")

	var wg sync.WaitGroup
	results := make(chan error, 5)
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); results <- ConfirmCustomerPayment(*e.XenditInvoiceID) }()
	}
	wg.Wait()
	close(results)
	ok := 0
	for err := range results {
		if err == nil {
			ok++
		} else if !errors.Is(err, ErrEventPaymentProcessed) {
			t.Errorf("error tak terduga: %v", err)
		}
	}
	if ok != 1 {
		t.Errorf("tepat 1 webhook yang boleh memproses, got %d", ok)
	}
	if got := reload(e.ID); got.PaymentStatus == nil || *got.PaymentStatus != "paid" {
		t.Errorf("payment_status harus paid, got %v", got.PaymentStatus)
	}
}

func TestConfirmCustomerPayment_BayarSetelahDicancelOtomatis_SlotKosong(t *testing.T) {
	skipIfNoDB(t)
	e := newPendingEvent(t, time.Now().Add(-repository.EventPaymentWindow-time.Minute), "upcoming")
	if _, err := repository.CancelExpiredUnpaid(); err != nil {
		t.Fatal(err)
	}
	if err := ConfirmCustomerPayment(*e.XenditInvoiceID); err != nil {
		t.Fatalf("slot masih kosong → event harus dihidupkan lagi, got %v", err)
	}
	got := reload(e.ID)
	if got.Status != "upcoming" || *got.PaymentStatus != "paid" {
		t.Errorf("want upcoming/paid, got %s/%s", got.Status, *got.PaymentStatus)
	}
}

func TestConfirmCustomerPayment_BayarSetelahDicancel_SlotTerpakai(t *testing.T) {
	skipIfNoDB(t)
	e := newPendingEvent(t, time.Now().Add(-repository.EventPaymentWindow-time.Minute), "upcoming")
	if _, err := repository.CancelExpiredUnpaid(); err != nil {
		t.Fatal(err)
	}
	// Event lain mengambil slot yang sama di store yang sama.
	other := &models.EventBooking{
		ID: uuid.NewString(), StoreID: e.StoreID, EventName: "TEST other", CustomerName: "TEST",
		BookingDate: e.BookingDate, StartTime: "13:00", EndTime: "15:00", DurationHours: 2,
		PricePerDay: 1, TotalPrice: 1, Status: "upcoming",
	}
	config.DB.Create(other)

	if err := ConfirmCustomerPayment(*e.XenditInvoiceID); !errors.Is(err, ErrEventPaidButSlotTaken) {
		t.Fatalf("want ErrEventPaidButSlotTaken, got %v", err)
	}
	if got := reload(e.ID); got.Status != "cancelled" {
		t.Errorf("event tidak boleh dihidupkan lagi, got %s", got.Status)
	}
}
